package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buildkite/bintest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validatePipelineWithAgent runs buildkite-agent pipeline upload --dry-run
// to verify the generated YAML is structurally valid. Skips gracefully if
// buildkite-agent is not installed.
func validatePipelineWithAgent(t *testing.T, pipelinePath string) {
	t.Helper()
	agentPath, err := exec.LookPath("buildkite-agent")
	if err != nil {
		t.Log("buildkite-agent not installed; skipping dry-run validation")
		return
	}
	// --agent-access-token is required by the agent even for --dry-run; "dummy" satisfies the check without a real token
	cmd := exec.Command(agentPath, "pipeline", "upload", pipelinePath, "--dry-run", "--agent-access-token", "dummy")
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	require.NoError(t, err, "Buildkite rejected generated YAML")
}

func mockGeneratePipeline(steps []Step, plugin Plugin) (*os.File, bool, error) {
	mockFile, err := os.Create("pipeline.txt")
	if err != nil {
		return nil, false, err
	}

	_, err = mockFile.WriteString(`steps:
  - command: echo "hello"
`)
	if err != nil {
		return nil, false, err
	}

	if err := mockFile.Close(); err != nil {
		return nil, false, err
	}

	return mockFile, true, nil
}

func TestUploadPipelineCallsBuildkiteAgentCommand(t *testing.T) {
	plugin := Plugin{Diff: "echo ./foo-service", Interpolation: true}

	agent, err := bintest.NewMock("buildkite-agent")
	require.NoError(t, err)

	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	_ = os.Setenv("PATH", filepath.Dir(agent.Path)+":"+oldPath)

	agent.
		Expect("pipeline", "upload", "pipeline.txt").
		AndExitWith(0)

	cmd, args, err := uploadPipeline(plugin, mockGeneratePipeline)

	assert.Equal(t, "buildkite-agent", cmd)
	assert.Equal(t, []string{"pipeline", "upload", "pipeline.txt"}, args)
	assert.NoError(t, err)

	require.NoError(t, agent.CheckAndClose(t))
}

func TestUploadPipelineCallsBuildkiteAgentCommandWithInterpolation(t *testing.T) {
	plugin := Plugin{Diff: "echo ./foo-service", Interpolation: false}

	agent, err := bintest.NewMock("buildkite-agent")
	require.NoError(t, err)

	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	_ = os.Setenv("PATH", filepath.Dir(agent.Path)+":"+oldPath)

	agent.
		Expect("pipeline", "upload", "pipeline.txt", "--no-interpolation").
		AndExitWith(0)

	cmd, args, err := uploadPipeline(plugin, mockGeneratePipeline)

	assert.Equal(t, "buildkite-agent", cmd)
	assert.Equal(t, []string{"pipeline", "upload", "pipeline.txt", "--no-interpolation"}, args)
	assert.NoError(t, err)

	require.NoError(t, agent.CheckAndClose(t))
}

func TestUploadPipelineCancelsIfThereIsNoDiffOutput(t *testing.T) {
	plugin := Plugin{Diff: "echo"}
	cmd, args, err := uploadPipeline(plugin, mockGeneratePipeline)

	assert.Equal(t, "", cmd)
	assert.Equal(t, []string{}, args)
	assert.Equal(t, nil, err)
}

func TestUploadPipelineWithEmptyGeneratedPipeline(t *testing.T) {
	plugin := Plugin{Diff: "echo ./bar-service"}
	cmd, args, err := uploadPipeline(plugin, generatePipeline)

	assert.Equal(t, "", cmd)
	assert.Equal(t, []string{}, args)
	assert.Equal(t, nil, err)
}

