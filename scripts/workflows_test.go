package scripts_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

type workflow struct {
	On   map[string]any `yaml:"on"`
	Jobs map[string]struct {
		Needs any               `yaml:"needs"`
		If    string            `yaml:"if"`
		Uses  string            `yaml:"uses"`
		With  map[string]string `yaml:"with"`
		Steps []struct {
			ID  string `yaml:"id"`
			Run string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func readWorkflow(t *testing.T, name string) workflow {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", ".github", "workflows", name+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var result workflow
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func workflowScript(t *testing.T, w workflow, job, step string) string {
	t.Helper()
	for _, candidate := range w.Jobs[job].Steps {
		if candidate.ID == step {
			return candidate.Run
		}
	}
	t.Fatalf("missing step %s/%s", job, step)
	return ""
}

func runBash(t *testing.T, script string, env ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("bash", "-e", "-o", "pipefail", "-c", script)
	cmd.Env = append(os.Environ(), env...)
	return cmd.CombinedOutput()
}

func TestReleaseMergeDetection(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Fatal("workflow tests require jq")
	}
	script := workflowScript(t, readWorkflow(t, "dev"), "changes", "pr")
	for _, tc := range []struct {
		name, branch, repo, sha, merged string
		want                            string
	}{
		{"release merge", "release-please--branches--main", "owner/repo", "checked", "today", "true"},
		{"ordinary PR", "feature", "owner/repo", "checked", "today", "false"},
		{"fork with release branch name", "release-please--branches--main", "fork/repo", "checked", "today", "false"},
		{"older associated release PR", "release-please--branches--main", "owner/repo", "older", "today", "false"},
		{"unmerged PR", "release-please--branches--main", "owner/repo", "checked", "", "false"},
		{"direct push", "", "", "", "", "false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var merged any
			if tc.merged != "" {
				merged = tc.merged
			}
			pr := map[string]any{"merged_at": merged, "merge_commit_sha": tc.sha,
				"base": map[string]any{"ref": "main"},
				"head": map[string]any{"ref": tc.branch, "repo": map[string]any{"full_name": tc.repo}}}
			fixture, err := json.Marshal(pr)
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "direct push" {
				fixture = nil
			}
			dir := t.TempDir()
			input, output := filepath.Join(dir, "prs"), filepath.Join(dir, "output")
			if err := os.WriteFile(input, fixture, 0600); err != nil {
				t.Fatal(err)
			}
			// Mock only the API; execute the workflow's real filtering and output code.
			out, err := runBash(t, "gh() { cat \"$PR_FIXTURE\"; };\n"+script,
				"PR_FIXTURE="+input, "GITHUB_OUTPUT="+output, "GITHUB_SHA=checked", "GITHUB_REPOSITORY=owner/repo")
			if err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			got, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(string(got)) != "release-merge="+tc.want {
				t.Fatalf("unexpected output: %s", got)
			}
		})
	}
	t.Run("API errors stop development publishing", func(t *testing.T) {
		if out, err := runBash(t, "gh() { return 1; };\n"+script,
			"GITHUB_OUTPUT="+filepath.Join(t.TempDir(), "output"), "GITHUB_SHA=checked", "GITHUB_REPOSITORY=owner/repo"); err == nil {
			t.Fatalf("API failure was ignored: %s", out)
		}
	})
}

func TestReleaseTagValidation(t *testing.T) {
	script := workflowScript(t, readWorkflow(t, "release"), "version", "tag")
	for _, tag := range []string{"v0.1.0", "v1.2.3-rc.1", "v1.2.3+build.1", "main", "1.2.3", "v01.2.3", "v1.2.3/branch", "v1.2.3\nmain"} {
		t.Run(tag, func(t *testing.T) {
			valid := tag == "v0.1.0" || tag == "v1.2.3-rc.1" || tag == "v1.2.3+build.1"
			out, err := runBash(t, script, "REQUESTED_TAG="+tag, "GITHUB_OUTPUT="+filepath.Join(t.TempDir(), "output"))
			if (err == nil) != valid {
				t.Fatalf("validation result: %v: %s", err, out)
			}
		})
	}
}

func TestPublishingTriggers(t *testing.T) {
	release := readWorkflow(t, "release")
	push := release.On["push"].(map[string]any)
	if _, ok := push["branches"]; ok {
		t.Fatal("release publishing must not run on branch pushes")
	}
	if got := push["tags"].([]any); len(got) != 1 || got[0] != "v*" {
		t.Fatalf("unexpected tags: %v", got)
	}
	if _, ok := release.On["workflow_dispatch"]; !ok {
		t.Fatal("missing manual release retry")
	}
	if release.Jobs["checks"].With["ref"] != "${{ needs.version.outputs.sha }}" {
		t.Fatal("checks must use the resolved tag commit")
	}
	if _, ok := release.Jobs["release-please"]; ok {
		t.Fatal("tag publishing must not run release-please")
	}
	for _, name := range []string{"dev", "release-please"} {
		w := readWorkflow(t, name)
		push := w.On["push"].(map[string]any)
		if _, ok := push["tags"]; ok {
			t.Fatalf("%s must not run on tags", name)
		}
		if got := push["branches"].([]any); len(got) != 1 || got[0] != "main" {
			t.Fatalf("unexpected branches: %v", got)
		}
	}
}
