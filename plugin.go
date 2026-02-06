package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

const pluginName = "monorepo-diff"

// Plugin buildkite monorepo diff plugin structure
type Plugin struct {
	Diff          string
	Wait          bool
	LogLevel      string `json:"log_level"`
	Interpolation bool
	Hooks         []HookConfig
	Watch         []WatchConfig
	RawEnv        interface{} `json:"env"`
	Env           map[string]string
	Metadata      map[string]string        `json:"meta_data"`
	RawNotify     []map[string]interface{} `json:"notify" yaml:",omitempty"`
	Notify        []PluginNotify           `yaml:"notify,omitempty"`
}

// HookConfig Plugin hook configuration
type HookConfig struct {
	Command string
}

// WatchConfig Plugin watch configuration
type WatchConfig struct {
	RawPath       interface{} `json:"path"`
	Paths         []string
	Step          Step        `json:"config"`
	Default       interface{} `json:"default"`
	RawSkipPath   interface{} `json:"skip_path"`
	RawExceptPath interface{} `json:"except_path"`
	SkipPaths     []string
	ExceptPaths   []string
}

type Group struct {
	Label string `yaml:"group"`
	Steps []Step `yaml:"steps"`
}

// GithubStatusNotification is notification config for github_commit_status
type GithubStatusNotification struct {
	Context string `yaml:"context,omitempty"`
}

// PluginNotify is notify configuration for pipeline
type PluginNotify struct {
	Slack        string                   `yaml:"slack,omitempty"`
	Email        string                   `yaml:"email,omitempty"`
	PagerDuty    string                   `yaml:"pagerduty_change_event,omitempty"`
	Webhook      string                   `yaml:"webhook,omitempty"`
	Basecamp     string                   `yaml:"basecamp_campfire,omitempty"`
	GithubStatus GithubStatusNotification `yaml:"github_commit_status,omitempty"`
	Condition    string                   `yaml:"if,omitempty"`
}

// Notify is Buildkite notification definition
type StepNotify struct {
	Slack        string                   `yaml:"slack,omitempty"`
	Basecamp     string                   `yaml:"basecamp_campfire,omitempty"`
	GithubStatus GithubStatusNotification `yaml:"github_commit_status,omitempty"`
	Condition    string                   `yaml:"if,omitempty"`
}

// Step is buildkite pipeline definition
type Step struct {
	Group         string                   `yaml:"group,omitempty"`
	Trigger       string                   `yaml:"trigger,omitempty"`
	Label         string                   `yaml:"label,omitempty"`
	Branches      string                   `yaml:"branches,omitempty"`
	Condition     string                   `json:"if,omitempty" yaml:"if,omitempty"`
	Build         Build                    `yaml:"build,omitempty"`
	Command       interface{}              `yaml:"command,omitempty"`
	Commands      interface{}              `yaml:"commands,omitempty"`
	Agents        Agent                    `yaml:"agents,omitempty"`
	ArtifactPaths []string                 `json:"artifact_paths" yaml:"artifact_paths,omitempty"`
	RawEnv        interface{}              `json:"env" yaml:",omitempty"`
	Plugins       []map[string]interface{} `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Env           map[string]string        `yaml:"env,omitempty"`
	Async         bool                     `yaml:"async,omitempty"`
	SoftFail      interface{}              `json:"soft_fail" yaml:"soft_fail,omitempty"`
	RawNotify     []map[string]interface{} `json:"notify" yaml:",omitempty"`
	Notify        []StepNotify             `yaml:"notify,omitempty"`
	DependsOn     interface{}              `json:"depends_on" yaml:"depends_on,omitempty"`
	Key           string                   `yaml:"key,omitempty"`
	Secrets       interface{}              `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	Steps         []Step                   `yaml:"steps,omitempty"`
}

// isValid checks if a step has required fields (command, trigger, or group with steps)
func (s Step) isValid() bool {
	if s.Group != "" {
		return s.hasValidNesting()
	}
	return s.hasAction()
}

// hasAction checks if a step has a command or trigger
func (s Step) hasAction() bool {
	return s.Command != nil || s.Commands != nil || s.Trigger != ""
}

// hasValidNesting validates group step nesting
func (s Step) hasValidNesting() bool {
	if s.hasAction() {
		return true
	}

	if len(s.Steps) > 0 {
		for _, nestedStep := range s.Steps {
			if !nestedStep.isValid() {
				return false
			}
		}
		return true
	}

	return false
}