func TestDiff(t *testing.T) {
	want := []string{
		"services/foo/serverless.yml",
		"services/bar/config.yml",
		"ops/bar/config.yml",
		"README.md",
	}

	got, err := diff(`printf 'services/foo/serverless.yml\nservices/bar/config.yml\n\nops/bar/config.yml\nREADME.md\n'`)

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffWithSubshell(t *testing.T) {
	want := []string{
		"user-service/infrastructure/cloudfront.yaml",
		"user-service/my config/settings.yaml",
		"user-service/serverless.yaml",
	}
	got, err := diff("cat e2e/multiple-paths")
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffRealisticGitOutput(t *testing.T) {
	// Fixture mirrors git diff --name-only output: plain paths, C-style
	// quoted paths (tab, octal emoji), and paths with spaces.
	want := []string{
		"normal/path.go",
		"path/with\tescape.go",
		"directory/File Name With Spaces.md",
		"projects/17_🪁_emoji.py",
		"another dir/some file.txt",
	}
	got, err := diff("cat e2e/diff-output-realistic")
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffWithQuotedPaths(t *testing.T) {
	want := []string{
		"projects/test/pages/17_🪁_testfile.py",
		"normal/file.txt",
	}
	got, err := diff(`printf '"projects/test/pages/17_\360\237\252\201_testfile.py"\nnormal/file.txt\n'`)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffWithSpacesInFilenames(t *testing.T) {
	// Simulates git diff --name-only output with one filename per line,
	// where some filenames contain spaces.
	want := []string{
		"directory/File Name With Spaces.md",
		"another dir/some file.txt",
		"no-spaces.go",
	}

	// printf produces newline-separated output, just like git diff --name-only
	got, err := diff(`printf 'directory/File Name With Spaces.md\nanother dir/some file.txt\nno-spaces.go\n'`)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffSingleFile(t *testing.T) {
	want := []string{
		"services/foo/serverless.yml",
	}

	got, err := diff(`printf 'services/foo/serverless.yml\n'`)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffWithSpacesInFilenamesSingleFile(t *testing.T) {
	want := []string{
		"directory/File Name With Spaces.md",
	}

	// printf produces newline-separated output, just like git diff --name-only
	got, err := diff(`printf 'directory/File Name With Spaces.md\n'`)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffEmptyOutput(t *testing.T) {
	got, err := diff(`printf ''`)
	assert.NoError(t, err)
	assert.Equal(t, []string{}, got)
}

func TestDiffWhitespaceOnlyOutput(t *testing.T) {
	got, err := diff(`printf '\n\n\n'`)
	assert.NoError(t, err)
	assert.Equal(t, []string{}, got)
}

func TestDiffWhitespaceOnlyNoNewlines(t *testing.T) {
	got, err := diff(`printf '   \t  '`)
	assert.NoError(t, err)
	assert.Equal(t, []string{}, got)
}

func TestDiffSingleFileNoTrailingNewline(t *testing.T) {
	// Legacy compat: custom diff commands may not emit a trailing newline
	want := []string{"services/foo/serverless.yml"}
	got, err := diff(`printf 'services/foo/serverless.yml'`)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffWindowsLineEndings(t *testing.T) {
	want := []string{
		"services/foo/file.go",
		"services/bar/file.go",
	}
	got, err := diff(`printf 'services/foo/file.go\r\nservices/bar/file.go\r\n'`)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffQuotedPathsWithSpaces(t *testing.T) {
	// Git C-style quotes paths with special chars; spaces alone don't trigger quoting,
	// but paths with both spaces and special chars will be quoted.
	want := []string{
		"projects/my docs/17_🪁_file.py",
		"normal/file.txt",
	}
	got, err := diff(`printf '"projects/my docs/17_\360\237\252\201_file.py"\nnormal/file.txt\n'`)
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestStepsToTriggerWithEmojiPaths(t *testing.T) {
	watch := []WatchConfig{
		{
			Paths: []string{"projects/**"},
			Step:  Step{Trigger: "test-pipeline"},
		},
	}

	changedFiles := []string{
		"projects/test/pages/17_🪁_testfile.py",
		"other/file.txt",
	}

	steps, err := stepsToTrigger(changedFiles, watch, false)
	assert.NoError(t, err)
	assert.Equal(t, []Step{{Trigger: "test-pipeline"}}, steps)
}

func TestPipelinesToTriggerGetsListOfPipelines(t *testing.T) {
	want := []string{"service-1", "service-2", "service-4"}

	watch := []WatchConfig{
		{
			Paths: []string{"watch-path-1"},
			Step:  Step{Trigger: "service-1"},
		},
		{
			Paths: []string{"watch-path-2/", "watch-path-3/", "watch-path-4"},
			Step:  Step{Trigger: "service-2"},
		},
		{
			Paths: []string{"watch-path-5"},
			Step:  Step{Trigger: "service-3"},
		},
		{
			Paths: []string{"watch-path-2"},
			Step:  Step{Trigger: "service-4"},
		},
	}

	changedFiles := []string{
		"watch-path-1/text.txt",
		"watch-path-2/.gitignore",
		"watch-path-2/src/index.go",
		"watch-path-4/test/index_test.go",
	}

	pipelines, err := stepsToTrigger(changedFiles, watch, false)
	assert.NoError(t, err)
	var got []string

	for _, v := range pipelines {
		got = append(got, v.Trigger)
	}

	assert.Equal(t, want, got)
}

func TestPipelinesStepsToTrigger(t *testing.T) {
	testCases := map[string]struct {
		ChangedFiles []string
		WatchConfigs []WatchConfig
		Expected     []Step
	}{
		"service-1": {
			ChangedFiles: []string{
				"watch-path-1/text.txt",
				"watch-path-2/.gitignore",
			},
			WatchConfigs: []WatchConfig{{
				Paths: []string{"watch-path-1"},
				Step:  Step{Trigger: "service-1"},
			}},
			Expected: []Step{
				{Trigger: "service-1"},
			},
		},
		"service-1-2": {
			ChangedFiles: []string{
				"watch-path-1/text.txt",
				"watch-path-2/.gitignore",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"watch-path-1"},
					Step:  Step{Trigger: "service-1"},
				},
				{
					Paths: []string{"watch-path-2"},
					Step:  Step{Trigger: "service-2"},
				},
			},
			Expected: []Step{
				{Trigger: "service-1"},
				{Trigger: "service-2"},
			},
		},
		"extension wildcard": {
			ChangedFiles: []string{
				"text.txt",
				".gitignore",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"*.txt"},
					Step:  Step{Trigger: "txt"},
				},
			},
			Expected: []Step{
				{Trigger: "txt"},
			},
		},
		"extension wildcard in subdir": {
			ChangedFiles: []string{
				"docs/text.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"docs/*.txt"},
					Step:  Step{Trigger: "txt"},
				},
			},
			Expected: []Step{
				{Trigger: "txt"},
			},
		},
		"directory wildcard": {
			ChangedFiles: []string{
				"docs/text.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"**/text.txt"},
					Step:  Step{Trigger: "txt"},
				},
			},
			Expected: []Step{
				{Trigger: "txt"},
			},
		},
		"directory and extension wildcard": {
			ChangedFiles: []string{
				"package/other.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"*/*.txt"},
					Step:  Step{Trigger: "txt"},
				},
			},
			Expected: []Step{
				{Trigger: "txt"},
			},
		},
		"double directory and extension wildcard": {
			ChangedFiles: []string{
				"package/docs/other.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"**/*.txt"},
					Step:  Step{Trigger: "txt"},
				},
			},
			Expected: []Step{
				{Trigger: "txt"},
			},
		},
		"default configuration": {
			ChangedFiles: []string{
				"unmatched/file.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"app/"},
					Step:  Step{Trigger: "app-deploy"},
				},
				{
					Paths: []string{"test/bin/"},
					Step:  Step{Command: "echo Make Changes to Bin"},
				},
				{
					Default: struct{}{},
					Step:    Step{Command: "buildkite-agent pipeline upload other_tests.yml"},
				},
			},
			Expected: []Step{
				{Command: "buildkite-agent pipeline upload other_tests.yml"},
			},
		},
		"skips service-2": {
			ChangedFiles: []string{
				"watch-path/text.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"watch-path"},
					Step:  Step{Trigger: "service-1"},
				},
				{
					Paths:     []string{"watch-path"},
					SkipPaths: []string{"watch-path/text.txt"},
					Step:      Step{Trigger: "service-2"},
				},
			},
			Expected: []Step{
				{Trigger: "service-1"},
			},
		},
		"skips extension wildcard": {
			ChangedFiles: []string{
				"text.secret.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"*.txt"},
					Step:  Step{Trigger: "service-1"},
				},
				{
					Paths:     []string{"*.txt"},
					SkipPaths: []string{"*.secret.txt"},
					Step:      Step{Trigger: "service-2"},
				},
			},
			Expected: []Step{
				{Trigger: "service-1"},
			},
		},
		"skips extension wildcard in subdir": {
			ChangedFiles: []string{
				"docs/text.secret.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"**/*.txt"},
					Step:  Step{Trigger: "service-1"},
				},
				{
					Paths:     []string{"**/*.txt"},
					SkipPaths: []string{"docs/*.txt"},
					Step:      Step{Trigger: "service-2"},
				},
			},
			Expected: []Step{
				{Trigger: "service-1"},
			},
		},
		"step is included even when one of the files is skipped": {
			ChangedFiles: []string{
				"docs/text.secret.txt",
				"docs/text.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"**/*.txt"},
					Step:  Step{Trigger: "service-1"},
				},
				{
					Paths:     []string{"**/*.txt"},
					SkipPaths: []string{"docs/*.secret.txt"},
					Step:      Step{Trigger: "service-2"},
				},
			},
			Expected: []Step{
				{Trigger: "service-1"},
				{Trigger: "service-2"},
			},
		},
		"fails if not path is included": {
			ChangedFiles: []string{
				"docs/text.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					SkipPaths: []string{"docs/*.secret.txt"},
					Step:      Step{Trigger: "service-1"},
				},
			},
			Expected: []Step{},
		},
		"step is not included if except path is set": {
			ChangedFiles: []string{
				"main/test/test.txt",
				"main/test/test2.txt",
				"main/other/other.txt",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:       []string{"**/*"},
					ExceptPaths: []string{"main/other/**/*"},
					Step:        Step{Trigger: "service-1"},
				},
				{
					Paths: []string{"main/other/**/*"},
					Step:  Step{Trigger: "service-2"},
				},
			},
			Expected: []Step{
				{Trigger: "service-2"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			steps, err := stepsToTrigger(tc.ChangedFiles, tc.WatchConfigs, false)
			assert.NoError(t, err)
			assert.Equal(t, tc.Expected, steps)
		})
	}
}

