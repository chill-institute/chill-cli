package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestRootCommandIncludesSchemaTopLevelCommand(t *testing.T) {
	command := NewRootCommand()
	expected := map[string]bool{
		"add-transfer": true,
		"get-transfer": true,
		"auth":         true,
		"completion":   true,
		"doctor":       true,
		"movies":       true,
		"tv-shows":     true,
		"schema":       true,
		"search":       true,
		"self-update":  true,
		"settings":     true,
		"user":         true,
		"version":      true,
		"whoami":       true,
	}

	for _, subcommand := range command.Commands() {
		if _, ok := expected[subcommand.Name()]; ok {
			expected[subcommand.Name()] = false
		}
	}

	for name, missing := range expected {
		if missing {
			t.Fatalf("missing top-level command %q", name)
		}
	}
}

func TestSchemaCommandDoctorReturnsMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "doctor", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.ID != "doctor" {
		t.Fatalf("ID = %q, want %q", output.ID, "doctor")
	}
	if !output.SupportsFields {
		t.Fatal("doctor metadata should support fields")
	}
}

func TestCompletionCommandReturnsMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "completion", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.ID != "completion" {
		t.Fatalf("ID = %q, want %q", output.ID, "completion")
	}
	if output.Output.JSON {
		t.Fatal("completion metadata should not claim json output")
	}
}

func TestSchemaCommandUserDownloadFolderReturnsMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "user download-folder", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.LinkedProcedure != procedureUserGetDownloadFolder {
		t.Fatalf("LinkedProcedure = %q, want %q", output.LinkedProcedure, procedureUserGetDownloadFolder)
	}
}

func TestSchemaCommandUserDownloadFolderSetReturnsMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "user download-folder set", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !output.Mutates || !output.SupportsDryRun {
		t.Fatalf("metadata = %#v, want mutating dry-run command", output)
	}

	foundJSON := false
	for _, input := range output.Inputs {
		if input.Name == "json" {
			foundJSON = true
			break
		}
	}
	if !foundJSON {
		t.Fatalf("inputs = %#v, want json input", output.Inputs)
	}
}

func TestSchemaCommandUserIndexersReturnsFieldMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "user indexers", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !output.SupportsFields {
		t.Fatalf("metadata = %#v, want fields support", output)
	}
}

func TestSchemaCommandTVShowsReturnsSourceMetadata(t *testing.T) {
	t.Parallel()

	output := commandSchemaForTest(t, "tv-shows")

	if output.LinkedProcedure != procedureUserGetTVShows {
		t.Fatalf("LinkedProcedure = %q, want %q", output.LinkedProcedure, procedureUserGetTVShows)
	}
	assertInputNotUnconditionallyRequired(t, output, "source")
}

func TestSchemaCommandAddTransferReturnsInputModes(t *testing.T) {
	t.Parallel()

	output := commandSchemaForTest(t, "add-transfer")

	assertInputNotUnconditionallyRequired(t, output, "url")
	assertInputNotUnconditionallyRequired(t, output, "json")
	assertInputModes(t, output, []schemaInputMode{
		{
			Name:        "url",
			Description: "Build the request from --url.",
			Inputs:      []string{"url"},
			Required:    []string{"url"},
		},
		{
			Name:        "json",
			Description: "Use a raw JSON request body from --json.",
			Inputs:      []string{"json"},
			Required:    []string{"json"},
		},
	})
}

func TestSchemaCommandSettingsSetReturnsInputModes(t *testing.T) {
	t.Parallel()

	output := commandSchemaForTest(t, "settings set")

	assertInputNotUnconditionallyRequired(t, output, "key")
	assertInputNotUnconditionallyRequired(t, output, "value")
	assertInputNotUnconditionallyRequired(t, output, "json")
	assertInputModes(t, output, []schemaInputMode{
		{
			Name:        "key-value",
			Description: "Set one setting from positional key and value arguments.",
			Inputs:      []string{"key", "value"},
			Required:    []string{"key", "value"},
		},
		{
			Name:        "json",
			Description: "Set one setting from a raw JSON request body.",
			Inputs:      []string{"json"},
			Required:    []string{"json"},
		},
	})
}

func TestSchemaProcedureGetFolderReturnsMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "procedure", procedureUserGetFolder, "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.Kind != "procedure" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "procedure")
	}
	if len(output.Inputs) == 0 || output.Inputs[0].Name != "id" {
		t.Fatalf("Inputs = %#v, want folder id input", output.Inputs)
	}
}