// UnmarshalJSON handles both "artifacts" and "artifact_paths" field names for backward compatibility
// Both fields are supported by the Buildkite API; "artifact_paths" is preferred per documentation
func (step *Step) UnmarshalJSON(data []byte) error {
	// Check which fields are present without full unmarshaling
	var fieldCheck map[string]json.RawMessage
	if err := json.Unmarshal(data, &fieldCheck); err != nil {
		return err
	}

	_, hasArtifacts := fieldCheck["artifacts"]
	_, hasArtifactPaths := fieldCheck["artifact_paths"]

	// Validate that both fields are not specified
	if hasArtifacts && hasArtifactPaths {
		return errors.New("cannot specify both 'artifacts' and 'artifact_paths'; please use 'artifact_paths'")
	}

	// Use a type alias to avoid infinite recursion
	type stepAlias Step

	// Unmarshal the main struct (this will populate artifact_paths if present)
	if err := json.Unmarshal(data, (*stepAlias)(step)); err != nil {
		return err
	}

	// If only "artifacts" was specified, manually extract and use it
	if hasArtifacts && !hasArtifactPaths {
		var temp struct {
			Artifacts []string `json:"artifacts"`
		}
		if err := json.Unmarshal(data, &temp); err != nil {
			return err
		}
		step.ArtifactPaths = temp.Artifacts
	}

	return nil
}

// Agent is Buildkite agent definition
type Agent map[string]string

