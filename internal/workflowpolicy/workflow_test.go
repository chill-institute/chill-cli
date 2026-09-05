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

func TestVerificationRequiresPinnedContracts(t *testing.T) {
	all := workflows(t)
	checkout := regexp.MustCompile(`(?m)          repository: chill-institute/chill-contracts\n          ref: ([0-9a-f]{40})(?: #[^\n]*)?\n          path: (\S+)\n          persist-credentials: false`)
	var pin string
	for _, name := range []string{"verify.yml", "main.yml"} {
		verification, _, _ := strings.Cut(all[name], "\n  release:")
		match := checkout.FindStringSubmatch(verification)
		if match == nil {
			t.Fatalf("%s verification needs an immutable contracts checkout", name)
		}
		if pin != "" && match[1] != pin {
			t.Fatalf("%s contracts pin differs from pull-request verification", name)
		}
		pin = match[1]
		parity := "CHILLY_CONTRACTS_PROTO: ${{ github.workspace }}/" + match[2] + "/proto/chill/v4/api.proto\n        run: mise run contracts:check"
		if !strings.Contains(verification, parity) {
			t.Fatalf("%s verification must require parity against its checked-out contracts", name)
		}
	}
	if !strings.Contains(all["main.yml"], "needs: [verify]\n    if: ${{ needs.verify.result == 'success' }}") {
		t.Fatal("release must require successful verification")
	}
}
