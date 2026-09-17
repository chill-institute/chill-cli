package cli

import (
	"fmt"
	"strings"

	"github.com/chill-institute/chill-cli/v2/pkg/chill"
)

type userSettingsPatch = chill.UserSettingsPatch

func normalizeUserSettingsPatch(field string, value string) (userSettingsPatch, error) {
	patch, err := chill.NormalizeUserSettingsPatch(field, value)
	return patch, wrapValidationError(err)
}

func applyUserSettingsPatch(settings map[string]any, patch userSettingsPatch) map[string]any {
	return chill.ApplyUserSettingsPatch(settings, patch)
}

func supportedUserSettingsPatchInputs() []schemaInput {
	fields := chill.UserSettingsFields()
	inputs := make([]schemaInput, 0, len(fields))
	for _, field := range fields {
		inputs = append(inputs, schemaInput{
			Name:        fmt.Sprintf("field:%s", field.Name()),
			Type:        field.ValueType,
			Description: field.Description,
		})
	}
	return inputs
}

func supportedUserSettingsPatchHelp() string {
	fields := chill.UserSettingsFields()
	lines := make([]string, 0, len(fields)+1)
	lines = append(lines, "Supported patch fields:")
	for _, field := range fields {
		lines = append(lines, "  - "+field.String())
	}
	return strings.Join(lines, "\n")
}

func normalizeNullableNonNegativeInt64Value(raw string) (any, error) {
	value, err := chill.NormalizeNullableNonNegativeInt64Value(raw)
	return value, wrapValidationError(err)
}

func normalizeEnumValue(values map[string]string) func(string) (any, error) {
	inner := chill.NormalizeEnumValue(values)
	return func(raw string) (any, error) {
		value, err := inner(raw)
		return value, wrapValidationError(err)
	}
}

func cloneJSONObject(source map[string]any) map[string]any {
	return chill.CloneJSONObject(source)
}
