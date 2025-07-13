/*
The github-responder command
*/
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/caddyserver/certmagic"
	responder "github.com/hairyhenderson/github-responder"
	"github.com/hairyhenderson/github-responder/version"
	"github.com/spf13/cobra"
)

type config struct {
	domain   string
	repos    []string
	events   []string
	env      []string
	printVer bool
	verbose  bool
}

func printVersion(name string) {
	fmt.Printf("%s version %s\n", name, version.Version)
}

func newCmd(cfg *config) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "github-responder [ACTION]",
		Short: "Create and listen to GitHub WebHooks",
		Example: `  Run ./handle_event.sh every time a webhook event is received:

  $ github-responder -d example.com ./handle_event.sh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if cfg.verbose {
				// Set debug level for verbose mode
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}
			if cfg.printVer {
				printVersion(cmd.Name())

				return nil
			}

			slog.Debug(cmd.CalledAs(),
				"version", version.Version,
				"commit", version.GitCommit)

			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			var action responder.HookHandler
			if len(args) > 0 {
				action = execArgs(cfg.env, args...)
			} else {
				slog.Info("No action command given, will perform default")
				action = defaultAction
			}

			r, err := responder.New(cfg.repos, cfg.domain, action)
			if err != nil {
				return err
			}

			return r.RegisterAndListen(ctx, cfg.events)
		},
	}

	return rootCmd
}

func initFlags(cfg *config, command *cobra.Command) {
	command.Flags().SortFlags = false

	command.Flags().StringArrayVarP(
		&cfg.repos, "repo", "r", []string{},
		"The GitHub repository to watch, in 'owner/repo' form. Specify multiple times to watch many repos.",
	)
	command.Flags().StringArrayVarP(
		&cfg.events, "events", "e", []string{"*"},
		"The GitHub event type(s) to listen for. Specify multiple times to watch many events."+
			" See https://developer.github.com/webhooks/#events for the full list.",
	)

	command.Flags().IntVar(&certmagic.HTTPPort, "http", 80, "Port to listen on for HTTP traffic")
	command.Flags().IntVar(&certmagic.HTTPSPort, "https", 443, "Port to listen on for HTTPS traffic")

	command.Flags().StringVarP(&cfg.domain, "domain", "d", "", "domain to serve - a cert will be acquired for this domain")
	command.Flags().StringVarP(
		&certmagic.DefaultACME.Email, "email", "m", "",
		"Email used for registration and recovery contact (optional, but recommended)",
	)
	command.Flags().StringVar(
		&certmagic.DefaultACME.CA, "ca", certmagic.LetsEncryptProductionCA,
		"URL to certificate authority's ACME server directory. Change this to point to a different server for testing.",
	)

	command.Flags().StringArrayVar(
		&cfg.env, "env", []string{},
		"Set environment variables in KEY=value form. Omit =value to inherit current KEY value."+
			" By default, actions are executed with the parent environment.",
	)

	command.Flags().BoolVarP(&cfg.verbose, "verbose", "V", false, "Output extra logs")
	command.Flags().BoolVarP(&cfg.printVer, "version", "v", false, "Print the version")
}

func main() {
	initLogger()

	cfg := config{}

	command := newCmd(&cfg)
	initFlags(&cfg, command)

	if err := command.Execute(); err != nil {
		slog.Error(command.Name()+" failed", "error", err)
		os.Exit(1)
	}
}
