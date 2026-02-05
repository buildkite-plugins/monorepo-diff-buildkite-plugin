package main

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func defaultPlugin() Plugin {
	return Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
	}
}

func defaultPluginWithDefault() Plugin {
	ret := defaultPlugin()
	ret.Watch = []WatchConfig{
		{
			Paths: []string{".buildkite/**/*"},
			Step: Step{
				Command: "echo hello world",
				Label:   "Example label",
			},
		},
		{
			Default: true,
			Paths:   []string{},
			Step: Step{
				Command: "echo default hello world",
				Label:   "Default label",
			},
		},
	}

	return ret
}

func TestPluginWithEmptyParameter(t *testing.T) {
	_, err := initializePlugin("[]")

	assert.EqualError(t, err, "could not initialize plugin")
}

func TestPluginWithInvalidParameter(t *testing.T) {
	_, err := initializePlugin("invalid")

	assert.EqualError(t, err, "failed to parse plugin configuration")
}

func TestPluginShouldHaveDefaultValues(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {}
	}]`

	got, _ := initializePlugin(param)

	assert.Equal(t, defaultPlugin(), got)
}

func TestPluginWithEmptyStringParameter(t *testing.T) {
	param := ""
	got, err := initializePlugin(param)
	expected := Plugin{}

	assert.EqualError(t, err, "failed to parse plugin configuration")
	assert.Equal(t, expected, got)
}

func TestPluginShouldUnmarshallCorrectly(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"diff": "cat ./hello.txt",
			"wait": true,
			"log_level": "debug",
			"interpolation": true,
			"hooks": [
				{ "command": "some-hook-command" },
				{ "command": "another-hook-command" }
			],
			"env": [
				"env1=env-1",
				"env2=env-2",
				"env3"
			],
		"notify": [
				{ "email": "foo@gmail.com" },
				{ "email": "bar@gmail.com" },
				{ "basecamp_campfire": "https://basecamp-url" },
				{ "webhook": "https://webhook-url", "if": "build.state === 'failed'" },
				{ "pagerduty_change_event": "636d22Yourc0418Key3b49eee3e8" },
				{ "github_commit_status": { "context" : "my-custom-status" } },
				{ "slack": "@someuser", "if": "build.state === 'passed'" }
			],
			"watch": [
				{
					"path": "watch-path-1",
					"config": {
						"trigger": "service-2",
						"build": {
							"message": "some message"
						}
					}
				},
				{
					"path": "watch-path-1",
					"config": {
						"command": "echo hello-world",
						"env": [
							"env4", "hi= bye"
						],
						"soft_fail": [{
							"exit_status": "*"
						}],
						"notify": [
							{ "email": "foo@gmail.com" },
							{ "email": "bar@gmail.com" },
							{ "basecamp_campfire": "https://basecamp-url" },
							{ "webhook": "https://webhook-url", "if": "build.state === 'failed'" },
							{ "pagerduty_change_event": "636d22Yourc0418Key3b49eee3e8" },
							{ "github_commit_status": { "context" : "my-custom-status" } },
							{ "slack": "@someuser", "if": "build.state === 'passed'" }
						]
					}
				},
				{
					"path": [
						"watch-path-1",
						"watch-path-2"
					],
					"config": {
						"trigger": "service-1",
						"label": "hello",
						"build": {
							"message": "build message",
							"branch": "current branch",
							"commit": "commit-hash",
							"meta_data": {
								"metadata1": "metadata-1",
								"metadata2": "metadata-2",
								"metadata3": "metadata-3"
							},
							"env": [
								"foo =bar",
								"bar= foo"
							]
						},
						"async": true,
						"agents": {
							"queue": "queue-1",
							"database": "postgres"
						},
						"artifacts": [ "artifact-1" ],
						"soft_fail": [{
							"exit_status": 127
						}]
					}
				},
				{
					"path": "watch-path-1",
					"config": {
						"group": "my group",
						"command": "echo hello-group",
						"soft_fail": true
					}
				},
				{
					"path": "watch-path-3",
					"config": {
						"group": "my group",
						"steps": [
							{ "command": "echo hello-group from first step" },
							{ "command": "echo hello-group from second step" }
						]
					}
				}
			]
		}
	}]`

	got, _ := initializePlugin(param)

	expected := Plugin{
		Diff:          "cat ./hello.txt",
		Wait:          true,
		LogLevel:      "debug",
		Interpolation: true,
		Hooks: []HookConfig{
			{Command: "some-hook-command"},
			{Command: "another-hook-command"},
		},
		Env: map[string]string{
			"env1": "env-1",
			"env2": "env-2",
			"env3": "env-3",
		},
		Notify: []PluginNotify{
			{Email: "foo@gmail.com"},
			{Email: "bar@gmail.com"},
			{Basecamp: "https://basecamp-url"},
			{Webhook: "https://webhook-url", Condition: "build.state === 'failed'"},
			{PagerDuty: "636d22Yourc0418Key3b49eee3e8"},
			{GithubStatus: GithubStatusNotification{Context: "my-custom-status"}},
			{Slack: "@someuser", Condition: "build.state === 'passed'"},
		},
		Watch: []WatchConfig{
			{
				Paths: []string{"watch-path-1"},
				Step: Step{
					Trigger: "service-2",
					Build: Build{
						Message: "some message",
						Branch:  "go-rewrite",
						Commit:  "123",

						Env: map[string]string{
							"env1": "env-1",
							"env2": "env-2",
							"env3": "env-3",
						},
					},
				},
			},
			{
				Paths: []string{"watch-path-1"},
				Step: Step{
					Command: "echo hello-world",
					Env: map[string]string{
						"env1": "env-1",
						"env2": "env-2",
						"env3": "env-3",
						"env4": "env-4",
						"hi":   "bye",
					},
					SoftFail: []interface{}{map[string]interface{}{"exit_status": "*"}},
					Notify: []StepNotify{
						{Basecamp: "https://basecamp-url"},
						{GithubStatus: GithubStatusNotification{Context: "my-custom-status"}},
						{Slack: "@someuser", Condition: "build.state === 'passed'"},
					},
				},
			},
			{
				Paths: []string{"watch-path-1", "watch-path-2"},
				Step: Step{
					Trigger: "service-1",
					Label:   "hello",
					Build: Build{
						Message: "build message",
						Branch:  "current branch",
						Commit:  "commit-hash",
						Metadata: map[string]string{
							"metadata1": "metadata-1",
							"metadata2": "metadata-2",
							"metadata3": "metadata-3",
						},
						Env: map[string]string{
							"foo":  "bar",
							"bar":  "foo",
							"env1": "env-1",
							"env2": "env-2",
							"env3": "env-3",
						},
					},
					Async:         true,
					Agents:        map[string]string{"queue": "queue-1", "database": "postgres"},
					ArtifactPaths: []string{"artifact-1"},
					SoftFail: []interface{}{map[string]interface{}{
						"exit_status": float64(127),
					}},
				},
			},
			{
				Paths: []string{"watch-path-1"},
				Step: Step{
					Group:   "my group",
					Command: "echo hello-group",
					Env: map[string]string{
						"env1": "env-1",
						"env2": "env-2",
						"env3": "env-3",
					},
					SoftFail: true,
				},
			},
			{
				Paths: []string{"watch-path-3"},
				Step: Step{
					Group: "my group",
					Steps: []Step{
						{
							Command: "echo hello-group from first step",
							Env: map[string]string{
								"env1": "env-1",
								"env2": "env-2",
								"env3": "env-3",
							},
						},
						{
							Command: "echo hello-group from second step",
							Env: map[string]string{
								"env1": "env-1",
								"env2": "env-2",
								"env3": "env-3",
							},
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got): \n%s", diff)
	}
}

func TestPluginShouldOnlyFullyUnmarshallItselfAndNotOtherPlugins(t *testing.T) {
	param := `[
		{
			"github.com/example/example-plugin#commit": {
				"env": {
					"EXAMPLE_TOKEN": {
						"json-key": ".TOKEN",
						"secret-id": "global/example/token"
					}
				}
			}
		},
		{
			"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": { }
		}
	]
	`
	got, _ := initializePlugin(param)
	assert.Equal(t, defaultPlugin(), got)
}

func TestPluginShouldErrorIfPluginConfigIsInvalid(t *testing.T) {
	// Test that truly invalid env format (string) returns error
	param := `[
		{
			"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
				"env": "invalid-string-format",
				"watch": [
					{
						"path": [
							".buildkite/**/*"
						],
						"config": {
							"label": "Example label",
							"command": "echo hello world\\n"
						}
					}
				]
			}
		}
	]
	`
	_, err := initializePlugin(param)
	assert.Error(t, err)
}

func TestPluginFullDifferentOrg(t *testing.T) {
	param := `[{
		"github.com/random-org/monorepo-diff-buildkite-plugin#commit": {}
	}]`

	got, _ := initializePlugin(param)
	assert.Equal(t, defaultPlugin(), got)
}

func TestPluginRawReference(t *testing.T) {
	param := `[{
		"monorepo-diff#v1.2": {}
	}]`

	got, _ := initializePlugin(param)
	assert.Equal(t, defaultPlugin(), got)
}

func TestPluginOrgRawReference(t *testing.T) {
	param := `[{
		"random-org/monorepo-diff-buildkite-plugin#commit": {}
	}]`

	got, _ := initializePlugin(param)
	assert.Equal(t, defaultPlugin(), got)
}

func TestPluginInvalidReference(t *testing.T) {
	param := `[{
		":invalid/monorepo-diff#v1.2": {}
	}]`

	_, err := initializePlugin(param)
	assert.Error(t, err)
}

func TestPluginDefaultCommand(t *testing.T) {
	param := `[
		{
			"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
				"watch": [
					{
						"path": [
							".buildkite/**/*"
						],
						"config": {
							"label": "Example label",
							"command": "echo hello world"
						}
					}, {
						"default": {
							"label": "Default label",
							"command": "echo default hello world"
						}
					}
				]
			}
		}
	]
	`

	got, err := initializePlugin(param)
	assert.NoError(t, err)
	assert.Equal(t, defaultPluginWithDefault(), got)
}

func TestPluginDefaultConfigCommand(t *testing.T) {
	param := `[
		{
			"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
				"watch": [
					{
						"path": [
							".buildkite/**/*"
						],
						"config": {
							"label": "Example label",
							"command": "echo hello world"
						}
					}, {
						"default": {
							"config": {
								"label": "Default label",
								"command": "echo default hello world"
							}
						}
					}
				]
			}
		}
	]
	`

	got, err := initializePlugin(param)
	assert.NoError(t, err)
	assert.Equal(t, defaultPluginWithDefault(), got)
}

func TestPluginWithoutWaitProperty(t *testing.T) {
	param := `[
		{
			"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
				"watch": [
					{
						"path": [
							".buildkite/**/*"
							],
						"config": {
							"label": "Example label",
							"command": "echo hello world"
						}
					}, {
						"default": {
							"config": {
								"label": "Default label",
								"command": "echo default hello world"
							}
						}
					}
				]
			}
		}
	]
	`

	got, err := initializePlugin(param)
	assert.NoError(t, err)
	assert.Equal(t, defaultPluginWithDefault(), got)
	assert.False(t, got.Wait)
}

func TestPluginWithBuildConfigFromEnv(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": ".buildkite/**/*",
					"config": {
						"trigger": "foo-service"
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	fmt.Print(err)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Watch: []WatchConfig{
			{
				Paths: []string{".buildkite/**/*"},
				Step: Step{
					Trigger: "foo-service",
					Build: Build{
						Message: "fix: temp file not correctly deleted",
						Branch:  "go-rewrite",
						Commit:  "123",
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got): \n%s", diff)
	}
}

func TestPluginWithMultiplePluginVersions(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#v1.2.0": {
			"diff":"echo foo-service/",
				"log_level": "debug",
				"watch": [
					{
						"path":"foo-service/",
						"config": {
							"trigger":"foo-service"
						}
					},
					{
						"path":"bar-service/",
						"config": {
							"trigger":"foo-service"
						}
					}
				]
			}
		},{
			"github.com/buildkite-plugins/kubernetes-buildkite-plugin": {
				"checkout": {
					"gitCredentialsSecret": {
						"secretName": "csb-playground-buildkite"
					}
				},
				"podSpecPatch": {
					"volumes": [{
						"name": "git-credentials-ro",
						"secret": {
							"secretName": "csb-playground-buildkite",
							"defaultMode": 420
						}
					}],
					"containers": [{
						"env": [{
							"name": "BUILDKITE_SHELL",
							"value": "/bin/bash -ec"
						}],
						"name": "container-0",
						"image": "alpine:latest"
					}]
				}
			}
	},{
		"github.com/buildkite-plugins/cache-buildkite-plugin#v1.5.1": {
			"path": ".cache/pip",
			"save": "file",
			"backend": "s3",
			"restore": "file",
			"manifest": ".pre-commit-config.yaml",
			"compression": "tgz"
		}
	}]`

	got, err := initializePlugin(param)
	fmt.Print(err)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "echo foo-service/",
		Wait:          false,
		LogLevel:      "debug",
		Interpolation: true,
		Watch: []WatchConfig{
			{
				Paths: []string{"foo-service/"},
				Step: Step{
					Trigger: "foo-service",
					Build: Build{
						Message: "fix: temp file not correctly deleted",
						Branch:  "go-rewrite",
						Commit:  "123",
					},
				},
			},
			{
				Paths: []string{"bar-service/"},
				Step: Step{
					Trigger: "foo-service",
					Build: Build{
						Message: "fix: temp file not correctly deleted",
						Branch:  "go-rewrite",
						Commit:  "123",
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got): \n%s", diff)
	}
}

func TestPluginShouldPreserveStepPlugins(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": ".buildkite/**/*",
					"config": {
						"plugins": [
							{ "some-plugin#v1": { "foo": "bar" } }
						]
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	fmt.Print(err)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Watch: []WatchConfig{
			{
				Paths: []string{".buildkite/**/*"},
				Step: Step{
					Plugins: []map[string]interface{}{
						{"some-plugin#v1": map[string]interface{}{"foo": "bar"}},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got):\n%s", diff)
	}
}

func TestPluginShouldPreserveStepBranches(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": ".buildkite/**/*",
					"config": {
						"branches": "!main feature/*"
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	fmt.Print(err)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Watch: []WatchConfig{
			{
				Paths: []string{".buildkite/**/*"},
				Step: Step{
					Branches: "!main feature/*",
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got):\n%s", diff)
	}
}

func TestPluginMetadataOnlyAppliedToTriggerSteps(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"meta_data": {
				"plugin_level_key": "plugin_level_value"
			},
			"watch": [
				{
					"path": "app/",
					"config": {
						"trigger": "app-deploy",
						"build": {
							"meta_data": {
								"step_level_key": "step_level_value"
							}
						}
					}
				},
				{
					"path": "test/",
					"config": {
						"command": "echo test command"
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Metadata: map[string]string{
			"plugin_level_key": "plugin_level_value",
		},
		Watch: []WatchConfig{
			{
				Paths: []string{"app/"},
				Step: Step{
					Trigger: "app-deploy",
					Build: Build{
						Message: "fix: temp file not correctly deleted",
						Branch:  "go-rewrite",
						Commit:  "123",
						Metadata: map[string]string{
							"step_level_key":   "step_level_value",
							"plugin_level_key": "plugin_level_value",
						},
					},
				},
			},
			{
				Paths: []string{"test/"},
				Step: Step{
					Command: "echo test command",
					// Build defaults are not set for command steps
					Build: Build{
						// No metadata should be set here
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got):\n%s", diff)
	}

	// Explicitly verify that command step doesn't have metadata
	commandStep := got.Watch[1].Step
	assert.Nil(t, commandStep.Build.Metadata, "Command step should not have metadata set")

	// Explicitly verify that trigger step has metadata
	triggerStep := got.Watch[0].Step
	assert.NotNil(t, triggerStep.Build.Metadata, "Trigger step should have metadata set")
	assert.Equal(t, "step_level_value", triggerStep.Build.Metadata["step_level_key"])
	assert.Equal(t, "plugin_level_value", triggerStep.Build.Metadata["plugin_level_key"])
}

func TestPluginLevelMetadataNotAppliedToCommandSteps(t *testing.T) {
	// This test demonstrates the fix for the original issue where plugin-level
	// metadata was being applied to all steps, including command steps where it doesn't belong
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"meta_data": {
				"foo": "bar"
			},
			"watch": [
				{
					"path": "app/",
					"config": {
						"trigger": "app-deploy"
					}
				},
				{
					"path": "test/bin/",
					"config": {
						"command": "echo Make Changes to Bin"
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	// Verify that trigger step gets plugin-level metadata
	triggerStep := got.Watch[0].Step
	assert.Equal(t, "app-deploy", triggerStep.Trigger)
	assert.NotNil(t, triggerStep.Build.Metadata)
	assert.Equal(t, "bar", triggerStep.Build.Metadata["foo"])

	// Verify that command step does NOT get plugin-level metadata
	commandStep := got.Watch[1].Step
	assert.Equal(t, "echo Make Changes to Bin", commandStep.Command)
	assert.Nil(t, commandStep.Build.Metadata, "Command step should not have metadata applied")
}

func TestPluginShouldPreserveStepCondition(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "service/**/*",
					"config": {
						"command": "echo deploy",
						"if": "build.branch == 'main' && build.pull_request.id == null"
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Watch: []WatchConfig{
			{
				Paths: []string{"service/**/*"},
				Step: Step{
					Command:   "echo deploy",
					Condition: "build.branch == 'main' && build.pull_request.id == null",
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got):\n%s", diff)
	}
}

func TestPluginShouldClearRawEnvFromNestedSteps(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"env": [
				"PLUGIN_ENV=plugin-value"
			],
			"watch": [
				{
					"path": "yarn.lock",
					"config": {
						"group": "test-group",
						"steps": [
							{
								"trigger": "test-pipeline",
								"label": "PR build",
								"build": {
									"commit": "abc123",
									"branch": "feature-branch",
									"env": [
										"BUNDLE_BUILD=pr"
									]
								}
							},
							{
								"trigger": "test-pipeline",
								"label": "Master build",
								"build": {
									"branch": "master",
									"env": [
										"BUNDLE_BUILD=master"
									]
								}
							}
						]
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	// Verify the structure is correct
	assert.Equal(t, 1, len(got.Watch))
	assert.Equal(t, "test-group", got.Watch[0].Step.Group)
	assert.Equal(t, 2, len(got.Watch[0].Step.Steps))

	// Verify nested steps have correct env values
	firstStep := got.Watch[0].Step.Steps[0]
	assert.Equal(t, "test-pipeline", firstStep.Trigger)
	assert.Equal(t, "pr", firstStep.Build.Env["BUNDLE_BUILD"])
	assert.Equal(t, "plugin-value", firstStep.Build.Env["PLUGIN_ENV"])

	secondStep := got.Watch[0].Step.Steps[1]
	assert.Equal(t, "test-pipeline", secondStep.Trigger)
	assert.Equal(t, "master", secondStep.Build.Env["BUNDLE_BUILD"])
	assert.Equal(t, "plugin-value", secondStep.Build.Env["PLUGIN_ENV"])

	// Most importantly: verify RawEnv fields are cleared
	assert.Nil(t, firstStep.RawEnv, "First nested step RawEnv should be nil")
	assert.Nil(t, firstStep.Build.RawEnv, "First nested step Build.RawEnv should be nil")
	assert.Nil(t, secondStep.RawEnv, "Second nested step RawEnv should be nil")
	assert.Nil(t, secondStep.Build.RawEnv, "Second nested step Build.RawEnv should be nil")
}

func TestPluginShouldPreserveDependsOnString(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "service/**/*",
					"config": {
						"command": "echo deploy",
						"depends_on": "build-step"
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Watch: []WatchConfig{
			{
				Paths: []string{"service/**/*"},
				Step: Step{
					Command:   "echo deploy",
					DependsOn: "build-step",
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got):\n%s", diff)
	}
}

func TestPluginShouldPreserveDependsOnArray(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "service/**/*",
					"config": {
						"trigger": "deploy-pipeline",
						"depends_on": ["build-step", "test-step"]
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Watch: []WatchConfig{
			{
				Paths: []string{"service/**/*"},
				Step: Step{
					Trigger: "deploy-pipeline",
					Build: Build{
						Message: "fix: temp file not correctly deleted",
						Branch:  "go-rewrite",
						Commit:  "123",
					},
					DependsOn: []interface{}{"build-step", "test-step"},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got):\n%s", diff)
	}
}

func TestPluginEnvWithEqualsSignsAndSpacesInValues(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"env": [
				"EXTRA_BUILD_ARGS=--build-arg=ARG1=value1",
				"QUOTED_ARGS=\"--build-arg=ARG1=value1\"",
				"SPACE_VALUE=value with spaces",
				"COMPLEX=\"--opt1=val1 --opt2=val2\""
			],
			"watch": [
				{
					"path": "services/",
					"config": {
						"command": "echo test"
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Env: map[string]string{
			"EXTRA_BUILD_ARGS": "--build-arg=ARG1=value1",
			"QUOTED_ARGS":      "\"--build-arg=ARG1=value1\"",
			"SPACE_VALUE":      "value with spaces",
			"COMPLEX":          "\"--opt1=val1 --opt2=val2\"",
		},
		Watch: []WatchConfig{
			{
				Paths: []string{"services/"},
				Step: Step{
					Command: "echo test",
					Env: map[string]string{
						"EXTRA_BUILD_ARGS": "--build-arg=ARG1=value1",
						"QUOTED_ARGS":      "\"--build-arg=ARG1=value1\"",
						"SPACE_VALUE":      "value with spaces",
						"COMPLEX":          "\"--opt1=val1 --opt2=val2\"",
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got):\n%s", diff)
	}

	// Explicitly verify the env values are correct
	assert.Equal(t, "--build-arg=ARG1=value1", got.Env["EXTRA_BUILD_ARGS"])
	assert.Equal(t, "\"--build-arg=ARG1=value1\"", got.Env["QUOTED_ARGS"])
	assert.Equal(t, "value with spaces", got.Env["SPACE_VALUE"])
	assert.Equal(t, "\"--opt1=val1 --opt2=val2\"", got.Env["COMPLEX"])
}

func TestPluginShouldPreserveSecretsAsMap(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "service/**/*",
					"config": {
						"command": "echo deploy",
						"secrets": {
							"DATABRICKS_HOST": "databricks_host_secret",
							"DATABRICKS_TOKEN": "databricks_token_secret"
						}
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Watch: []WatchConfig{
			{
				Paths: []string{"service/**/*"},
				Step: Step{
					Command: "echo deploy",
					Secrets: map[string]interface{}{
						"DATABRICKS_HOST":  "databricks_host_secret",
						"DATABRICKS_TOKEN": "databricks_token_secret",
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got):\n%s", diff)
	}
}

func TestPluginShouldPreserveSecretsAsArray(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "service/**/*",
					"config": {
						"command": "echo deploy",
						"secrets": ["API_ACCESS_TOKEN", "DATABASE_PASSWORD"]
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Watch: []WatchConfig{
			{
				Paths: []string{"service/**/*"},
				Step: Step{
					Command: "echo deploy",
					Secrets: []interface{}{"API_ACCESS_TOKEN", "DATABASE_PASSWORD"},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got):\n%s", diff)
	}
}

func TestPluginShouldPreserveSecretsInNestedSteps(t *testing.T) {
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "service/**/*",
					"config": {
						"group": "deploy group",
						"steps": [
							{
								"command": "echo deploy uat",
								"label": "Deploy UAT",
								"secrets": {
									"DB_HOST": "uat_db_host"
								}
							},
							{
								"command": "echo deploy prod",
								"label": "Deploy Prod",
								"secrets": ["PROD_DB_HOST", "PROD_DB_PASS"]
							}
						]
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(got.Watch))
	assert.Equal(t, "deploy group", got.Watch[0].Step.Group)
	assert.Equal(t, 2, len(got.Watch[0].Step.Steps))

	// Verify first nested step has secrets as map
	firstStep := got.Watch[0].Step.Steps[0]
	assert.Equal(t, "echo deploy uat", firstStep.Command)
	secretsMap, ok := firstStep.Secrets.(map[string]interface{})
	assert.True(t, ok, "first step secrets should be a map")
	assert.Equal(t, "uat_db_host", secretsMap["DB_HOST"])

	// Verify second nested step has secrets as array
	secondStep := got.Watch[0].Step.Steps[1]
	assert.Equal(t, "echo deploy prod", secondStep.Command)
	secretsArray, ok := secondStep.Secrets.([]interface{})
	assert.True(t, ok, "second step secrets should be an array")
	assert.Equal(t, []interface{}{"PROD_DB_HOST", "PROD_DB_PASS"}, secretsArray)
}

func TestPluginAcceptsArtifactPathsFieldName(t *testing.T) {
	// Test that artifact_paths field name works
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "service/**/*",
					"config": {
						"command": "echo test",
						"artifact_paths": ["logs/**/*", "coverage/**/*"]
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)
	assert.NoError(t, err)

	expected := Plugin{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
		Watch: []WatchConfig{
			{
				Paths: []string{"service/**/*"},
				Step: Step{
					Command:       "echo test",
					ArtifactPaths: []string{"logs/**/*", "coverage/**/*"},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatalf("plugin diff (-want +got):\n%s", diff)
	}
}

func TestPluginRejectsBothArtifactsFields(t *testing.T) {
	// Test that specifying both fields returns an error
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "service/**/*",
					"config": {
						"command": "echo test",
						"artifacts": ["old.log"],
						"artifact_paths": ["new.log"]
					}
				}
			]
		}
	}]`

	_, err := initializePlugin(param)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify both 'artifacts' and 'artifact_paths'")
}
// Unit tests for parseEnv function

func TestParseEnvMapStringFormat(t *testing.T) {
	// Test direct string map with explicit values
	input := map[string]string{
		"NODE_ENV": "production",
		"API_URL":  "https://api.example.com",
		"PORT":     "8080",
	}

	result, err := parseEnv(input)

	assert.NoError(t, err)
	assert.Equal(t, "production", result["NODE_ENV"])
	assert.Equal(t, "https://api.example.com", result["API_URL"])
	assert.Equal(t, "8080", result["PORT"])
	assert.Len(t, result, 3)
}

func TestParseEnvMapOSEnvReading(t *testing.T) {
	// Test only null values read from OS env
	t.Setenv("TEST_VAR", "from-env")

	input := map[string]interface{}{
		"TEST_VAR": nil, // Only null reads from OS environment
		"EXPLICIT": "explicit-value",
	}

	result, err := parseEnv(input)

	assert.NoError(t, err)
	assert.Equal(t, "from-env", result["TEST_VAR"])
	assert.Equal(t, "explicit-value", result["EXPLICIT"])
}

func TestParseEnvMapEmptyStringPreserved(t *testing.T) {
	// Test empty strings are preserved as literal values, not read from env
	t.Setenv("TEST_VAR", "should-not-be-used")

	input := map[string]interface{}{
		"TEST_VAR":    "", // Empty string should be preserved
		"ANOTHER_VAR": "",
		"EXPLICIT":    "value",
	}

	result, err := parseEnv(input)

	assert.NoError(t, err)
	assert.Equal(t, "", result["TEST_VAR"])        // Empty string preserved
	assert.Equal(t, "", result["ANOTHER_VAR"])     // Empty string preserved
	assert.Equal(t, "value", result["EXPLICIT"])
}

func TestParseEnvMapMixedTypes(t *testing.T) {
	// Test numbers, booleans, floats are converted to strings
	input := map[string]interface{}{
		"PORT":       8080,
		"MAX_SIZE":   1.5,
		"DEBUG":      true,
		"ENABLED":    false,
		"STRING_VAL": "text",
	}

	result, err := parseEnv(input)

	assert.NoError(t, err)
	assert.Equal(t, "8080", result["PORT"])
	assert.Equal(t, "1.5", result["MAX_SIZE"])
	assert.Equal(t, "true", result["DEBUG"])
	assert.Equal(t, "false", result["ENABLED"])
	assert.Equal(t, "text", result["STRING_VAL"])
}

func TestParseEnvMapWhitespacePreservation(t *testing.T) {
	// Test leading/trailing spaces in values preserved, keys trimmed
	input := map[string]string{
		"  TRIMMED_KEY  ": "  preserved value  ",
		"NORMAL":          "value",
	}

	result, err := parseEnv(input)

	assert.NoError(t, err)
	assert.Equal(t, "  preserved value  ", result["TRIMMED_KEY"])
	assert.Equal(t, "value", result["NORMAL"])
	assert.Len(t, result, 2)
}

func TestParseEnvMapEmptyKeys(t *testing.T) {
	// Test empty/whitespace-only keys are skipped
	input := map[string]string{
		"":      "value1",
		"  ":    "value2",
		"VALID": "value3",
	}

	result, err := parseEnv(input)

	assert.NoError(t, err)
	assert.Equal(t, "value3", result["VALID"])
	assert.Len(t, result, 1)
}

func TestParseEnvArrayFormatRegression(t *testing.T) {
	// Test existing array format behavior unchanged
	input := []interface{}{
		"KEY1=value1",
		"KEY2=value2",
		"KEY3=value with spaces",
	}

	result, err := parseEnv(input)

	assert.NoError(t, err)
	assert.Equal(t, "value1", result["KEY1"])
	assert.Equal(t, "value2", result["KEY2"])
	assert.Equal(t, "value with spaces", result["KEY3"])
}

func TestParseEnvArrayOSEnvReading(t *testing.T) {
	// Test key-only entries read from OS env
	t.Setenv("FROM_ENV", "env-value")

	input := []interface{}{
		"FROM_ENV",
		"EXPLICIT=explicit-value",
	}

	result, err := parseEnv(input)

	assert.NoError(t, err)
	assert.Equal(t, "env-value", result["FROM_ENV"])
	assert.Equal(t, "explicit-value", result["EXPLICIT"])
}

func TestParseEnvArrayEqualsInValues(t *testing.T) {
	// Test complex values with equals signs
	input := []interface{}{
		"BUILD_ARGS=--arg1=val1 --arg2=val2",
		"URL=https://example.com?key=value",
	}

	result, err := parseEnv(input)

	assert.NoError(t, err)
	assert.Equal(t, "--arg1=val1 --arg2=val2", result["BUILD_ARGS"])
	assert.Equal(t, "https://example.com?key=value", result["URL"])
}

func TestParseEnvArrayWhitespaceTrimming(t *testing.T) {
	// Test keys and values are trimmed in array format
	input := []interface{}{
		"  KEY1  =  value1  ",
		"  KEY2=value2",
		"KEY3  =  value3  ",
	}

	result, err := parseEnv(input)

	assert.NoError(t, err)
	assert.Equal(t, "value1", result["KEY1"])
	assert.Equal(t, "value2", result["KEY2"])
	assert.Equal(t, "value3", result["KEY3"])
}

func TestParseEnvErrorInvalidType(t *testing.T) {
	// Test string or number input returns error
	tests := []struct {
		name  string
		input interface{}
	}{
		{"string", "invalid"},
		{"number", 123},
		{"boolean", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseEnv(tt.input)
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "env configuration must be")
		})
	}
}

func TestParseEnvNilInput(t *testing.T) {
	// Test nil input returns nil map without error
	result, err := parseEnv(nil)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

// Integration tests for map env format

func TestPluginWithMapEnv(t *testing.T) {
	// Test plugin-level map format
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"env": {
				"PLUGIN_VAR": "plugin-value",
				"PORT": 8080,
				"DEBUG": true
			},
			"watch": [
				{
					"path": "services/",
					"config": {
						"command": "echo test"
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)

	assert.NoError(t, err)
	assert.Equal(t, "plugin-value", got.Env["PLUGIN_VAR"])
	assert.Equal(t, "8080", got.Env["PORT"])
	assert.Equal(t, "true", got.Env["DEBUG"])
}

func TestStepWithMapEnv(t *testing.T) {
	// Test step-level map format (command step)
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "services/",
					"config": {
						"command": "npm test",
						"env": {
							"NODE_ENV": "test",
							"MAX_WORKERS": 4,
							"COVERAGE": true
						}
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)

	assert.NoError(t, err)
	assert.Len(t, got.Watch, 1)
	assert.Equal(t, "test", got.Watch[0].Step.Env["NODE_ENV"])
	assert.Equal(t, "4", got.Watch[0].Step.Env["MAX_WORKERS"])
	assert.Equal(t, "true", got.Watch[0].Step.Env["COVERAGE"])
}

func TestBuildWithMapEnv(t *testing.T) {
	// Test build-level map format (trigger step)
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "services/",
					"config": {
						"trigger": "downstream-pipeline",
						"build": {
							"env": {
								"DEPLOY_ENV": "staging",
								"REPLICAS": 3
							}
						}
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)

	assert.NoError(t, err)
	assert.Len(t, got.Watch, 1)
	assert.Equal(t, "staging", got.Watch[0].Step.Build.Env["DEPLOY_ENV"])
	assert.Equal(t, "3", got.Watch[0].Step.Build.Env["REPLICAS"])
}

func TestNestedStepsWithMapEnv(t *testing.T) {
	// Test nested steps with map format
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"watch": [
				{
					"path": "services/",
					"config": {
						"group": "Test Group",
						"steps": [
							{
								"command": "test1",
								"env": {
									"TEST": "value1"
								}
							},
							{
								"command": "test2",
								"env": {
									"TEST": "value2"
								}
							}
						]
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)

	assert.NoError(t, err)
	assert.Len(t, got.Watch, 1)
	assert.Len(t, got.Watch[0].Step.Steps, 2)
	assert.Equal(t, "value1", got.Watch[0].Step.Steps[0].Env["TEST"])
	assert.Equal(t, "value2", got.Watch[0].Step.Steps[1].Env["TEST"])
}

func TestMixedEnvFormats(t *testing.T) {
	// Test array at plugin level, map at step level
	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"env": [
				"PLUGIN_VAR=plugin-value",
				"GLOBAL_VAR"
			],
			"watch": [
				{
					"path": "services/",
					"config": {
						"command": "echo test",
						"env": {
							"STEP_VAR": "step-value",
							"PORT": 8080
						}
					}
				}
			]
		}
	}]`

	t.Setenv("GLOBAL_VAR", "from-env")

	got, err := initializePlugin(param)

	assert.NoError(t, err)
	assert.Equal(t, "plugin-value", got.Env["PLUGIN_VAR"])
	assert.Equal(t, "from-env", got.Env["GLOBAL_VAR"])
	assert.Equal(t, "step-value", got.Watch[0].Step.Env["STEP_VAR"])
	assert.Equal(t, "8080", got.Watch[0].Step.Env["PORT"])
}

func TestMapEnvWithOSEnvReading(t *testing.T) {
	// Test map format: null reads from OS, empty string is literal
	t.Setenv("AWS_REGION", "us-west-2")
	t.Setenv("EMPTY_VAR", "should-not-be-used")

	param := `[{
		"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
			"env": {
				"AWS_REGION": null,
				"EMPTY_VAR": "",
				"EXPLICIT": "explicit-value"
			},
			"watch": [
				{
					"path": "services/",
					"config": {
						"command": "echo test"
					}
				}
			]
		}
	}]`

	got, err := initializePlugin(param)

	assert.NoError(t, err)
	assert.Equal(t, "us-west-2", got.Env["AWS_REGION"])  // Null reads from OS env
	assert.Equal(t, "", got.Env["EMPTY_VAR"])            // Empty string is literal
	assert.Equal(t, "explicit-value", got.Env["EXPLICIT"])
}
