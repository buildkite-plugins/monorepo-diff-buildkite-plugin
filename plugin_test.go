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
						"artifacts": [ "artifiact-1" ],
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
					Async:     true,
					Agents:    map[string]string{"queue": "queue-1", "database": "postgres"},
					Artifacts: []string{"artifiact-1"},
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
	param := `[
		{
			"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#commit": {
				"env": {
					"anInvalidKey": "An Invalid Value"
				},
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