func TestStepsToTriggerSkipOnNoChanges(t *testing.T) {
	testCases := map[string]struct {
		ChangedFiles    []string
		WatchConfigs    []WatchConfig
		SkipOnNoChanges bool
		Expected        []Step
	}{
		"unmatched step is emitted with skip when enabled": {
			ChangedFiles: []string{"app/main.go"},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"services/"},
					Step:  Step{Trigger: "deploy-services"},
				},
				{
					Paths: []string{"app/"},
					Step:  Step{Trigger: "deploy-app"},
				},
			},
			SkipOnNoChanges: true,
			Expected: []Step{
				{Trigger: "deploy-services", Skip: skipNoChangesMessage},
				{Trigger: "deploy-app"},
			},
		},
		"unmatched step is omitted when disabled (legacy behaviour)": {
			ChangedFiles: []string{"app/main.go"},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"services/"},
					Step:  Step{Trigger: "deploy-services"},
				},
				{
					Paths: []string{"app/"},
					Step:  Step{Trigger: "deploy-app"},
				},
			},
			SkipOnNoChanges: false,
			Expected: []Step{
				{Trigger: "deploy-app"},
			},
		},
		"matched step is never marked skipped": {
			ChangedFiles: []string{"services/main.go"},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"services/"},
					Step:  Step{Trigger: "deploy-services"},
				},
			},
			SkipOnNoChanges: true,
			Expected: []Step{
				{Trigger: "deploy-services"},
			},
		},
		"excepted watch stays fully omitted even when enabled": {
			ChangedFiles: []string{"main/other/file.txt"},
			WatchConfigs: []WatchConfig{
				{
					Paths:       []string{"**/*"},
					ExceptPaths: []string{"main/other/**/*"},
					Step:        Step{Trigger: "service-1"},
				},
			},
			SkipOnNoChanges: true,
			Expected:        []Step{},
		},
		"default watch still fires when nothing matches, skipped placeholders don't count as a match": {
			ChangedFiles: []string{"unmatched/file.txt"},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"app/"},
					Step:  Step{Trigger: "app-deploy"},
				},
				{
					Default: struct{}{},
					Step:    Step{Command: "buildkite-agent pipeline upload other_tests.yml"},
				},
			},
			SkipOnNoChanges: true,
			Expected: []Step{
				{Trigger: "app-deploy", Skip: skipNoChangesMessage},
				{Command: "buildkite-agent pipeline upload other_tests.yml"},
			},
		},
		"group step is emitted with skip when its watch path doesn't match": {
			ChangedFiles: []string{"app/main.go"},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"services/"},
					Step: Step{
						Group: "CI/CD Infrastructure",
						Key:   "group:cicd",
						Steps: []Step{
							{Command: "echo deploy"},
						},
					},
				},
				{
					Paths: []string{"app/"},
					Step: Step{
						Command:   "echo build-app",
						DependsOn: "group:cicd",
					},
				},
			},
			SkipOnNoChanges: true,
			Expected: []Step{
				{
					Group: "CI/CD Infrastructure",
					Key:   "group:cicd",
					Skip:  skipNoChangesMessage,
					Steps: []Step{
						{Command: "echo deploy"},
					},
				},
				{
					Command:   "echo build-app",
					DependsOn: "group:cicd",
				},
			},
		},
		"changes excluded entirely by skip_path get a distinct skip reason, not 'no changes detected'": {
			ChangedFiles: []string{"services/api/README.md"},
			WatchConfigs: []WatchConfig{
				{
					Paths:     []string{"services/api/"},
					SkipPaths: []string{"services/api/README.md"},
					Step:      Step{Trigger: "deploy-api"},
				},
			},
			SkipOnNoChanges: true,
			Expected: []Step{
				{Trigger: "deploy-api", Skip: skipPathExcludedMessage},
			},
		},
		"watch entry with no path configured is not injected as a skip placeholder": {
			ChangedFiles: []string{"vendor/lib.go"},
			WatchConfigs: []WatchConfig{
				{
					SkipPaths: []string{"vendor/"},
					Step:      Step{Command: "echo deploy-something"},
				},
			},
			SkipOnNoChanges: true,
			Expected:        []Step{},
		},
		"two watch entries sharing a key don't produce duplicate steps when only one matches": {
			ChangedFiles: []string{"path-a/main.go"},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"path-a/"},
					Step: Step{
						Group: "Deploy",
						Key:   "group:deploy",
						Steps: []Step{{Command: "echo deploy"}},
					},
				},
				{
					Paths: []string{"path-b/"},
					Step: Step{
						Group: "Deploy",
						Key:   "group:deploy",
						Steps: []Step{{Command: "echo deploy"}},
					},
				},
			},
			SkipOnNoChanges: true,
			Expected: []Step{
				{
					Group: "Deploy",
					Key:   "group:deploy",
					Steps: []Step{{Command: "echo deploy"}},
				},
			},
		},
		"two watch entries sharing a key that both genuinely match are not silently collapsed": {
			ChangedFiles: []string{"path-a/main.go", "path-b/main.go"},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"path-a/"},
					Step:  Step{Command: "echo a", Key: "dup"},
				},
				{
					Paths: []string{"path-b/"},
					Step:  Step{Command: "echo b", Key: "dup"},
				},
			},
			SkipOnNoChanges: true,
			Expected: []Step{
				{Command: "echo a", Key: "dup"},
				{Command: "echo b", Key: "dup"},
			},
		},
		"two watch entries sharing a key that both go unmatched keep the more specific skip_path reason": {
			ChangedFiles: []string{"other/file.txt", "path-b/README.md"},
			WatchConfigs: []WatchConfig{
				{
					Paths: []string{"path-a/"},
					Step:  Step{Command: "echo a", Key: "dup"},
				},
				{
					Paths:     []string{"path-b/"},
					SkipPaths: []string{"path-b/README.md"},
					Step:      Step{Command: "echo b", Key: "dup"},
				},
			},
			SkipOnNoChanges: true,
			Expected: []Step{
				{Command: "echo b", Key: "dup", Skip: skipPathExcludedMessage},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			steps, err := stepsToTrigger(tc.ChangedFiles, tc.WatchConfigs, tc.SkipOnNoChanges)
			assert.NoError(t, err)
			assert.Equal(t, tc.Expected, steps)
		})
	}
}

