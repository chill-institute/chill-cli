package cli

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type protoField struct {
	Name     string
	Type     string
	Repeated bool
	Optional bool
}

func TestLocalSchemaMatchesContractsProto(t *testing.T) {
	t.Parallel()

	// The proto lives outside this module, where Go's test cache does not
	// look, so only `mise run contracts:check` (with -count=1) names it.
	path := strings.TrimSpace(os.Getenv("CHILLY_CONTRACTS_PROTO"))
	if path == "" {
		t.Skip("CHILLY_CONTRACTS_PROTO is unset; run mise run contracts:check")
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		if strings.TrimSpace(os.Getenv("CHILLY_REQUIRE_CONTRACTS")) == "1" {
			t.Fatalf("read contracts proto %q: %v", path, err)
		}
		t.Skipf("contracts proto not available at %q", path)
	}

	messages := parseProtoMessages(string(payload))
	for _, typeID := range localContractTypeIDs() {
		messageName, ok := strings.CutPrefix(typeID, "chill.v4.")
		if !ok {
			continue
		}
		assertTypeMatchesProtoMessage(t, typeID, messages[messageName])
	}
}

func TestAdvertisedOutputTypesResolve(t *testing.T) {
	t.Parallel()

	for _, entry := range append(listCommandSchemas(), listProcedureSchemas()...) {
		entry := entry
		t.Run(fmt.Sprintf("%s/%s", entry.Kind, entry.ID), func(t *testing.T) {
			t.Parallel()
			assertSchemaTypeResolves(t, entry.Output.Type)
		})
	}

	for _, entry := range listTypeSchemas() {
		entry := entry
		t.Run(fmt.Sprintf("type/%s", entry.ID), func(t *testing.T) {
			t.Parallel()
			for _, field := range entry.Fields {
				assertSchemaTypeResolves(t, field.Type)
			}
		})
	}
}

func localContractTypeIDs() []string {
	typeIDs := make([]string, 0, len(typeSchemaRegistry))
	for typeID := range typeSchemaRegistry {
		if strings.HasPrefix(typeID, "chill.v4.") {
			typeIDs = append(typeIDs, typeID)
		}
	}
	sort.Strings(typeIDs)
	return typeIDs
}

func assertSchemaTypeResolves(t *testing.T, typeID string) {
	t.Helper()

	if typeID == "" || isPrimitiveSchemaType(typeID) {
		return
	}
	if _, ok := lookupTypeSchema(typeID); !ok {
		t.Fatalf("schema type %q does not resolve", typeID)
	}
}

func isPrimitiveSchemaType(typeID string) bool {
	switch typeID {
	case "array", "boolean", "integer", "number", "object", "string":
		return true
	default:
		return false
	}
}

func assertTypeMatchesProtoMessage(t *testing.T, typeID string, protoFields []protoField) {
	t.Helper()

	schema, ok := typeSchemaRegistry[typeID]
	if !ok {
		t.Fatalf("missing local schema type %q", typeID)
	}
	if len(protoFields) == 0 {
		t.Fatalf("missing proto message for %q", typeID)
	}
	if len(schema.Fields) != len(protoFields) {
		t.Fatalf("%s field count = %d, want %d", typeID, len(schema.Fields), len(protoFields))
	}

	byName := make(map[string]schemaField, len(schema.Fields))
	for _, field := range schema.Fields {
		byName[field.Name] = field
	}

	for _, protoField := range protoFields {
		field, ok := byName[protoField.Name]
		if !ok {
			t.Fatalf("%s missing field %q", typeID, protoField.Name)
		}
		if field.Type != protoField.Type {
			t.Fatalf("%s.%s type = %q, want %q", typeID, field.Name, field.Type, protoField.Type)
		}
		if field.Repeated != protoField.Repeated {
			t.Fatalf("%s.%s repeated = %v, want %v", typeID, field.Name, field.Repeated, protoField.Repeated)
		}
		if field.Optional != protoField.Optional {
			t.Fatalf("%s.%s optional = %v, want %v", typeID, field.Name, field.Optional, protoField.Optional)
		}
		if field.JSONName != snakeToLowerCamel(field.Name) {
			t.Fatalf("%s.%s json_name = %q, want %q", typeID, field.Name, field.JSONName, snakeToLowerCamel(field.Name))
		}
	}
}

func parseProtoMessages(payload string) map[string][]protoField {
	messages := map[string][]protoField{}
	enumTypes := parseProtoEnumTypes(payload)
	fieldPattern := regexp.MustCompile(`(?m)^\s*(optional\s+|repeated\s+)?([A-Za-z_]\w*)\s+([A-Za-z_]\w*)\s*=\s*\d+\s*(?:\[[^\]]*\])?\s*;`)

	for _, message := range parseProtoMessageBlocks(payload) {
		fields := make([]protoField, 0)
		for _, fieldMatch := range fieldPattern.FindAllStringSubmatch(message.body, -1) {
			modifier := strings.TrimSpace(fieldMatch[1])
			fields = append(fields, protoField{
				Name:     fieldMatch[3],
				Type:     schemaTypeForProtoField(fieldMatch[2], enumTypes),
				Repeated: modifier == "repeated",
				Optional: modifier == "optional",
			})
		}
		messages[message.name] = fields
	}
	return messages
}

type protoMessageBlock struct {
	name string
	body string
}

func parseProtoMessageBlocks(payload string) []protoMessageBlock {
	messagePattern := regexp.MustCompile(`\bmessage\s+(\w+)\s*\{`)
	matches := messagePattern.FindAllStringSubmatchIndex(payload, -1)
	blocks := make([]protoMessageBlock, 0, len(matches))
	for _, match := range matches {
		name := payload[match[2]:match[3]]
		bodyStart := match[1]
		depth := 1
		for cursor := bodyStart; cursor < len(payload); cursor++ {
			switch payload[cursor] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					blocks = append(blocks, protoMessageBlock{
						name: name,
						body: payload[bodyStart:cursor],
					})
					cursor = len(payload)
				}
			}
		}
	}
	return blocks
}

func parseProtoEnumTypes(payload string) map[string]bool {
	enumPattern := regexp.MustCompile(`(?m)^\s*enum\s+(\w+)\s*\{`)
	enumTypes := map[string]bool{}
	for _, enumMatch := range enumPattern.FindAllStringSubmatch(payload, -1) {
		enumTypes[enumMatch[1]] = true
	}
	return enumTypes
}

func schemaTypeForProtoField(protoType string, enumTypes map[string]bool) string {
	switch protoType {
	case "string":
		return "string"
	case "int32", "int64":
		return "integer"
	case "double", "float":
		return "number"
	case "bool":
		return "boolean"
	case "ReleaseInfo":
		return typeReleaseInfo
	case "SearchResult":
		return typeSearchResult
	default:
		if enumTypes[protoType] {
			return "string"
		}
		return "chill.v4." + protoType
	}
}
