package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

var (
	slackCmd = &cobra.Command{
		Use:   "slack [TEXT]",
		Short: "slackにポストする",
		Long:  `slackにポストする`,
		Run:   slackRun,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires TEXT")
			}
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(slackCmd)
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

func slackRun(cmd *cobra.Command, args []string) {

	text := args[0]

	slackWebhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	if len(slackWebhookURL) == 0 {
		panic(fmt.Errorf("environment not set SLACK_WEBHOOK_URL"))
	}
	msg := slack.WebhookMessage{
		Text: text,
	}
	err := slack.PostWebhook(slackWebhookURL, &msg)
	if err != nil {
		fmt.Println(err.Error())
	}
}