func TestRegexPaths(t *testing.T) {
	testCases := map[string]struct {
		ChangedFiles []string
		WatchConfigs []WatchConfig
		Expected     []Step
		ExpectError  bool
	}{
		"matches path using regex with lookahead": {
			ChangedFiles: []string{
				"src/components/Button.tsx",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:      []string{`src/(?!__tests__/).*\.[tj]sx?`},
					RegexPaths: true,
					Step:       Step{Trigger: "service-1"},
				},
			},
			Expected: []Step{
				{Trigger: "service-1"},
			},
		},
		"does not match path excluded by negative lookahead": {
			ChangedFiles: []string{
				"src/__tests__/Button.test.tsx",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:      []string{`src/(?!__tests__/).*\.[tj]sx?`},
					RegexPaths: true,
					Step:       Step{Trigger: "service-1"},
				},
			},
			Expected: []Step{},
		},
		"matches customer example pattern": {
			ChangedFiles: []string{
				"src/components/Button.tsx",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:      []string{`src/(?!pulumi|ci-generators|desktop|mobile|test)(?!.*\.test\.)(?!.*\.snap$)(?!.*/__test__/)(?!.*/__mocks__/)(?!.*/__snapshots__/).*\.[tj]sx?`},
					RegexPaths: true,
					Step:       Step{Trigger: "service-1"},
				},
			},
			Expected: []Step{
				{Trigger: "service-1"},
			},
		},
		"skip_path regex excludes matched files": {
			ChangedFiles: []string{
				"src/components/Button.tsx",
				"src/components/Button.test.tsx",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:      []string{`src/.*\.[tj]sx?`},
					SkipPaths:  []string{`.*\.test\.[tj]sx?`},
					RegexPaths: true,
					Step:       Step{Trigger: "service-1"},
				},
			},
			Expected: []Step{
				{Trigger: "service-1"},
			},
		},
		"except_path regex prevents step from triggering": {
			ChangedFiles: []string{
				"src/generated/schema.ts",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:       []string{`src/.*`},
					ExceptPaths: []string{`src/generated/.*`},
					RegexPaths:  true,
					Step:        Step{Trigger: "service-1"},
				},
			},
			Expected: []Step{},
		},
		"invalid regex returns error": {
			ChangedFiles: []string{
				"src/components/Button.tsx",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:      []string{`src/[invalid`},
					RegexPaths: true,
					Step:       Step{Trigger: "service-1"},
				},
			},
			Expected:    []Step{},
			ExpectError: true,
		},
		"regex_paths false uses existing glob behaviour": {
			ChangedFiles: []string{
				"src/components/Button.tsx",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:      []string{`src/**/*.tsx`},
					RegexPaths: false,
					Step:       Step{Trigger: "service-1"},
				},
			},
			Expected: []Step{
				{Trigger: "service-1"},
			},
		},
		"regex and non-regex watch blocks work independently": {
			ChangedFiles: []string{
				"src/components/Button.tsx",
				"services/api/main.go",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:      []string{`src/(?!__tests__/).*\.tsx`},
					RegexPaths: true,
					Step:       Step{Trigger: "frontend"},
				},
				{
					Paths: []string{"services/api/"},
					Step:  Step{Trigger: "backend"},
				},
			},
			Expected: []Step{
				{Trigger: "frontend"},
				{Trigger: "backend"},
			},
		},
		"invalid regex in skip_path returns error": {
			ChangedFiles: []string{
				"src/main.go",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:      []string{`src/.*`},
					SkipPaths:  []string{`src/[invalid`},
					RegexPaths: true,
					Step:       Step{Trigger: "service-1"},
				},
			},
			Expected:    []Step{},
			ExpectError: true,
		},
		"regex does not match unrelated file": {
			ChangedFiles: []string{
				"docs/readme.md",
			},
			WatchConfigs: []WatchConfig{
				{
					Paths:      []string{`src/.*\.go`},
					RegexPaths: true,
					Step:       Step{Trigger: "service-1"},
				},
			},
			Expected: []Step{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			steps, err := stepsToTrigger(tc.ChangedFiles, tc.WatchConfigs, false)
			if tc.ExpectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, steps)
			}
		})
	}
}

