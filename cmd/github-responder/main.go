/*
The github-responder command

*/
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"

	"github.com/hairyhenderson/github-responder/autotls"
	"github.com/pkg/errors"

	"github.com/hairyhenderson/github-responder"
	"github.com/hairyhenderson/github-responder/version"
	"github.com/satori/go.uuid"
	"github.com/spf13/cobra"
)

var (
	printVer bool
	verbose  bool
	opts     responder.Config
	repo     string
)

const (
	githubTokenName = "GITHUB_TOKEN"
)

func validateOpts(cmd *cobra.Command, args []string) error {
	if repo == "" {
		return errors.New("must provide repo")
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return errors.Errorf("invalid repo %s - need 'owner/repo' form", repo)
	}
	opts.Owner = parts[0]
	opts.Repo = parts[1]

	if opts.CallbackURL == "" {
		u := uuid.NewV4()
		callbackURL := ""
		if opts.EnableTLS {
			callbackURL = "https://"
		} else {
			callbackURL = "http://"
		}
		callbackURL = callbackURL + opts.Domain + "/gh-callback/" + u.String()
		opts.CallbackURL = callbackURL
	}

	if opts.HookSecret == "" {
		opts.HookSecret = fmt.Sprintf("%x", rand.Int63())
	}

	if opts.GitHubToken == "" {
		token := os.Getenv(githubTokenName)
		if token == "" {
			return errors.Errorf("GitHub API token missing - must set %s", githubTokenName)
		}

		opts.GitHubToken = token
	}

	return nil
}

func printVersion(name string) {
	fmt.Printf("%s version %s\n", name, version.Version)
}

func newCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "github-responder [ACTION]",
		Short: "Create and listen to GitHub WebHooks",
		Example: `  Run ./handle_event.sh every time a webhook event is received:

  $ github-responder -a -d example.com -e me@example.com ./handle_event.sh`,
		PreRunE: validateOpts,
		RunE: func(cmd *cobra.Command, args []string) error {
			if printVer {
				printVersion(cmd.Name())
				return nil
			}
			if verbose {
				// nolint: errcheck
				fmt.Fprintf(os.Stderr, "%s version %s, build %s (%v)\n\n",
					cmd.Name(), version.Version, version.GitCommit, version.BuildDate)
			}
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			var action responder.ActionFunc
			if len(args) > 0 {
				action = execArgs(args...)
			} else {
				logf("No action command given, will perform default")
				action = defaultAction
			}

			ctx := context.Background()
			logf("Starting responder with options %#v", opts)
			cleanup, err := responder.Start(ctx, opts, action)
			if err != nil {
				return err
			}
			logf("Responder started...")
			defer cleanup()

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)

			select {
			case s := <-c:
				logf("received %v, shutting down gracefully...", s)
			case <-ctx.Done():
				err = ctx.Err()
			}
			return err
		},
		Args: cobra.ArbitraryArgs,
	}
	return rootCmd
}

func logf(format string, v ...interface{}) {
	if verbose {
		log.Printf(format, v...)
	}
}

func initFlags(command *cobra.Command) {
	command.Flags().SortFlags = false

	command.Flags().StringVarP(&repo, "repo", "r", "", "The GitHub repository to watch, in 'owner/repo' form")
	command.Flags().StringVar(&opts.CallbackURL, "callback", "", "The WebHook Callback URL. If left blank, one will be generated for you.")
	command.Flags().StringArrayVarP(&opts.Events, "events", "e", []string{"*"}, "The GitHub event types to listen for. See https://developer.github.com/webhooks/#events for the full list.")

	command.Flags().StringVar(&opts.HTTPAddress, "http", ":80", "Address to listen to for HTTP traffic.")
	command.Flags().StringVar(&opts.TLSAddress, "https", ":443", "Address to listen to for TLS traffic.")

	command.Flags().BoolVar(&opts.EnableTLS, "tls", true, "Enable automatic TLS negotiation")
	command.Flags().StringVarP(&opts.Domain, "domain", "d", "", "domain to serve - a cert will be acquired for this domain")
	command.Flags().StringVarP(&opts.Email, "email", "m", "", "Email used for registration and recovery contact.")
	command.Flags().BoolVarP(&opts.Accept, "accept-tos", "a", false, "By setting this flag to true you indicate that you accept the current Let's Encrypt terms of service.")
	command.Flags().StringVar(&opts.CAEndpoint, "ca", autotls.LetsEncryptProductionURL, "URL to certificate authority's ACME server directory. Change this to point to a different server for testing.")
	command.Flags().StringVar(&opts.StoragePath, "path", "", "Directory to use for storing data")

	command.Flags().BoolVarP(&verbose, "verbose", "V", false, "Output extra logs")
	command.Flags().BoolVarP(&printVer, "version", "v", false, "Print the version")
}

func main() {
	command := newCmd()
	initFlags(command)
	if err := command.Execute(); err != nil {
		// nolint: errcheck
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
