package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newVersionCommand(app *appContext) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "version",
		Short: "Show CLI build info",
		Example: strings.TrimSpace(`
chilly version
chilly version --output json
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			info := currentBuildInfo()
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			if selection == nil && !wantsJSONOutput(app.opts.output) {
				_, err := fmt.Fprintln(app.stdout, formatVersionLine(info.Version, info.Commit))
				return wrapInternalError("stdout_write_failed", "write version output", err)
			}
			return app.writeAnyWithRenderer(map[string]any{
				"name":       "chilly",
				"version":    info.Version,
				"commit":     info.Commit,
				"build_date": info.BuildDate,
			}, selection, nil)
		},
	}
	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func formatVersionLine(version string, commit string) string {
	normalizedVersion := strings.TrimSpace(version)
	if normalizedVersion == "" {
		normalizedVersion = "dev"
	}

	normalizedCommit := strings.TrimSpace(commit)
	switch {
	case normalizedCommit == "", strings.EqualFold(normalizedCommit, "unknown"):
		return normalizedVersion
	case len(normalizedCommit) > 7:
		return fmt.Sprintf("%s (%s)", normalizedVersion, normalizedCommit[:7])
	default:
		return fmt.Sprintf("%s (%s)", normalizedVersion, normalizedCommit)
	}
}
