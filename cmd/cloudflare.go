package cmd

import (
	"errors"
	"fmt"
	"net"
	"strings"

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
	// ViperKeyCloudflareAPIToken is a viper key for the cloudflare api token.
	ViperKeyCloudflareAPIToken string = "cloudflare.api.token"
	// ViperKeyDNSDomain is a viper key for the DNS Domain.
	ViperKeyDNSDomain string = "dns.domain"
	// ViperKeyFQDNFilters is a viper key for the FQDN filters.
	ViperKeyFQDNFilters string = "fqdn.filters"
	// ViperKeyFQDNIgnoreFilters is a viper key for the FQDN ignore filters.
	ViperKeyFQDNIgnoreFilters string = "fqdn.ignore.filters"
)

func init() {
	flare.PersistentFlags().String(ViperKeyCloudflareAPIToken, "", "cloudflare api token")
	flare.PersistentFlags().String(ViperKeyDNSDomain, "", "DNS Domain")
	flare.PersistentFlags().StringSlice(ViperKeyFQDNFilters, []string{}, "FQDN filters")
	flare.PersistentFlags().StringSlice(ViperKeyFQDNIgnoreFilters, []string{}, "FQDN ignore filters")

	viper.BindPFlags(flare.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	rootCmd.AddCommand(flare)
}

func flareValidate() error {
	if len(viper.GetString(ViperKeyCloudflareAPIToken)) == 0 {
		return fmt.Errorf("config not set %s", ViperKeyCloudflareAPIToken)
	}
	if len(viper.GetString(ViperKeyDNSDomain)) == 0 {
		return fmt.Errorf("config not set %s", ViperKeyDNSDomain)
	}
	return nil
}

func updateDNSARecords(cIP net.IP) error {
	cfAPI, err := cloudflare.NewWithAPIToken(viper.GetString(ViperKeyCloudflareAPIToken))
	if err != nil {
		return err
	}

	id, err := cfAPI.ZoneIDByName(viper.GetString(ViperKeyDNSDomain))
	if err != nil {
		return err
	}

	records, err := cfAPI.DNSRecords(id, cloudflare.DNSRecord{})
	if err != nil {
		return err
	}

	fmt.Printf("%s : %v\n", ViperKeyFQDNFilters, viper.GetStringSlice(ViperKeyFQDNFilters))
	fmt.Printf("%s : %v\n", ViperKeyFQDNIgnoreFilters, viper.GetStringSlice(ViperKeyFQDNIgnoreFilters))
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

		if !matchFQDNFilter(viper.GetStringSlice(ViperKeyFQDNFilters), rec.Name, true) {
			fmt.Println("not filtered")
			continue
		}

		if matchFQDNFilter(viper.GetStringSlice(ViperKeyFQDNIgnoreFilters), rec.Name, false) {
			fmt.Println("ignore filtered")
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

func matchFQDNFilter(filters []string, fqdn string, emptyval bool) bool {
	if len(filters) == 0 {
		return emptyval
	}
	for _, filter := range filters {
		if strings.Contains(fqdn, filter) {
			return true
		}
	}

	return false
}
