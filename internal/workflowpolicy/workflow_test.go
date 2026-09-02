package workflowpolicy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const workflowsDir = "../../.github/workflows"

var shaPinned = regexp.MustCompile(`uses: [^./][^\s@]+@[0-9a-f]{40}( #.*)?$`)

func workflows(t *testing.T) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(workflowsDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = string(b)
	}
	if len(out) == 0 {
		t.Fatal("no workflows found")
	}
	return out
}

func TestOrgWorkflowInvariants(t *testing.T) {
	for name, w := range workflows(t) {
		t.Run(name, func(t *testing.T) {
			if strings.Contains(w, "\n  schedule:\n") {
				t.Fatal("declares a GitHub schedule; Cloudflare owns recurring dispatch")
			}
			if strings.Contains(w, "pull_request_target") {
				t.Fatal("uses pull_request_target")
			}
			if !strings.Contains(w, "\npermissions:") {
				t.Fatal("no workflow-level permissions block")
			}
			for _, line := range strings.Split(w, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "uses: ") && !strings.HasPrefix(trimmed, "uses: ./") && !shaPinned.MatchString(trimmed) {
					t.Fatalf("action is not SHA-pinned: %s", trimmed)
				}
			}
			if strings.Count(w, "actions/checkout@") != strings.Count(w, "persist-credentials: false") {
				t.Fatal("every actions/checkout must set persist-credentials: false")
			}
		})
	}
}
