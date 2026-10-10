package workflowpolicy

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const workflowsDir = "../../.github/workflows"

var (
	shaPinned = regexp.MustCompile(`^[^@\s]+@[0-9a-f]{40}$`)
	commitSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

type workflow struct {
	On          yaml.Node      `yaml:"on"`
	Permissions *yaml.Node     `yaml:"permissions"`
	Jobs        map[string]job `yaml:"jobs"`
}

type job struct {
	Needs yaml.Node `yaml:"needs"`
	If    string    `yaml:"if"`
	Uses  string    `yaml:"uses"`
	Steps []step    `yaml:"steps"`
}

type step struct {
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]any    `yaml:"with"`
	Env  map[string]string `yaml:"env"`
}

func workflows(t *testing.T) map[string]workflow {
	t.Helper()
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]workflow{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(workflowsDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var w workflow
		if err := yaml.Unmarshal(b, &w); err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		out[e.Name()] = w
	}
	if len(out) == 0 {
		t.Fatal("no workflows found")
	}
	return out
}

// names returns the values of a scalar or sequence node, or the keys of a mapping node.
func names(node yaml.Node) []string {
	var out []string
	switch node.Kind {
	case yaml.ScalarNode:
		out = append(out, node.Value)
	case yaml.SequenceNode:
		for _, item := range node.Content {
			out = append(out, item.Value)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			out = append(out, node.Content[i].Value)
		}
	}
	return out
}

func TestOrgWorkflowInvariants(t *testing.T) {
	for name, w := range workflows(t) {
		t.Run(name, func(t *testing.T) {
			triggers := names(w.On)
			if slices.Contains(triggers, "schedule") {
				t.Fatal("declares a GitHub schedule; Cloudflare owns recurring dispatch")
			}
			if slices.Contains(triggers, "pull_request_target") {
				t.Fatal("uses pull_request_target")
			}
			if w.Permissions == nil {
				t.Fatal("no workflow-level permissions block")
			}
			for jobName, j := range w.Jobs {
				uses := []string{j.Uses}
				for _, s := range j.Steps {
					uses = append(uses, s.Uses)
					if strings.HasPrefix(s.Uses, "actions/checkout@") && s.With["persist-credentials"] != false {
						t.Fatalf("%s: every actions/checkout must set persist-credentials: false", jobName)
					}
				}
				for _, ref := range uses {
					if ref != "" && !strings.HasPrefix(ref, "./") && !shaPinned.MatchString(ref) {
						t.Fatalf("%s: action is not SHA-pinned: %s", jobName, ref)
					}
				}
			}
		})
	}
}

func TestVerificationRequiresPinnedContracts(t *testing.T) {
	all := workflows(t)
	var pin string
	for _, name := range []string{"verify.yml", "main.yml"} {
		verify, ok := all[name].Jobs["verify"]
		if !ok {
			t.Fatalf("%s has no verify job", name)
		}
		var ref, path string
		for _, s := range verify.Steps {
			if strings.HasPrefix(s.Uses, "actions/checkout@") && s.With["repository"] == "chill-institute/chill-contracts" {
				ref, _ = s.With["ref"].(string)
				path, _ = s.With["path"].(string)
			}
		}
		if !commitSHA.MatchString(ref) || path == "" {
			t.Fatalf("%s verification needs an immutable contracts checkout", name)
		}
		if pin != "" && ref != pin {
			t.Fatalf("%s contracts pin differs from pull-request verification", name)
		}
		pin = ref

		proto := "${{ github.workspace }}/" + path + "/proto/chill/v4/api.proto"
		if !slices.ContainsFunc(verify.Steps, func(s step) bool {
			return strings.TrimSpace(s.Run) == "mise run contracts:check" && s.Env["CHILLY_CONTRACTS_PROTO"] == proto
		}) {
			t.Fatalf("%s verification must require parity against its checked-out contracts", name)
		}
	}

	release := all["main.yml"].Jobs["release"]
	if !slices.Contains(names(release.Needs), "verify") || !strings.Contains(release.If, "needs.verify.result == 'success'") {
		t.Fatal("release must require successful verification")
	}
}
