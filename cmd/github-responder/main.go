/*
The github-responder command

*/
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mholt/certmagic"

	responder "github.com/hairyhenderson/github-responder"
	"github.com/hairyhenderson/github-responder/version"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	printVer bool
	verbose  bool
	repo     string
	events   []string
	domain   string
)

func printVersion(name string) {
	fmt.Printf("%s version %s\n", name, version.Version)
}

func newCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "github-responder [ACTION]",
		Short: "Create and listen to GitHub WebHooks",
		Example: `  Run ./handle_event.sh every time a webhook event is received:

  $ github-responder -a -d example.com -e me@example.com ./handle_event.sh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if verbose {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}
			if printVer {
				printVersion(cmd.Name())
				return nil
			}
			log.Debug().
				Str("version", version.Version).
				Str("commit", version.GitCommit).
				Msg(cmd.CalledAs())
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			var action responder.HookHandler
			if len(args) > 0 {
				action = execArgs(args...)
			} else {
				log.Info().Msg("No action command given, will perform default")
				action = defaultAction
			}

			ctx := context.Background()
			r, err := responder.New(repo, domain)
			if err != nil {
				return err
			}

			return r.RegisterAndListen(ctx, events, action)
		},
	}
	return rootCmd
}

func initFlags(command *cobra.Command) {
	command.Flags().SortFlags = false

	command.Flags().StringVarP(&repo, "repo", "r", "", "The GitHub repository to watch, in 'owner/repo' form")
	command.Flags().StringArrayVarP(&events, "events", "e", []string{"*"}, "The GitHub event types to listen for. See https://developer.github.com/webhooks/#events for the full list.")

	command.Flags().IntVar(&certmagic.HTTPPort, "http", 80, "Port to listen on for HTTP traffic")
	command.Flags().IntVar(&certmagic.HTTPSPort, "https", 443, "Port to listen on for HTTPS traffic")

	command.Flags().StringVarP(&domain, "domain", "d", "", "domain to serve - a cert will be acquired for this domain")
	command.Flags().StringVarP(&certmagic.Email, "email", "m", "", "Email used for registration and recovery contact (optional, but recommended)")
	command.Flags().StringVar(&certmagic.CA, "ca", certmagic.LetsEncryptProductionCA, "URL to certificate authority's ACME server directory. Change this to point to a different server for testing.")

	command.Flags().BoolVarP(&verbose, "verbose", "V", false, "Output extra logs")
	command.Flags().BoolVarP(&printVer, "version", "v", false, "Print the version")
}

func main() {
	initLogger()

	command := newCmd()
	initFlags(command)
	if err := command.Execute(); err != nil {
		log.Error().Err(err).Msg(command.Name() + " failed")
		os.Exit(1)
	}
}