func TestSchemaListsCommandsAndProceduresInJSON(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output struct {
		Commands   []schemaEntry `json:"commands"`
		Procedures []schemaEntry `json:"procedures"`
		Types      []schemaType  `json:"types"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(output.Commands) == 0 {
		t.Fatal("expected commands in schema output")
	}
	if len(output.Procedures) == 0 {
		t.Fatal("expected procedures in schema output")
	}
	if len(output.Types) == 0 {
		t.Fatal("expected types in schema output")
	}
	if output.Commands[0].ID != "add-transfer" {
		t.Fatalf("first command id = %q, want %q", output.Commands[0].ID, "add-transfer")
	}
}

func TestSchemaCommandSearchReturnsMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "search", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.ID != "search" {
		t.Fatalf("ID = %q, want %q", output.ID, "search")
	}
	if output.AuthMode != "user" {
		t.Fatalf("AuthMode = %q, want %q", output.AuthMode, "user")
	}
	if output.LinkedProcedure != procedureUserSearch {
		t.Fatalf("LinkedProcedure = %q, want %q", output.LinkedProcedure, procedureUserSearch)
	}
	if output.Output.Type != typeSearchResponse {
		t.Fatalf("Output.Type = %q, want %q", output.Output.Type, typeSearchResponse)
	}
	if output.Mutates {
		t.Fatal("search metadata unexpectedly marked mutating")
	}
	if output.SupportsDryRun {
		t.Fatal("search metadata unexpectedly supports dry run")
	}
	if !output.SupportsFields {
		t.Fatal("search metadata should support field selection")
	}
}

func TestSchemaProcedureSearchReturnsMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "procedure", procedureUserSearch, "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.Kind != "procedure" {
		t.Fatalf("Kind = %q, want %q", output.Kind, "procedure")
	}
	if output.AuthMode != "user" {
		t.Fatalf("AuthMode = %q, want %q", output.AuthMode, "user")
	}
	if output.Output.Type != typeSearchResponse {
		t.Fatalf("Output.Type = %q, want %q", output.Output.Type, typeSearchResponse)
	}
}

func TestSchemaProcedureTVShowsReturnsSourceInput(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "procedure", procedureUserGetTVShows, "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	assertInputNotUnconditionallyRequired(t, output, "source")
}

func TestSchemaTypeReleaseInfoReturnsJSONNames(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "type", typeReleaseInfo, "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaType
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.ID != typeReleaseInfo {
		t.Fatalf("ID = %q, want %q", output.ID, typeReleaseInfo)
	}

	foundBitDepth := false
	foundThreeD := false
	for _, field := range output.Fields {
		switch field.Name {
		case "bit_depth":
			if field.JSONName != "bitDepth" {
				t.Fatalf("bit_depth json_name = %q, want bitDepth", field.JSONName)
			}
			foundBitDepth = true
		case "three_d":
			if field.JSONName != "threeD" {
				t.Fatalf("three_d json_name = %q, want threeD", field.JSONName)
			}
			foundThreeD = true
		}
	}
	if !foundBitDepth || !foundThreeD {
		t.Fatalf("fields = %#v, want bit_depth and three_d", output.Fields)
	}
}

func TestSchemaTypeCanBeFieldFiltered(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "type", typeSearchResult, "--fields", "id,fields.name,fields.json_name", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output["id"] != typeSearchResult {
		t.Fatalf("id = %v, want %q", output["id"], typeSearchResult)
	}

	fields, ok := output["fields"].([]any)
	if !ok || len(fields) == 0 {
		t.Fatalf("fields = %#v, want populated fields", output["fields"])
	}

	foundReleaseInfo := false
	for _, rawField := range fields {
		field, ok := rawField.(map[string]any)
		if !ok {
			t.Fatalf("field = %#v, want object", rawField)
		}
		if _, ok := field["type"]; ok {
			t.Fatalf("field = %#v, did not expect type after field filtering", field)
		}
		if field["name"] == "release_info" && field["json_name"] == "releaseInfo" {
			foundReleaseInfo = true
		}
	}
	if !foundReleaseInfo {
		t.Fatalf("fields = %#v, want release_info/releaseInfo field", fields)
	}
}

func TestSchemaCommandSchemaTypeReturnsMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "schema type", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.ID != "schema type" {
		t.Fatalf("ID = %q, want schema type", output.ID)
	}
	if !output.SupportsFields {
		t.Fatal("schema type metadata should support fields")
	}
}

func TestSchemaTypeUnknownReturnsUsageError(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "type", "missing.Type", "--output", "json"})
	err := command.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown type schema") {
		t.Fatalf("error = %v, want unknown type schema", err)
	}
}

func TestSearchDescribeOutputsMetadataWithoutExecuting(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"search", "--describe", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.ID != "search" {
		t.Fatalf("ID = %q, want %q", output.ID, "search")
	}
}

func TestAuthLoginDescribeOutputsMetadataWithoutExecuting(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:    &appOptions{output: outputJSON},
		stdin:   strings.NewReader(""),
		stdout:  stdout,
		stderr:  &bytes.Buffer{},
		openURL: func(string) error { t.Fatal("openURL should not be called during describe"); return nil },
	})
	command.SetArgs([]string{"auth", "login", "--describe", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.ID != "auth login" {
		t.Fatalf("ID = %q, want %q", output.ID, "auth login")
	}
}

func TestSchemaCommandAddTransferReturnsDryRunMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "add-transfer", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !output.Mutates {
		t.Fatal("add-transfer metadata should be mutating")
	}
	if !output.SupportsDryRun {
		t.Fatal("add-transfer metadata should support dry run")
	}

	foundDryRun := false
	for _, input := range output.Inputs {
		if input.Name == "dry-run" {
			foundDryRun = true
			break
		}
	}
	if !foundDryRun {
		t.Fatal("add-transfer metadata missing dry-run input")
	}
}

func TestSchemaCommandGetTransferReturnsFieldMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "get-transfer", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.LinkedProcedure != procedureUserGetTransfer {
		t.Fatalf("LinkedProcedure = %q, want %q", output.LinkedProcedure, procedureUserGetTransfer)
	}
	if !output.SupportsFields {
		t.Fatal("get-transfer metadata should support fields")
	}
}

func TestSchemaCommandSettingsSetReturnsDryRunMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "settings set", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !output.Mutates {
		t.Fatal("settings set metadata should be mutating")
	}
	if !output.SupportsDryRun {
		t.Fatal("settings set metadata should support dry run")
	}

	foundDryRun := false
	for _, input := range output.Inputs {
		if input.Name == "dry-run" {
			foundDryRun = true
			break
		}
	}
	if !foundDryRun {
		t.Fatal("settings set metadata missing dry-run input")
	}
}

func TestSchemaCommandAuthLogoutReturnsDryRunMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "auth logout", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !output.SupportsDryRun {
		t.Fatal("auth logout metadata should support dry run")
	}
}

func TestSchemaCommandAuthLoginReturnsJSONInputMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "auth login", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	foundJSON := false
	foundDryRun := false
	for _, input := range output.Inputs {
		switch input.Name {
		case "json":
			foundJSON = true
		case "dry-run":
			foundDryRun = true
		}
	}
	if !foundJSON || !foundDryRun {
		t.Fatalf("inputs = %#v, want json and dry-run inputs", output.Inputs)
	}
}

func TestSchemaCommandUserSettingsSetReturnsPatchMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "user settings set", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	seenField := false
	seenValue := false
	seenPatchFieldSpec := false
	for _, input := range output.Inputs {
		switch input.Name {
		case "field":
			seenField = true
		case "value":
			seenValue = true
		case "field:search.filterNastyResults":
			seenPatchFieldSpec = true
		}
	}
	if !seenField || !seenValue || !seenPatchFieldSpec {
		t.Fatalf("missing patch metadata in %#v", output.Inputs)
	}
}

func TestSchemaCommandWhoamiReturnsFieldSelectionMetadata(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "whoami", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !output.SupportsFields {
		t.Fatal("whoami metadata should support fields")
	}

	foundFields := false
	for _, input := range output.Inputs {
		if input.Name == "fields" {
			foundFields = true
			break
		}
	}
	if !foundFields {
		t.Fatal("whoami metadata missing fields input")
	}
}

func TestSchemaCommandUserSearchReportsAliasTarget(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "user search", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output.AliasFor != "search" {
		t.Fatalf("AliasFor = %q, want %q", output.AliasFor, "search")
	}
	if !output.SupportsFields {
		t.Fatal("user search metadata should support fields")
	}
}

func TestSchemaCommandOutputCanBeFieldFiltered(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", "search", "--fields", "id,linked_procedure", "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(output) != 2 {
		t.Fatalf("output = %#v, want only selected fields", output)
	}
	if output["id"] != "search" {
		t.Fatalf("id = %v, want %q", output["id"], "search")
	}
	if output["linked_procedure"] != procedureUserSearch {
		t.Fatalf("linked_procedure = %v, want %q", output["linked_procedure"], procedureUserSearch)
	}
}

func TestRootHelpStillWorks(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--help"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "schema") {
		t.Fatalf("help output missing schema command: %s", stdout.String())
	}
}

func commandSchemaForTest(t *testing.T, commandID string) schemaEntry {
	t.Helper()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"schema", "command", commandID, "--output", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output schemaEntry
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return output
}

func assertInputNotUnconditionallyRequired(t *testing.T, entry schemaEntry, inputName string) {
	t.Helper()

	for _, input := range entry.Inputs {
		if input.Name == inputName {
			if input.Required {
				t.Fatalf("%s input %q is unconditionally required", entry.ID, inputName)
			}
			return
		}
	}
	t.Fatalf("%s inputs = %#v, want input %q", entry.ID, entry.Inputs, inputName)
}

func assertInputModes(t *testing.T, entry schemaEntry, expected []schemaInputMode) {
	t.Helper()

	if !reflect.DeepEqual(entry.InputModes, expected) {
		t.Fatalf("%s input modes = %#v, want %#v", entry.ID, entry.InputModes, expected)
	}
}