func TestGeneratePipeline(t *testing.T) {
	steps := []Step{
		{
			Trigger:  "foo-service-pipeline",
			Build:    Build{Message: "build message"},
			SoftFail: true,
			Notify: []StepNotify{
				{Slack: "@adikari"},
			},
		},
		{
			Trigger: "notification-test",
			Command: "command-to-run",
			Notify: []StepNotify{
				{Basecamp: "https://basecamp-url"},
				{GithubStatus: GithubStatusNotification{Context: "my-custom-status"}},
				{Slack: "@someuser", Condition: "build.state === \"passed\""},
			},
		},
		{
			Group:   "my group",
			Trigger: "foo-service-pipeline",
			Build:   Build{Message: "build message"},
		},
	}

	plugin := Plugin{
		Wait: true,
		Notify: []PluginNotify{
			{Email: "foo@gmail.com"},
			{Email: "bar@gmail.com"},
			{Basecamp: "https://basecamp"},
			{Webhook: "https://webhook"},
			{Slack: "@adikari"},
			{GithubStatus: GithubStatusNotification{Context: "github-context"}},
		},
		Hooks: []HookConfig{
			{Command: "echo \"hello world\""},
			{Command: "cat ./file.txt"},
		},
	}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("Failed to remove temporary pipeline file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)

	want := `notify:
    - email: foo@gmail.com
    - email: bar@gmail.com
    - basecamp_campfire: https://basecamp
    - webhook: https://webhook
    - slack: '@adikari'
    - github_commit_status:
        context: github-context
steps:
    - trigger: foo-service-pipeline
      build:
        message: build message
      soft_fail: true
      notify:
        - slack: '@adikari'
    - trigger: notification-test
      command: command-to-run
      notify:
        - basecamp_campfire: https://basecamp-url
        - github_commit_status:
            context: my-custom-status
        - slack: '@someuser'
          if: build.state === "passed"
    - group: my group
      steps:
        - trigger: foo-service-pipeline
          build:
            message: build message
    - wait: null
    - command: echo "hello world"
    - command: cat ./file.txt
`

	t.Log("Generated pipeline:\n" + string(got))
	assert.Equal(t, want, string(got))

	validatePipelineWithAgent(t, pipeline.Name())
}

func TestGeneratePipelineWithNoStepsAndHooks(t *testing.T) {
	steps := []Step{}

	want := `steps:
    - wait: null
    - command: echo "hello world"
    - command: cat ./file.txt
`

	plugin := Plugin{
		Wait: true,
		Hooks: []HookConfig{
			{Command: "echo \"hello world\""},
			{Command: "cat ./file.txt"},
		},
	}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("failed to remove temporary file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)
	t.Log("Generated pipeline:\n" + string(got))
	assert.Equal(t, want, string(got))

	validatePipelineWithAgent(t, pipeline.Name())
}

func TestGeneratePipelineWithNoStepsAndNoHooks(t *testing.T) {
	steps := []Step{}

	want := `steps: []
`

	plugin := Plugin{}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("Failed to remove temporary pipeline file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)
	t.Log("Generated pipeline:\n" + string(got))
	assert.Equal(t, want, string(got))

	validatePipelineWithAgent(t, pipeline.Name())
}

func TestGeneratePipelineWithCondition(t *testing.T) {
	steps := []Step{
		{
			Command:   "echo deploy to production",
			Label:     "Deploy",
			Condition: "build.branch == 'main' && build.pull_request.id == null",
		},
		{
			Trigger:   "test-pipeline",
			Condition: "build.message =~ /\\[deploy\\]/",
		},
	}

	want := `steps:
    - label: Deploy
      if: build.branch == 'main' && build.pull_request.id == null
      command: echo deploy to production
    - trigger: test-pipeline
      if: build.message =~ /\[deploy\]/
`

	plugin := Plugin{Wait: false}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("Failed to remove temporary pipeline file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)
	t.Log("Generated pipeline:\n" + string(got))
	assert.Equal(t, want, string(got))

	validatePipelineWithAgent(t, pipeline.Name())
}

func TestGeneratePipelineWithDependsOn(t *testing.T) {
	steps := []Step{
		{
			Command: "echo build",
			Label:   "Build",
		},
		{
			Command:   "echo test",
			Label:     "Test",
			DependsOn: "build-step",
		},
		{
			Trigger:   "deploy-pipeline",
			DependsOn: []interface{}{"build-step", "test-step"},
		},
	}

	want := `steps:
    - label: Build
      command: echo build
    - label: Test
      command: echo test
      depends_on: build-step
    - trigger: deploy-pipeline
      depends_on:
        - build-step
        - test-step
`

	plugin := Plugin{Wait: false}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("Failed to remove temporary pipeline file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)
	t.Log("Generated pipeline:\n" + string(got))
	assert.Equal(t, want, string(got))

	validatePipelineWithAgent(t, pipeline.Name())
}

func TestGeneratePipelineWithStepKey(t *testing.T) {
	steps := []Step{
		{
			Command: "echo build",
			Label:   "Build",
			Key:     "build-step",
		},
		{
			Command:   "echo test",
			Label:     "Test",
			Key:       "test-step",
			DependsOn: "build-step",
		},
		{
			Trigger:   "deploy-pipeline",
			DependsOn: []interface{}{"build-step", "test-step"},
		},
	}

	want := `steps:
    - label: Build
      command: echo build
      key: build-step
    - label: Test
      command: echo test
      depends_on: build-step
      key: test-step
    - trigger: deploy-pipeline
      depends_on:
        - build-step
        - test-step
`

	plugin := Plugin{Wait: false}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("Failed to remove temporary pipeline file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)

	t.Log("Generated pipeline:\n" + string(got))
	assert.Equal(t, want, string(got))

	validatePipelineWithAgent(t, pipeline.Name())
}

func TestGeneratePipelineWithSecretsAsMap(t *testing.T) {
	steps := []Step{
		{
			Command: "echo deploy",
			Label:   "Deploy",
			Secrets: map[string]interface{}{
				"DATABRICKS_HOST":  "databricks_host_secret",
				"DATABRICKS_TOKEN": "databricks_token_secret",
			},
		},
	}

	plugin := Plugin{Wait: false}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("Failed to remove temporary pipeline file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)

	// Check that the output contains the expected secrets (order may vary in maps)
	assert.Contains(t, string(got), "label: Deploy")
	assert.Contains(t, string(got), "command: echo deploy")
	assert.Contains(t, string(got), "secrets:")
	assert.Contains(t, string(got), "DATABRICKS_HOST: databricks_host_secret")
	assert.Contains(t, string(got), "DATABRICKS_TOKEN: databricks_token_secret")

	validatePipelineWithAgent(t, pipeline.Name())
}

func TestGeneratePipelineWithSecretsAsArray(t *testing.T) {
	steps := []Step{
		{
			Command: "echo deploy",
			Label:   "Deploy",
			Secrets: []interface{}{"API_ACCESS_TOKEN", "DATABASE_PASSWORD"},
		},
	}

	want := `steps:
    - label: Deploy
      command: echo deploy
      secrets:
        - API_ACCESS_TOKEN
        - DATABASE_PASSWORD
`

	plugin := Plugin{Wait: false}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("Failed to remove temporary pipeline file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)

	t.Log("Generated pipeline:\n" + string(got))
	assert.Equal(t, want, string(got))

	validatePipelineWithAgent(t, pipeline.Name())
}

func TestGeneratePipelineWithSecretsInGroup(t *testing.T) {
	steps := []Step{
		{
			Group: "deploy group",
			Steps: []Step{
				{
					Command: "echo deploy uat",
					Label:   "Deploy UAT",
					Secrets: map[string]interface{}{
						"DB_HOST": "uat_db_host",
					},
				},
				{
					Command: "echo deploy prod",
					Label:   "Deploy Prod",
					Secrets: []interface{}{"PROD_DB_HOST", "PROD_DB_PASS"},
				},
			},
		},
	}

	plugin := Plugin{Wait: false}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("Failed to remove temporary pipeline file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)

	// Check structure and content (map order may vary)
	assert.Contains(t, string(got), "group: deploy group")
	assert.Contains(t, string(got), "label: Deploy UAT")
	assert.Contains(t, string(got), "command: echo deploy uat")
	assert.Contains(t, string(got), "DB_HOST: uat_db_host")
	assert.Contains(t, string(got), "label: Deploy Prod")
	assert.Contains(t, string(got), "command: echo deploy prod")
	assert.Contains(t, string(got), "- PROD_DB_HOST")
	assert.Contains(t, string(got), "- PROD_DB_PASS")

	validatePipelineWithAgent(t, pipeline.Name())
}

func TestGeneratePipelineWithNotifyInGroup(t *testing.T) {
	steps := []Step{{
		Group: "Test Group",
		Steps: []Step{{
			Label:   "Run Tests",
			Command: "echo 'test'",
			Notify: []StepNotify{{
				GithubStatus: GithubStatusNotification{
					Context: "buildkite/test/status",
				},
			}},
		}},
	}}

	plugin := Plugin{}

	tmp, hasPipeline, err := generatePipeline(steps, plugin)
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)

	output := string(content)

	// Verify correct notify syntax (not rawnotify)
	assert.Contains(t, output, "notify:")
	assert.NotContains(t, output, "rawnotify")
	assert.Contains(t, output, "github_commit_status:")
	assert.Contains(t, output, "context: buildkite/test/status")

	validatePipelineWithAgent(t, tmp.Name())
}

func TestGeneratePipelineWithKeyInGroup(t *testing.T) {
	steps := []Step{{
		Group: "Test Group",
		Key:   "test-group",
		Steps: []Step{{
			Label:   "Run Tests",
			Command: "echo 'test'",
		}},
	}}

	plugin := Plugin{}

	tmp, hasPipeline, err := generatePipeline(steps, plugin)
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)

	output := string(content)
	assert.Contains(t, output, "group: Test Group")
	assert.Contains(t, output, "key: test-group")
	assert.Contains(t, output, "label: Run Tests")

	validatePipelineWithAgent(t, tmp.Name())
}

func TestGeneratePipelineWithDependsOnInGroup(t *testing.T) {
	steps := []Step{{
		Group:     "Deploy Group",
		DependsOn: "build",
		Steps: []Step{{
			Label:   "Deploy",
			Command: "echo deploy",
		}},
	}}

	tmp, hasPipeline, err := generatePipeline(steps, Plugin{})
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)
	output := string(content)

	assert.Contains(t, output, "group: Deploy Group")
	assert.Contains(t, output, "depends_on: build", "depends_on should be propagated to group step")
	validatePipelineWithAgent(t, tmp.Name())
}

func TestGeneratePipelineWithSkipInGroup(t *testing.T) {
	steps := []Step{{
		Group: "CI/CD Infrastructure",
		Key:   "group:cicd",
		Skip:  "No changes detected",
		Steps: []Step{{
			Command: "echo deploy",
		}},
	}}

	tmp, hasPipeline, err := generatePipeline(steps, Plugin{})
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)
	output := string(content)

	assert.Contains(t, output, "group: CI/CD Infrastructure")
	assert.Contains(t, output, `skip: No changes detected`, "skip should be propagated to group step")
	validatePipelineWithAgent(t, tmp.Name())
}

func TestGeneratePipelineWithSkipOnPlainStep(t *testing.T) {
	steps := []Step{{
		Command: "echo deploy",
		Skip:    "No changes detected",
	}}

	tmp, hasPipeline, err := generatePipeline(steps, Plugin{})
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)
	output := string(content)

	assert.Contains(t, output, "command: echo deploy")
	assert.Contains(t, output, `skip: No changes detected`, "skip should be propagated to a plain (non-group) step")
	validatePipelineWithAgent(t, tmp.Name())
}

func TestGeneratePipelineWithConditionInGroup(t *testing.T) {
	steps := []Step{{
		Group:     "Conditional Group",
		Condition: "build.branch == 'main'",
		Steps: []Step{{
			Label:   "Run",
			Command: "echo run",
		}},
	}}

	tmp, hasPipeline, err := generatePipeline(steps, Plugin{})
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)
	output := string(content)

	assert.Contains(t, output, "group: Conditional Group")
	assert.Contains(t, output, "if: build.branch == 'main'", "if condition value should be propagated to group step")
	validatePipelineWithAgent(t, tmp.Name())
}

func TestGeneratePipelineWithNotifyOnGroup(t *testing.T) {
	steps := []Step{{
		Group: "Notify Group",
		Notify: []StepNotify{{
			Slack: "#deployments",
		}},
		Steps: []Step{{
			Label:   "Deploy",
			Command: "echo deploy",
		}},
	}}

	tmp, hasPipeline, err := generatePipeline(steps, Plugin{})
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)
	output := string(content)

	assert.Contains(t, output, "group: Notify Group")
	assert.Contains(t, output, "notify:", "notify should be propagated to group step")
	assert.Contains(t, output, "slack: '#deployments'", "slack notify value should appear in group step")
	validatePipelineWithAgent(t, tmp.Name())
}

func TestGeneratePipelineWithAllowDependencyFailureInGroup(t *testing.T) {
	steps := []Step{{
		Group:                  "Flaky Group",
		DependsOn:              "setup",
		AllowDependencyFailure: true,
		Steps: []Step{{
			Label:   "Flaky Test",
			Command: "echo test",
		}},
	}}

	tmp, hasPipeline, err := generatePipeline(steps, Plugin{})
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)
	output := string(content)

	assert.Contains(t, output, "group: Flaky Group")
	assert.Contains(t, output, "allow_dependency_failure: true", "allow_dependency_failure should be propagated to group step")
	validatePipelineWithAgent(t, tmp.Name())
}

func TestGeneratePipelineWithDependsOnListInGroup(t *testing.T) {
	steps := []Step{{
		Group:     "Multi-dep Group",
		DependsOn: []string{"build-a", "build-b"},
		Steps: []Step{{
			Label:   "Deploy",
			Command: "echo deploy",
		}},
	}}

	tmp, hasPipeline, err := generatePipeline(steps, Plugin{})
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)
	output := string(content)

	assert.Contains(t, output, "group: Multi-dep Group")
	assert.Contains(t, output, "- build-a", "list-valued depends_on should be propagated to group step")
	assert.Contains(t, output, "- build-b", "list-valued depends_on should be propagated to group step")
	validatePipelineWithAgent(t, tmp.Name())
}

