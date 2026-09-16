package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/chill-institute/chill-cli/pkg/config"
	"github.com/chill-institute/chill-cli/pkg/rpc"
	"github.com/spf13/cobra"
)

func newDoctorCommand(app *appContext) *cobra.Command {
	var offline bool
	var fields string

	command := &cobra.Command{
		Use:   "doctor",
		Short: "Inspect CLI health, config, and auth status",
		Example: strings.TrimSpace(`
chilly doctor
chilly doctor --offline --output json
chilly doctor --fields auth.status,config.profile --output json
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}

			report, err := app.buildDoctorReport(cmd.Context(), offline)
			if err != nil {
				return err
			}
			return app.writeAnyWithRenderer(report, selection, renderDoctorPretty)
		},
	}

	command.Flags().BoolVar(&offline, "offline", false, "skip auth verification and inspect local state only")
	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func (app *appContext) buildDoctorReport(ctx context.Context, offline bool) (map[string]any, error) {
	store, err := app.configStore()
	if err != nil {
		return nil, err
	}

	configPath := store.Path()
	configExists := true
	if _, statErr := os.Stat(configPath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			configExists = false
		} else {
			return nil, wrapInternalError("doctor_config_stat_failed", "inspect config file", statErr)
		}
	}

	cfg, err := app.loadConfig()
	if err != nil {
		return nil, err
	}
	profile, err := app.activeProfile()
	if err != nil {
		return nil, err
	}

	info := currentBuildInfo()
	auth := map[string]any{
		"configured": strings.TrimSpace(cfg.AuthToken) != "",
	}
	if !auth["configured"].(bool) {
		auth["status"] = "missing"
	} else if offline {
		auth["status"] = "skipped"
	} else {
		status, details := app.verifyDoctorAuth(ctx, cfg)
		auth["status"] = status
		for key, value := range details {
			auth[key] = value
		}
	}

	report := map[string]any{
		"status": doctorOverallStatus(auth),
		"build": map[string]any{
			"name":       "chilly",
			"version":    info.Version,
			"commit":     info.Commit,
			"build_date": info.BuildDate,
			"dev":        info.IsDev(),
		},
		"config": map[string]any{
			"profile": profile,
			"path":    configPath,
			"exists":  configExists,
		},
		"api": map[string]any{
			"base_url": cfg.APIBaseURL,
		},
		"auth": auth,
	}
	return report, nil
}

func (app *appContext) verifyDoctorAuth(ctx context.Context, cfg config.Config) (string, map[string]any) {
	token, err := app.userToken(cfg)
	if err != nil {
		return "missing", map[string]any{}
	}

	response, err := app.callRPC(ctx, cfg, procedureUserGetUserProfile, map[string]any{}, rpc.AuthUser, token)
	if err != nil {
		classified := classifyError(err)
		details := map[string]any{
			"code":    classified.Code,
			"message": classified.Message,
			"kind":    string(classified.Kind),
		}
		if strings.TrimSpace(classified.RequestID) != "" {
			details["request_id"] = classified.RequestID
		}
		if classified.StatusCode > 0 {
			details["status_code"] = classified.StatusCode
		}
		if classified.Kind == errorKindAuth {
			return "invalid", details
		}
		return "error", details
	}

	payload := map[string]any{
		"request_id": response.RequestID,
	}
	var profile map[string]any
	if err := json.Unmarshal(response.Body, &profile); err == nil && profile != nil {
		user := map[string]any{}
		for _, key := range []string{"username", "email", "userId", "user_id", "plan"} {
			if value, ok := profile[key]; ok {
				user[key] = value
			}
		}
		if len(user) > 0 {
			payload["user"] = user
		}
	}
	return "ok", payload
}

func doctorOverallStatus(auth map[string]any) string {
	status, _ := auth["status"].(string)
	switch status {
	case "ok":
		return "ok"
	case "missing", "skipped", "invalid":
		return "warn"
	case "error":
		return "error"
	default:
		return "warn"
	}
}
