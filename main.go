package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/slack-go/slack"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	cmName = "dns-tools"
)

var (
	slackWebhookURL    string
	cloudflareAPIToken string
	dnsDomain          string
	loopInterval       time.Duration = 5 * time.Second
)

func init() {
	slackWebhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	if len(slackWebhookURL) == 0 {
		panic(fmt.Errorf("environment not set SLACK_WEBHOOK_URL"))
	}

	cloudflareAPIToken = os.Getenv("CLOUDFLARE_API_TOKEN")
	if len(cloudflareAPIToken) == 0 {
		panic(fmt.Errorf("environment not set CLOUDFLARE_API_TOKEN"))
	}

	dnsDomain = os.Getenv("DNS_DOMAIN")
	if len(dnsDomain) == 0 {
		panic(fmt.Errorf("environment not set DNS_DOMAIN"))
	}

	if i, err := time.ParseDuration(os.Getenv("LOOP_INTERVAL")); err != nil {
		fmt.Println(err.Error())
	} else {
		loopInterval = i
	}
}

func getCurrentNamespace() (string, error) {
	if ns, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return ns, nil
	}

	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns, nil
		}
	}

	return "", errors.New("Failed current namespace")
}

func getPreviousGlobalIP(clientset *kubernetes.Clientset, ns string) (net.IP, error) {

	cmClient := clientset.CoreV1().ConfigMaps(ns)

	cm, err := cmClient.Get(context.TODO(), cmName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	strIP := cm.Data["globalIP"]
	objIP := net.ParseIP(strIP)
	if objIP == nil {
		return nil, fmt.Errorf("Failed ParseIP(%s)", strIP)
	}

	return objIP, nil
}

func getCurrentGlobalIP() (net.IP, error) {
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

func saveGlobalIP(clientset *kubernetes.Clientset, ns string, ip net.IP) error {

	cmClient := clientset.CoreV1().ConfigMaps(ns)

	cm, err := cmClient.Get(context.TODO(), cmName, metav1.GetOptions{})
	if err != nil {
		// configmapがなければ作成
		cm := &corev1.ConfigMap{}
		cm.Name = cmName
		cm.Data = map[string]string{
			"globalIP": ip.String(),
		}

		rcm, err := cmClient.Create(context.TODO(), cm, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		fmt.Printf("Created ConfigMap name: %s, data: %v.\n", rcm.Name, rcm.Data)
	} else {
		// configmapがあれば上書き
		fmt.Printf("Exists ConfigMap name: %s, data: %v.\n", cm.Name, cm.Data)

		cm.Data["globalIP"] = ip.String()

		rcm, err := cmClient.Update(context.TODO(), cm, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		fmt.Printf("Updated ConfigMap name: %s, data: %v.\n", rcm.Name, rcm.Data)
	}

	return nil
}

func postSlack(pIP net.IP, cIP net.IP) error {
	msg := slack.WebhookMessage{
		Text: fmt.Sprintf("Changed global ip : %s => %s\n", pIP.String(), cIP.String()),
	}

	err := slack.PostWebhook(slackWebhookURL, &msg)
	if err != nil {
		return err
	}
	return nil
}

func updateDNSRecords(cIP net.IP) error {
	cfAPI, err := cloudflare.NewWithAPIToken(cloudflareAPIToken)
	if err != nil {
		return err
	}

	id, err := cfAPI.ZoneIDByName(dnsDomain)
	if err != nil {
		return err
	}

	records, err := cfAPI.DNSRecords(id, cloudflare.DNSRecord{})
	if err != nil {
		return err
	}
	for _, rec := range records {
		fmt.Printf("%s %s %s \n", rec.Type, rec.Name, rec.Content)

		if rec.Type != "A" {
			fmt.Println("is not record type A")
			continue
		}

		pIP := net.ParseIP(rec.Content)
		if pIP == nil {
			return fmt.Errorf("Failed ParseIP(%s)", rec.Content)
		}

		if pIP.IsLoopback() {
			fmt.Printf("%s is loopback address\n", pIP.String())
			continue
		}

		if pIP.Equal(cIP) {
			fmt.Printf("%s is not change\n", pIP.String())
			continue
		}

		fmt.Printf("Updating %s => %s\n", pIP.String(), cIP.String())
		err := cfAPI.UpdateDNSRecord(id, rec.ID, cloudflare.DNSRecord{
			Content: cIP.String(),
			Proxied: rec.Proxied,
		})
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
	}

	return nil
}

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	cns, err := getCurrentNamespace()
	if err != nil {
		panic(err.Error())
	}

	for {
		time.Sleep(loopInterval)

		pIP, err := getPreviousGlobalIP(clientset, cns)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}

		cIP, err := getCurrentGlobalIP()
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			continue
		}
		fmt.Printf("previous IP: %s, current IP: %s\n", pIP.String(), cIP.String())

		if !pIP.Equal(cIP) {
			if err := postSlack(pIP, cIP); err != nil {
				fmt.Printf("%s\n", err.Error())
			}
			if err := saveGlobalIP(clientset, cns, cIP); err != nil {
				fmt.Printf("%s\n", err.Error())
			}
			if err := updateDNSRecords(cIP); err != nil {
				fmt.Printf("%s\n", err.Error())
			}
		}
	}
}