func TestGeneratePipelineWithDependsOnGroupAndNestedStep(t *testing.T) {
	steps := []Step{{
		Group:     "Deploy Group",
		DependsOn: "setup",
		Steps: []Step{{
			Label:     "Deploy",
			Command:   "echo deploy",
			DependsOn: "build",
		}},
	}}

	tmp, hasPipeline, err := generatePipeline(steps, Plugin{})
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)
	output := string(content)

	assert.Contains(t, output, "group: Deploy Group")
	assert.Equal(t, 2, strings.Count(output, "depends_on:"), "depends_on should appear on both the group and the nested step independently")
	validatePipelineWithAgent(t, tmp.Name())
}

func TestGeneratePipelineAllowDependencyFailureFalseOmitted(t *testing.T) {
	// allow_dependency_failure uses omitempty — an explicit false must not appear in output,
	// matching the same behaviour as other bool fields like async.
	steps := []Step{{
		Group:                  "Normal Group",
		DependsOn:              "build",
		AllowDependencyFailure: false,
		Steps: []Step{{
			Label:   "Deploy",
			Command: "echo deploy",
		}},
	}}

	tmp, hasPipeline, err := generatePipeline(steps, Plugin{})
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)
	output := string(content)

	assert.NotContains(t, output, "allow_dependency_failure", "allow_dependency_failure: false should be omitted from output")
}

