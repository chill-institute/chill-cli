package cli

import (
	"io"
	"strings"

	"github.com/chill-institute/chill-cli/pkg/config"
	"github.com/spf13/cobra"
)

var (
	resolveConfigProfile     = config.ResolveProfile
	resolveDefaultConfigPath = config.DefaultPath
)

func NewRootCommand() *cobra.Command {
	opts := &appOptions{output: outputPretty}
	app := newAppContext(opts)
	return newRootCommand(app)
}

func Run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return runCommand(args, stdin, stdout, stderr)
}

func newRootCommand(app *appContext) *cobra.Command {
	opts := app.opts
	command := &cobra.Command{
		Use:   "chilly",
		Short: "chill.institute CLI for humans and agents",
		Example: strings.TrimSpace(`
chilly auth login
chilly search --query "dune"
chilly movies
chilly schema command search --output json
`),
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(opts.configPath) == "" {
				resolvedProfile, err := resolveConfigProfile(opts.profile, currentBuildInfo().IsDev())
				if err != nil {
					return usageError("invalid_profile", "invalid --profile %q", opts.profile)
				}
				opts.profile = resolvedProfile

				defaultPath, err := resolveDefaultConfigPath(resolvedProfile)
				if err != nil {
					return wrapInternalError("resolve_config_path_failed", "resolve config path", err)
				}
				opts.configPath = defaultPath
			} else {
				resolvedProfile, err := resolveConfigProfile(opts.profile, currentBuildInfo().IsDev())
				if err != nil {
					return usageError("invalid_profile", "invalid --profile %q", opts.profile)
				}
				opts.profile = resolvedProfile
			}
			if trimmedAPIURL := strings.TrimSpace(opts.apiURL); trimmedAPIURL != "" {
				normalizedAPIURL, err := normalizeAPIBaseURL(trimmedAPIURL)
				if err != nil {
					return err
				}
				opts.apiURL = normalizedAPIURL
			}
			opts.outputSet = cmd.Flags().Changed("output")
			if !opts.outputSet && app.isTerminal != nil && !app.isTerminal(app.stdout) {
				opts.output = outputJSON
			}
			opts.output = strings.ToLower(strings.TrimSpace(opts.output))
			if opts.output != outputPretty && opts.output != outputJSON && opts.output != outputNDJSON {
				return usageError("invalid_output_mode", "invalid --output %q (expected: pretty|json|ndjson)", opts.output)
			}

			describe, err := cmd.Flags().GetBool("describe")
			if err != nil {
				return wrapInternalError("describe_flag_lookup_failed", "resolve describe flag", err)
			}
			if describe {
				if err := describeCommand(app, cmd); err != nil {
					return err
				}
				cmd.RunE = func(*cobra.Command, []string) error { return nil }
				cmd.Run = nil
			}
			return nil
		},
	}
	command.SetFlagErrorFunc(classifyFlagParseError)

	command.SetIn(app.stdin)
	command.SetOut(app.stdout)
	command.SetErr(app.stderr)
	command.PersistentFlags().StringVar(&opts.configPath, "config", "", "config file path")
	command.PersistentFlags().StringVar(&opts.profile, "profile", "", "config profile to use (default builds use default, dev builds use dev)")
	command.PersistentFlags().StringVar(&opts.apiURL, "api-url", "", "override API base URL")
	command.PersistentFlags().StringVar(&opts.output, "output", outputPretty, "output mode: pretty|json|ndjson")
	command.PersistentFlags().Bool("describe", false, "print command metadata and exit")

	command.AddCommand(newAuthCommand(app))
	command.AddCommand(newCompletionCommand(app))
	command.AddCommand(newDoctorCommand(app))
	command.AddCommand(newWhoamiCommand(app))
	command.AddCommand(newSettingsCommand(app))
	command.AddCommand(newSearchCommand(app))
	command.AddCommand(newMoviesCommand(app))
	command.AddCommand(newTVShowsCommand(app))
	command.AddCommand(newAddTransferCommand(app))
	command.AddCommand(newGetTransferCommand(app))
	command.AddCommand(newUserCommand(app))
	command.AddCommand(newSchemaCommand(app))
	command.AddCommand(newVersionCommand(app))
	command.AddCommand(newSelfUpdateCommand(app))

	return command
}

func classifyFlagParseError(cmd *cobra.Command, err error) error {
	if usageErr := negativeNumericPositionalUsageError(cmd, err); usageErr != nil {
		return usageErr
	}
	return err
}

func negativeNumericPositionalUsageError(cmd *cobra.Command, err error) error {
	rawArg, ok := negativeNumericShorthandArg(err)
	if cmd == nil || !ok {
		return nil
	}

	parsedArgCount := len(cmd.Flags().Args())
	switch schemaCommandID(cmd.CommandPath()) {
	case "get-transfer", "user transfer get":
		return validationErrorForPositionalArg(parsedArgCount, rawArg, validateTransferIDArg)
	case "user folder get", "user download-folder set":
		return validationErrorForPositionalArg(parsedArgCount, rawArg, validateFolderIDArg)
	case "tv-shows detail", "user tv-shows detail":
		return validationErrorForPositionalArg(parsedArgCount, rawArg, validateIMDbIDArg)
	case "tv-shows season", "tv-shows season-downloads", "user tv-shows season", "user tv-shows season-downloads":
		return validationErrorForPositionalArg(parsedArgCount, rawArg, validateIMDbIDArg, validateSeasonNumberArg)
	case "tv-shows episode-download", "user tv-shows episode-download":
		return validationErrorForPositionalArg(parsedArgCount, rawArg, validateIMDbIDArg, validateSeasonNumberArg, validateEpisodeNumberArg)
	default:
		return nil
	}
}

type positionalArgValidator func(string) error

func validationErrorForPositionalArg(index int, rawArg string, validators ...positionalArgValidator) error {
	if index < 0 || index >= len(validators) {
		return nil
	}
	return validators[index](rawArg)
}

func validateTransferIDArg(raw string) error {
	_, err := normalizeTransferID(raw)
	return err
}

func validateFolderIDArg(raw string) error {
	_, err := normalizeFolderID(raw)
	return err
}

func validateIMDbIDArg(raw string) error {
	_, err := normalizeIMDbID(raw)
	return err
}

func validateSeasonNumberArg(raw string) error {
	_, err := normalizeEpisodeOrdinal(raw, "season")
	return err
}

func validateEpisodeNumberArg(raw string) error {
	_, err := normalizeEpisodeOrdinal(raw, "episode")
	return err
}

func negativeNumericShorthandArg(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	message := strings.TrimSpace(err.Error())
	marker := " in -"
	index := strings.LastIndex(message, marker)
	if index < 0 {
		return "", false
	}
	value := strings.TrimSpace(message[index+len(marker):])
	if value == "" {
		return "", false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return "-" + value, true
}

func describeCommand(app *appContext, cmd *cobra.Command) error {
	entry, ok := lookupCommandSchema(schemaCommandID(cmd.CommandPath()))
	if !ok {
		return usageError("describe_not_supported", "describe is not supported for %q", cmd.CommandPath())
	}
	return app.writeJSONPayload(entry)
}
