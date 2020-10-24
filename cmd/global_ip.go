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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

const (
	// GipConfigMapName is the name of the configmap resource used by gip
	GipConfigMapName = "dns-tools-gip"
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
	gipRun.Flags().BoolVar(&isNilfire, "nilfire", isNilfire, "Firing CloudEvents the last time a global IP could not be obtained. Defaults to false.")
	gipRun.Flags().DurationVar(&pollInterval, "interval", pollInterval, "Polling interval to check global IP addresses like 5s, 2m, or 3h. Defaults to 5s.")
	rootCmd.AddCommand(gipRun)
}

func getPreviousGlobalIPv4() (net.IP, error) {

	cm, err := cmClient.Get(context.TODO(), GipConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	strIP := cm.Data["globalIPv4"]
	objIP := net.ParseIP(strIP)
	if objIP == nil {
		return nil, fmt.Errorf("Failed ParseIP(%s)", strIP)
	}

	return objIP, nil
}

func getCurrentGlobalIPv4() (net.IP, error) {
	resp, err := http.Get("https://ifconfig.io/ip")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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
	if err != nil {
		// Create if there is no configmap
		cm := &corev1.ConfigMap{}
		cm.Name = GipConfigMapName
		cm.Data = map[string]string{
			"globalIPv4": ip.String(),
		}

		rcm, err := cmClient.Create(context.TODO(), cm, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		log.Printf("Created ConfigMap name: %s, data: %v.", rcm.Name, rcm.Data)
	} else {
		// If you have a configmap, overwrite it.
		log.Printf("Exists ConfigMap name: %s, data: %v.", cm.Name, cm.Data)

		cm.Data["globalIPv4"] = ip.String()

		rcm, err := cmClient.Update(context.TODO(), cm, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		log.Printf("Updated ConfigMap name: %s, data: %v.", rcm.Name, rcm.Data)
	}

	return nil
}

func postCloudEvent(obj interface{}) error {
	// The default client is HTTP.
	c, err := cloudevents.NewDefaultClient()
	if err != nil {
		return fmt.Errorf("failed to create client, %v", err)
	}

	log.Printf("%+v", obj)

	// Create an Event.
	event := cloudevents.NewEvent()
	event.SetSource("github.com/faruryo/dns-tools/cmd/postCloudEvent")
	event.SetType("github.com/faruryo/dns-tools/cmd/ChangeGlobalIP")
	err = event.SetData(
		cloudevents.ApplicationJSON,
		obj,
	)

	// Set a target.
	ctx := cloudevents.ContextWithTarget(context.TODO(), cloudEventsTarget)

	// Send that Event.
	result := c.Send(ctx, event)
	if cloudevents.IsUndelivered(result) {
		return fmt.Errorf("failed to send, %v", result)
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
		panic(err.Error())
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	cns, err := getCurrentNamespace()
	if err != nil {
		panic(err.Error())
	}

	cmClient = clientset.CoreV1().ConfigMaps(cns)
}

func globalIPRun(cmd *cobra.Command, args []string) {
	globalIPInit()

	for {
		time.Sleep(pollInterval)

		pIP, err := getPreviousGlobalIPv4()
		if err != nil {
			log.Printf("%s", err.Error())
		}

		cIP, err := getCurrentGlobalIPv4()
		if err != nil {
			log.Printf("%s", err.Error())
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
