package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dns-tools",
	Short: "Tools available for DNS operation",
	Long:  `Tools available for DNS operation`,
}

// Execute executes the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getCurrentNamespace() (string, error) {
	const nsFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	if data, err := ioutil.ReadFile(nsFile); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns, nil
		}
	}

	return "", errors.New("Failed current namespace")
}
