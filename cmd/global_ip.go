package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

const (
	// GipConfigMapName is the name of the configmap resource used by gip
	GipConfigMapName = "dns-tools-gip"
	getIPEndPoint    = "https://ifconfig.io/ip"
)

var (
	gipRun = &cobra.Command{
		Use:   "gip",
		Short: "Monitoring GlobalIPv4 and generating CloudEvents",
		Long: `Monitoring GlobalIPv4 and generating CloudEvents
		It is intended to be used on Knative Eventing.`,
		Run: globalIPRun,
	}
	pollInterval      time.Duration = 5 * time.Second
	isNilfire         bool          = false
	cloudEventsTarget string
	cmClient          v1.ConfigMapInterface
	clientset         *kubernetes.Clientset
)

func init() {
	gipRun.Flags().BoolVar(&isNilfire, "nilfire", isNilfire, "Fire CloudEvents when the previous global IP does not exist. Defaults to false.")
	gipRun.Flags().DurationVar(&pollInterval, "interval", pollInterval, "Polling interval to check global IP addresses like 5s, 2m, or 3h. Defaults to 5s.")
	rootCmd.AddCommand(gipRun)
}

func getPreviousGlobalIPv4() (net.IP, error) {

	cm, err := cmClient.Get(context.TODO(), GipConfigMapName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, err
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		return nil, fmt.Errorf("Error getting ConfigMap %s: ErrStatus=%v", GipConfigMapName, statusError.ErrStatus.Message)
	} else if err != nil {
		return nil, fmt.Errorf("Error getting ConfigMap %s: %w", GipConfigMapName, err)
	}

	strIP := cm.Data["globalIPv4"]
	objIP := net.ParseIP(strIP)
	if objIP == nil {
		return nil, fmt.Errorf("Failed ParseIP(%s)", strIP)
	}

	return objIP, nil
}

func getCurrentGlobalIPv4() (net.IP, error) {
	resp, err := http.Get(getIPEndPoint)
	if err != nil {
		return nil, fmt.Errorf("Failed GET (%s): %w", getIPEndPoint, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read Body: %w", err)
	}

	strIP := strings.TrimRight(string(body), "\r\n")
	objIP := net.ParseIP(strIP)
	if objIP == nil {
		return nil, fmt.Errorf("Failed ParseIP(%s)", strIP)
	}

	return objIP, nil
}

func saveGlobalIPv4(ip net.IP) error {

	cm, err := cmClient.Get(context.TODO(), GipConfigMapName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// Create if there is no configmap
		cm := &corev1.ConfigMap{}
		cm.Name = GipConfigMapName
		cm.Data = map[string]string{
			"globalIPv4": ip.String(),
		}

		rcm, err := cmClient.Create(context.TODO(), cm, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("Failed Create ConfigMap %v: %w", cm, err)
		}
		log.Printf("Created ConfigMap name: %s, data: %v.", rcm.Name, rcm.Data)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		return fmt.Errorf("Error getting ConfigMap %s: statusError=%v", GipConfigMapName, statusError.ErrStatus.Message)
	} else if err != nil {
		return fmt.Errorf("Error getting ConfigMap %s: %w", GipConfigMapName, err)
	} else {
		// If you have a configmap, overwrite it.
		log.Printf("Exists ConfigMap name: %s, data: %v.", cm.Name, cm.Data)

		cm.Data["globalIPv4"] = ip.String()

		rcm, err := cmClient.Update(context.TODO(), cm, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("Failed Update ConfigMap %v: %w", cm, err)
		}
		log.Printf("Updated ConfigMap name: %s, data: %v.", rcm.Name, rcm.Data)
	}

	return nil
}

func postCloudEvent(obj interface{}) error {
	// The default client is HTTP.
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		return fmt.Errorf("failed to create client, %w", err)
	}

	log.Printf("%+v", obj)

	// Create an Event.
	event := cloudevents.NewEvent()
	event.SetSource("github.com/faruryo/dns-tools/cmd/postCloudEvent")
	event.SetType("github.com/faruryo/dns-tools/cmd/ChangeGlobalIP")
	if err := event.SetData(
		cloudevents.ApplicationJSON,
		obj,
	); err != nil {
		return fmt.Errorf("Failed SetData, %v: %w", obj, err)
	}

	// Set a target.
	ctx := cloudevents.ContextWithTarget(context.TODO(), cloudEventsTarget)

	// Send that Event.
	result := c.Send(ctx, event)
	if cloudevents.IsUndelivered(result) {
		return fmt.Errorf("Failed to send, %v", result)
	}

	log.Printf("CloudEvents accepted: %t", cloudevents.IsACK(result))

	return nil
}

func globalIPInit() {
	cloudEventsTarget = os.Getenv("K_SINK")
	if len(cloudEventsTarget) == 0 {
		panic("environment variables not set K_SINK")
	}
	log.Printf("%s", cloudEventsTarget)

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(fmt.Errorf("Failed to retrieve cluster config: %w", err))
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(fmt.Errorf("Failed to retrieve Clientset: %w", err))
	}

	cns, err := getCurrentNamespace()
	if err != nil {
		panic(fmt.Errorf("Failed to retrieve Namespace: %w", err))
	}

	cmClient = clientset.CoreV1().ConfigMaps(cns)
}

func globalIPRun(cmd *cobra.Command, args []string) {
	globalIPInit()

	for {
		time.Sleep(pollInterval)

		pIP, err := getPreviousGlobalIPv4()
		if errors.IsNotFound(err) {
			log.Printf("Previous global IPv4 not found: %s", err)
		} else if err != nil {
			// Abort the process except for Notfound.
			log.Printf("Failed to get previous global IPv4: %s", err)
			continue
		}

		cIP, err := getCurrentGlobalIPv4()
		if err != nil {
			log.Printf("Failed to get current global IPv4: %s", err)
			continue
		}
		log.Printf("previous IP: %s, current IP: %s", pIP.String(), cIP.String())

		if !pIP.Equal(cIP) {
			if err := saveGlobalIPv4(cIP); err != nil {
				log.Printf("%s", err.Error())
			}

			if pIP != nil || isNilfire {
				if err := postCloudEvent(NewChangeGlobalIP(pIP, cIP)); err != nil {
					log.Printf("%s", err.Error())
				}
			}
		}
	}
}
