package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/buildkite/bintest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	realAgent, err := exec.LookPath("buildkite-agent")
	if err != nil {
		t.Skip("real buildkite-agent not installed")
	}

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

	// --- NEW: validate pipeline YAML via real agent ---
	dryRun := exec.Command(
		realAgent,
		"pipeline",
		"upload",
		"pipeline.txt",
		"--dry-run",
		"--agent-access-token",
		"dummy",
	)

	out, err := dryRun.CombinedOutput()
	t.Log(string(out))
	require.NoError(t, err, "Buildkite rejected generated YAML")
}

func TestUploadPipelineCallsBuildkiteAgentCommandWithInterpolation(t *testing.T) {
	realAgent, err := exec.LookPath("buildkite-agent")
	if err != nil {
		t.Skip("buildkite-agent not installed; skipping dry-run validation")
	}

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

	// --- NEW: real Buildkite validation ---
	dryRun := exec.Command(
		realAgent,
		"pipeline",
		"upload",
		"pipeline.txt",
		"--dry-run",
		"--agent-access-token",
		"dummy",
	)

	out, err := dryRun.CombinedOutput()
	t.Log(string(out))
	require.NoError(t, err, "Buildkite rejected generated YAML")
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

	got, err := diff(`echo services/foo/serverless.yml
services/bar/config.yml

ops/bar/config.yml
README.md`)

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffWithSubshell(t *testing.T) {
	want := []string{
		"user-service/infrastructure/cloudfront.yaml",
		"user-service/serverless.yaml",
	}
	got, err := diff("echo $(cat e2e/multiple-paths)")
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDiffWithQuotedPaths(t *testing.T) {
	want := []string{
		"projects/test/pages/17_🪁_testfile.py",
		"normal/file.txt",
	}
	got, err := diff(`printf '"projects/test/pages/17_\360\237\252\201_testfile.py" normal/file.txt'`)
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

	steps, err := stepsToTrigger(changedFiles, watch)
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

	pipelines, err := stepsToTrigger(changedFiles, watch)
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
			steps, err := stepsToTrigger(tc.ChangedFiles, tc.WatchConfigs)
			assert.NoError(t, err)
			assert.Equal(t, tc.Expected, steps)
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

	assert.Equal(t, want, string(got))

	// --- New: validate generated pipeline with buildkite-agent dry-run ---
	if _, err := exec.LookPath("buildkite-agent"); err == nil {
		cmd := exec.Command(
			"buildkite-agent",
			"pipeline", "upload",
			pipeline.Name(),
			"--dry-run",
			"--agent-access-token", "dummy",
		)
		out, err := cmd.CombinedOutput()
		t.Log(string(out))
		require.NoError(t, err, "Buildkite rejected generated YAML")
	} else {
		t.Log("buildkite-agent not installed; skipping dry-run validation")
	}
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
	assert.Equal(t, want, string(got))

	// --- New: validate generated pipeline with buildkite-agent dry-run ---
	if _, err := exec.LookPath("buildkite-agent"); err == nil {
		cmd := exec.Command(
			"buildkite-agent",
			"pipeline", "upload",
			pipeline.Name(),
			"--dry-run",
			"--agent-access-token", "dummy",
		)
		out, err := cmd.CombinedOutput()
		t.Log(string(out))
		require.NoError(t, err, "Buildkite rejected generated YAML")
	} else {
		t.Log("buildkite-agent not installed; skipping dry-run validation")
	}
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
	assert.Equal(t, want, string(got))

	// --- New: validate generated pipeline with buildkite-agent dry-run ---
	if _, err := exec.LookPath("buildkite-agent"); err == nil {
		cmd := exec.Command(
			"buildkite-agent",
			"pipeline", "upload",
			pipeline.Name(),
			"--dry-run",
			"--agent-access-token", "dummy",
		)
		out, err := cmd.CombinedOutput()
		t.Log(string(out))
		require.NoError(t, err, "Buildkite rejected generated YAML")
	} else {
		t.Log("buildkite-agent not installed; skipping dry-run validation")
	}
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
	assert.Equal(t, want, string(got))

	// --- New: validate pipeline YAML with buildkite-agent dry-run ---
	if _, err := exec.LookPath("buildkite-agent"); err == nil {
		cmd := exec.Command(
			"buildkite-agent",
			"pipeline", "upload",
			pipeline.Name(),
			"--dry-run",
			"--agent-access-token", "dummy",
		)
		out, err := cmd.CombinedOutput()
		t.Log(string(out))
		require.NoError(t, err, "Buildkite rejected generated YAML")
	} else {
		t.Log("buildkite-agent not installed; skipping dry-run validation")
	}
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
	assert.Equal(t, want, string(got))

	// --- New: validate pipeline YAML with buildkite-agent dry-run ---
	if _, err := exec.LookPath("buildkite-agent"); err == nil {
		cmd := exec.Command(
			"buildkite-agent",
			"pipeline", "upload",
			pipeline.Name(),
			"--dry-run",
			"--agent-access-token", "dummy",
		)
		out, err := cmd.CombinedOutput()
		t.Log(string(out))
		require.NoError(t, err, "Buildkite rejected generated YAML")
	} else {
		t.Log("buildkite-agent not installed; skipping dry-run validation")
	}
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

	assert.Equal(t, want, string(got))
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

	assert.Equal(t, want, string(got))
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

	steps, err := stepsToTrigger(changedFiles, watch)

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

	steps, err := stepsToTrigger(changedFiles, watch)

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

	steps, err := stepsToTrigger(changedFiles, watch)

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

	steps, err := stepsToTrigger(changedFiles, watch)

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

	steps, err := stepsToTrigger(changedFiles, watch)

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