func TestGeneratePipelineGroupAttributesNotDuplicatedOnNestedStep(t *testing.T) {
	// When Steps is nil, the step itself becomes the single nested step.
	// Group-level attributes must not also appear on that nested step.
	steps := []Step{{
		Group:                  "Deploy Group",
		Command:                "echo deploy",
		DependsOn:              "build",
		Condition:              "build.branch == 'main'",
		AllowDependencyFailure: true,
		Notify: []StepNotify{{
			Slack: "#deployments",
		}},
	}}

	tmp, hasPipeline, err := generatePipeline(steps, Plugin{})
	assert.NoError(t, err)
	assert.True(t, hasPipeline)
	defer os.Remove(tmp.Name())

	content, err := os.ReadFile(tmp.Name())
	assert.NoError(t, err)
	output := string(content)

	// Group-level attributes must appear exactly once
	assert.Equal(t, 1, strings.Count(output, "depends_on:"), "depends_on should appear once (on the group, not also on the nested step)")
	assert.Equal(t, 1, strings.Count(output, "if:"), "if should appear once (on the group, not also on the nested step)")
	assert.Equal(t, 1, strings.Count(output, "notify:"), "notify should appear once (on the group, not also on the nested step)")
	assert.Equal(t, 1, strings.Count(output, "allow_dependency_failure:"), "allow_dependency_failure should appear once (on the group, not also on the nested step)")
}

func TestGeneratePipelineWithPlugins(t *testing.T) {
	steps := []Step{
		{
			Command: "echo deploy",
			Label:   "Deploy",
			Plugins: []map[string]interface{}{
				{"docker#v5.13.0": map[string]interface{}{
					"image":   "node:20",
					"workdir": "/app",
				}},
			},
		},
		{
			Trigger: "downstream-pipeline",
			Label:   "Trigger downstream",
			Plugins: []map[string]interface{}{
				{"some-plugin#v1.0.0": map[string]interface{}{
					"setting": "value",
				}},
			},
		},
	}

	want := `steps:
    - label: Deploy
      command: echo deploy
      plugins:
        - docker#v5.13.0:
            image: node:20
            workdir: /app
    - trigger: downstream-pipeline
      label: Trigger downstream
      plugins:
        - some-plugin#v1.0.0:
            setting: value
`

	plugin := Plugin{}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("Failed to remove temporary pipeline file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)
	t.Log("Generated pipeline:\n" + string(got))
	assert.Equal(t, want, string(got))

	validatePipelineWithAgent(t, pipeline.Name())
}

func TestFilterValidSteps_AllValid(t *testing.T) {
	steps := []Step{
		{Command: "echo valid 1"},
		{Trigger: "valid-trigger"},
		{Commands: []string{"echo valid 2"}},
	}

	valid, invalid := filterValidSteps(steps)

	assert.Len(t, valid, 3)
	assert.Len(t, invalid, 0)
}

func TestFilterValidSteps_AllInvalid(t *testing.T) {
	steps := []Step{
		{},
		{Label: "no command"},
		{Env: map[string]string{"KEY": "value"}},
	}

	valid, invalid := filterValidSteps(steps)

	assert.Len(t, valid, 0)
	assert.Len(t, invalid, 3)
}

func TestFilterValidSteps_MixedValidInvalid(t *testing.T) {
	steps := []Step{
		{Command: "echo valid"},
		{},
		{Trigger: "valid-trigger"},
		{Label: "invalid - no command/trigger"},
		{Commands: []string{"echo also valid"}},
	}

	valid, invalid := filterValidSteps(steps)

	assert.Len(t, valid, 3)
	assert.Len(t, invalid, 2)
	assert.Equal(t, "echo valid", valid[0].Command)
	assert.Equal(t, "valid-trigger", valid[1].Trigger)
	assert.NotNil(t, valid[2].Commands)
}

func TestFilterValidSteps_GroupSteps(t *testing.T) {
	steps := []Step{
		{
			Group: "valid-group",
			Steps: []Step{
				{Command: "echo test"},
			},
		},
		{
			Group: "empty-group",
			Steps: []Step{},
		},
		{
			Group: "invalid-nested-group",
			Steps: []Step{
				{Label: "no command"},
			},
		},
	}

	valid, invalid := filterValidSteps(steps)

	assert.Len(t, valid, 1)
	assert.Len(t, invalid, 2)
	assert.Equal(t, "valid-group", valid[0].Group)
}

func TestStepsToTrigger_Issue83(t *testing.T) {
	// Integration test for issue #83 - empty step configuration
	watch := []WatchConfig{
		{
			Paths: []string{"some-path/**"},
			Step:  Step{}, // Empty step - should be filtered
		},
		{
			Paths: []string{"other-path/**"},
			Step:  Step{Command: "echo valid"},
		},
	}

	changedFiles := []string{
		"some-path/file.txt",
		"other-path/file.txt",
	}

	steps, err := stepsToTrigger(changedFiles, watch, false)

	assert.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, "echo valid", steps[0].Command)
}