// Build is buildkite build definition
type Build struct {
	Message  string            `yaml:"message,omitempty"`
	Branch   string            `yaml:"branch,omitempty"`
	Commit   string            `yaml:"commit,omitempty"`
	RawEnv   interface{}       `json:"env" yaml:",omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	Metadata map[string]string `json:"meta_data" yaml:"meta_data,omitempty"`
	// Notify  []Notify          `yaml:"notify,omitempty"`
}

// UnmarshalJSON set defaults properties
func (plugin *Plugin) UnmarshalJSON(data []byte) error {
	type plain Plugin

	def := &plain{
		Diff:          "git diff --name-only HEAD~1",
		Wait:          false,
		LogLevel:      "info",
		Interpolation: true,
	}

	if err := json.Unmarshal(data, def); err != nil {
		return err
	}

	*plugin = Plugin(*def)

	parseResult, err := parseEnv(plugin.RawEnv)
	if err != nil {
		return errors.New("failed to parse plugin configuration")
	}

	plugin.Env = parseResult
	plugin.RawEnv = nil

	metaDataParseResult, err := parseMetadata(plugin.Metadata)
	if err != nil {
		return errors.New("failed to parse metadata configuration")
	}

	plugin.Metadata = metaDataParseResult

	setPluginNotify(&plugin.Notify, &plugin.RawNotify)

	for i, p := range plugin.Watch {
		if p.Default != nil {
			plugin.Watch[i].Paths = []string{}
			if config, ok := p.Default.(map[string]interface{}); ok && len(config) > 0 {
				// Use the default config directly
				conf := config
				if _, ok := config["config"]; ok {
					// or allow for it to be in a config configuration
					conf = config["config"].(map[string]interface{})
				}
				b, _ := json.Marshal(conf)
				err := json.Unmarshal(b, &plugin.Watch[i].Step)
				if err != nil {
					return err
				}
			}
			plugin.Watch[i].Default = true
		} else if p.RawPath != nil {
			// Path, SkipPath and ExceptPath can be string or an array of strings,
			// handle both cases and create an array of paths on all.
			switch p.RawPath.(type) {
			case string:
				plugin.Watch[i].Paths = []string{plugin.Watch[i].RawPath.(string)}
			case []interface{}:
				for _, v := range plugin.Watch[i].RawPath.([]interface{}) {
					plugin.Watch[i].Paths = append(plugin.Watch[i].Paths, v.(string))
				}
			}
		}

		switch p.RawSkipPath.(type) {
		case string:
			plugin.Watch[i].SkipPaths = []string{plugin.Watch[i].RawSkipPath.(string)}
		case []interface{}:
			for _, v := range plugin.Watch[i].RawSkipPath.([]interface{}) {
				plugin.Watch[i].SkipPaths = append(plugin.Watch[i].SkipPaths, v.(string))
			}
		}

		switch p.RawExceptPath.(type) {
		case string:
			plugin.Watch[i].ExceptPaths = []string{plugin.Watch[i].RawExceptPath.(string)}
		case []interface{}:
			for _, v := range plugin.Watch[i].RawExceptPath.([]interface{}) {
				plugin.Watch[i].ExceptPaths = append(plugin.Watch[i].ExceptPaths, v.(string))
			}
		}

		if plugin.Watch[i].Step.Trigger != "" {
			setBuild(&plugin.Watch[i].Step.Build)
		}

		if plugin.Watch[i].Step.RawNotify != nil {
			setNotify(&plugin.Watch[i].Step.Notify, &plugin.Watch[i].Step.RawNotify)
		}

		appendEnv(&plugin.Watch[i], plugin.Env)

		// Attempt to parse the metadata after the env's
		parsedMetadata, err := parseMetadata(plugin.Metadata)
		if err != nil {
			return errors.New("failed to parse metadata configuration")
		}

		appendMetadata(&plugin.Watch[i], parsedMetadata)

		p.RawPath = nil
		p.RawSkipPath = nil
	}

	return nil
}

func initializePlugin(data string) (Plugin, error) {
	log.Debugf("parsing plugin config: %v", data)

	var pluginConfigs []map[string]json.RawMessage

	if err := json.Unmarshal([]byte(data), &pluginConfigs); err != nil {
		log.Debug(err)
		return Plugin{}, errors.New("failed to parse plugin configuration")
	}

	for _, p := range pluginConfigs {
		for key, pluginConfig := range p {
			if strings.HasPrefix(getPluginName(key), pluginName) {
				var plugin Plugin

				if err := json.Unmarshal(pluginConfig, &plugin); err != nil {
					log.Debug(err)
					return Plugin{}, err
				}

				return plugin, nil
			}
		}
	}

	return Plugin{}, errors.New("could not initialize plugin")
}

func setPluginNotify(notifications *[]PluginNotify, rawNotify *[]map[string]interface{}) {
	for _, v := range *rawNotify {
		var notify PluginNotify

		if condition, ok := isString(v["if"]); ok {
			notify.Condition = condition
		}

		if email, ok := isString(v["email"]); ok {
			notify.Email = email
			*notifications = append(*notifications, notify)
			continue
		}

		if basecamp, ok := isString(v["basecamp_campfire"]); ok {
			notify.Basecamp = basecamp
			*notifications = append(*notifications, notify)
			continue
		}

		if webhook, ok := isString(v["webhook"]); ok {
			notify.Webhook = webhook
			*notifications = append(*notifications, notify)
			continue
		}

		if pagerduty, ok := isString(v["pagerduty_change_event"]); ok {
			notify.PagerDuty = pagerduty
			*notifications = append(*notifications, notify)
			continue
		}

		if slack, ok := isString(v["slack"]); ok {
			notify.Slack = slack
			*notifications = append(*notifications, notify)
			continue
		}

		if github, ok := v["github_commit_status"].(map[string]interface{}); ok {
			if context, ok := isString(github["context"]); ok {
				notify.GithubStatus = GithubStatusNotification{Context: context}
				*notifications = append(*notifications, notify)
			}
			continue
		}
	}

	*rawNotify = nil
}

func setNotify(notifications *[]StepNotify, rawNotify *[]map[string]interface{}) {
	for _, v := range *rawNotify {
		var notify StepNotify

		if condition, ok := isString(v["if"]); ok {
			notify.Condition = condition
		}

		if basecamp, ok := isString(v["basecamp_campfire"]); ok {
			notify.Basecamp = basecamp
			*notifications = append(*notifications, notify)
			continue
		}

		if slack, ok := isString(v["slack"]); ok {
			notify.Slack = slack
			*notifications = append(*notifications, notify)
			continue
		}

		if github, ok := v["github_commit_status"].(map[string]interface{}); ok {
			if context, ok := isString(github["context"]); ok {
				notify.GithubStatus = GithubStatusNotification{Context: context}
				*notifications = append(*notifications, notify)
			}
			continue
		}
	}

	*rawNotify = nil
}

func escapeInterpolation(s string) string {
	return strings.ReplaceAll(s, "$", "$$")
}

func setBuild(build *Build) {
	// when defaulting to existing literal values make sure those values
	// don't trigger interpolation with any stray dollar characters.

	if build.Message == "" {
		build.Message = escapeInterpolation(env("BUILDKITE_MESSAGE", ""))
	}

	if build.Branch == "" {
		build.Branch = escapeInterpolation(env("BUILDKITE_BRANCH", ""))
	}

	if build.Commit == "" {
		build.Commit = escapeInterpolation(env("BUILDKITE_COMMIT", ""))
	}
}

// processNestedStepsEnv recursively processes environment variables for nested steps
func processNestedStepsEnv(steps []Step, env map[string]string) {
	for i := range steps {
		// Parse the step's own env
		steps[i].Env, _ = parseEnv(steps[i].RawEnv)
		steps[i].Build.Env, _ = parseEnv(steps[i].Build.RawEnv)

		// Append top-level env to this step
		for key, value := range env {
			if steps[i].Command != nil || steps[i].Commands != nil {
				if steps[i].Env == nil {
					steps[i].Env = make(map[string]string)
				}
				steps[i].Env[key] = value
			} else if steps[i].Trigger != "" {
				if steps[i].Build.Env == nil {
					steps[i].Build.Env = make(map[string]string)
				}
				steps[i].Build.Env[key] = value
			}
		}

		// Clear RawEnv fields
		steps[i].RawEnv = nil
		steps[i].Build.RawEnv = nil

		// Recursively process any nested steps
		if len(steps[i].Steps) > 0 {
			processNestedStepsEnv(steps[i].Steps, env)
		}
	}
}

// appends top level env to Step.Env and Step.Build.Env
func appendEnv(watch *WatchConfig, env map[string]string) {
	watch.Step.Env, _ = parseEnv(watch.Step.RawEnv)
	watch.Step.Build.Env, _ = parseEnv(watch.Step.Build.RawEnv)

	for key, value := range env {
		if watch.Step.Command != nil || watch.Step.Commands != nil {
			if watch.Step.Env == nil {
				watch.Step.Env = make(map[string]string)
			}

			watch.Step.Env[key] = value
			continue
		}
		if watch.Step.Trigger != "" {
			if watch.Step.Build.Env == nil {
				watch.Step.Build.Env = make(map[string]string)
			}

			watch.Step.Build.Env[key] = value
			continue
		}
	}

	watch.Step.RawEnv = nil
	watch.Step.Build.RawEnv = nil
	// Process nested steps' environment variables
	if len(watch.Step.Steps) > 0 {
		processNestedStepsEnv(watch.Step.Steps, env)
	}
	watch.RawPath = nil
	watch.RawSkipPath = nil
}

// appends build metadata
func appendMetadata(watch *WatchConfig, metadata map[string]string) {
	if len(metadata) == 0 {
		return
	}
	// Only apply metadata to trigger steps, not command steps
	if watch.Step.Trigger != "" {
		if watch.Step.Build.Metadata == nil {
			watch.Step.Build.Metadata = make(map[string]string)
		}
		for k, v := range metadata {
			watch.Step.Build.Metadata[k] = v
		}
	}
}

// parseEnv converts env configuration from various formats to map[string]string.
// Supports two formats:
//   - Array format: ["KEY=value", "KEY2"] - existing format, KEY2 reads from OS env
//   - Map format: {"KEY": "value", "KEY2": nil} - new format, only nil reads from OS env
func parseEnv(raw interface{}) (map[string]string, error) {
	if raw == nil {
		return nil, nil
	}

	switch v := raw.(type) {
	case map[string]string:
		// Direct string map - all values are literal (including empty strings)
		result := make(map[string]string, len(v))
		for k, val := range v {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			// Preserve all values including empty strings
			result[key] = val
		}
		return result, nil

	case map[string]interface{}:
		// Generic map - only nil reads from OS environment
		result := make(map[string]string, len(v))
		for k, val := range v {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			// Only null values read from OS environment
			if val == nil {
				result[key] = env(key, "")
			} else {
				// Convert to string, preserving empty strings
				result[key] = fmt.Sprintf("%v", val)
			}
		}
		return result, nil

	case []interface{}:
		// Array format - preserve existing behavior exactly
		result := make(map[string]string)
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				continue
			}

			split := strings.SplitN(str, "=", 2)
			key := strings.TrimSpace(split[0])

			if key == "" {
				continue
			}

			// Only key exists - read from OS environment
			if len(split) == 1 {
				result[key] = env(key, "")
				continue
			}

			// Key=value - trim both (backwards compatibility)
			if len(split) == 2 {
				result[key] = strings.TrimSpace(split[1])
			}
		}
		return result, nil

	default:
		return nil, errors.New("env configuration must be an array of strings (e.g., ['KEY=value']) or a map (e.g., {KEY: 'value'})")
	}
}

// parse metadata in format from key:value to map[key] = value
func parseMetadata(raw interface{}) (map[string]string, error) {
	if raw == nil {
		return nil, nil
	}

	switch v := raw.(type) {
	case map[string]string:
		return v, nil
	case map[string]interface{}:
		result := make(map[string]string)
		for k, val := range v {
			result[k] = fmt.Sprintf("%v", val)
		}
		return result, nil
	case []interface{}:
		result := make(map[string]string)
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				continue
			}
			split := strings.SplitN(str, ":", 2)
			key := strings.TrimSpace(split[0])
			value := ""
			if len(split) > 1 {
				value = strings.TrimSpace(split[1])
			}
			if key != "" {
				result[key] = value
			}
		}
		return result, nil
	default:
		return nil, errors.New("failed to parse metadata configuration: unknown type")
	}
}

func getPluginName(s string) string {
	ref := s
	if strings.HasPrefix(ref, "github.com/") && !strings.Contains(ref, "://") {
		ref = "https://" + ref
	}

	u, err := url.Parse(ref)
	// if URL could not be parsed, assume it is a direct reference
	if err != nil {
		return s
	}

	// remove the org from the path
	_, repo := path.Split(u.Path)
	return repo
}
