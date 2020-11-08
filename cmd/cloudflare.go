package cmd

import (
	"errors"
	"fmt"
	"net"

	"github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flare = &cobra.Command{
		Use:   "flare [newIP]",
		Short: "Rewriting Cloudflare's DNS A records with a given IP",
		Long: `Rewriting Cloudflare's DNS A records with a given IP

[newIP]: New IP to set the A record for DNS

The environment variables must be set.
[CLOUDFLARE_API_TOKEN]: cloudflare api token
[DNS_DOMAIN]: DNS Domain
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("[newIP] is required")
			}
			return nil
		},
		Run: flareRun,
	}
	newIP net.IP
)

const (
	// EnvCloudflareAPIToken is a config key for the cloudflare api token.
	EnvCloudflareAPIToken string = "cloudflare_api_token"
	// EnvDNSDomain is a config key for the DNS Domain.
	EnvDNSDomain string = "dns_domain"
)

func init() {
	viper.AutomaticEnv()

	rootCmd.AddCommand(flare)
}

func flareValidate() error {
	if len(viper.GetString(EnvCloudflareAPIToken)) == 0 {
		return fmt.Errorf("config not set %s", EnvCloudflareAPIToken)
	}
	if len(viper.GetString(EnvDNSDomain)) == 0 {
		return fmt.Errorf("config not set %s", EnvDNSDomain)
	}
	return nil
}

func updateDNSARecords(cIP net.IP) error {
	cfAPI, err := cloudflare.NewWithAPIToken(viper.GetString(EnvCloudflareAPIToken))
	if err != nil {
		return err
	}

	id, err := cfAPI.ZoneIDByName(viper.GetString(EnvDNSDomain))
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

func flareRun(cmd *cobra.Command, args []string) {
	if err := flareValidate(); err != nil {
		panic(fmt.Errorf("Validation error : %s", err.Error()))
	}

	newIP := net.ParseIP(args[0])
	if newIP == nil {
		panic(fmt.Errorf("Failed ParseIP newIP=%s", args[0]))
	}

	if err := updateDNSARecords(newIP); err != nil {
		panic(err.Error())
	}
}