func TestStepsToTrigger_DefaultWithEmptyStep(t *testing.T) {
	// Test that empty default step is filtered
	watch := []WatchConfig{
		{
			Paths: []string{"app/"},
			Step:  Step{Command: "echo app"},
		},
		{
			Default: struct{}{},
			Step:    Step{}, // Empty default - should be filtered
		},
	}

	changedFiles := []string{"unmatched/file.txt"}

	steps, err := stepsToTrigger(changedFiles, watch, false)

	assert.NoError(t, err)
	assert.Len(t, steps, 0)
}

func TestStepsToTrigger_AllStepsInvalid(t *testing.T) {
	// Test that when all matched steps are invalid, empty array is returned
	watch := []WatchConfig{
		{
			Paths: []string{"path1/"},
			Step:  Step{},
		},
		{
			Paths: []string{"path2/"},
			Step:  Step{Label: "no command"},
		},
	}

	changedFiles := []string{"path1/file.txt", "path2/file.txt"}

	steps, err := stepsToTrigger(changedFiles, watch, false)

	assert.NoError(t, err)
	assert.Len(t, steps, 0)
}

func TestStepsToTrigger_EmptyGroupInConfig(t *testing.T) {
	// Test that empty group steps are filtered
	watch := []WatchConfig{
		{
			Paths: []string{"services/"},
			Step: Step{
				Group: "deploy",
				Steps: []Step{}, // Empty group
			},
		},
	}

	changedFiles := []string{"services/main.go"}

	steps, err := stepsToTrigger(changedFiles, watch, false)

	assert.NoError(t, err)
	assert.Len(t, steps, 0)
}

func TestStepsToTrigger_ValidAndInvalidStepsMixed(t *testing.T) {
	// Test that valid steps are kept while invalid are filtered
	watch := []WatchConfig{
		{
			Paths: []string{"app/"},
			Step:  Step{Command: "echo valid app"},
		},
		{
			Paths: []string{"tests/"},
			Step:  Step{}, // Invalid
		},
		{
			Paths: []string{"deploy/"},
			Step:  Step{Trigger: "deploy-pipeline"},
		},
	}

	changedFiles := []string{
		"app/main.go",
		"tests/test.go",
		"deploy/script.sh",
	}

	steps, err := stepsToTrigger(changedFiles, watch, false)

	assert.NoError(t, err)
	assert.Len(t, steps, 2)

	// Verify we got the valid steps
	hasAppStep := false
	hasDeployStep := false
	for _, step := range steps {
		if step.Command == "echo valid app" {
			hasAppStep = true
		}
		if step.Trigger == "deploy-pipeline" {
			hasDeployStep = true
		}
	}
	assert.True(t, hasAppStep)
	assert.True(t, hasDeployStep)
}

func TestGeneratePipelineWithRetry(t *testing.T) {
	steps := []Step{
		{
			Command: "echo deploy",
			Label:   "Deploy",
			Retry: map[string]interface{}{
				"automatic": []interface{}{
					map[string]interface{}{"exit_status": -1, "limit": 2},
					map[string]interface{}{"exit_status": 143, "limit": 2, "signal_reason": "agent_stop"},
				},
				"manual": map[string]interface{}{
					"allowed":          true,
					"reason":           "Retry after investigating",
					"permit_on_passed": false,
				},
			},
		},
	}

	plugin := Plugin{Wait: false}

	pipeline, _, err := generatePipeline(steps, plugin)
	require.NoError(t, err)
	defer func() {
		if err = os.Remove(pipeline.Name()); err != nil {
			t.Logf("Failed to remove temporary pipeline file: %v", err)
		}
	}()

	got, err := os.ReadFile(pipeline.Name())
	require.NoError(t, err)

	t.Log("Generated pipeline:\n" + string(got))

	// Verify key elements are present (map ordering may vary)
	assert.Contains(t, string(got), "label: Deploy")
	assert.Contains(t, string(got), "command: echo deploy")
	assert.Contains(t, string(got), "retry:")
	assert.Contains(t, string(got), "automatic:")
	assert.Contains(t, string(got), "exit_status: -1")
	assert.Contains(t, string(got), "limit: 2")
	assert.Contains(t, string(got), "exit_status: 143")
	assert.Contains(t, string(got), "signal_reason: agent_stop")
	assert.Contains(t, string(got), "manual:")
	assert.Contains(t, string(got), "allowed: true")
	assert.Contains(t, string(got), "reason: Retry after investigating")
	assert.Contains(t, string(got), "permit_on_passed: false")

	validatePipelineWithAgent(t, pipeline.Name())
}
