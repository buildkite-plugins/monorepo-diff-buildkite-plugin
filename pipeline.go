package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// WaitStep represents a Buildkite Wait Step
// https://buildkite.com/docs/pipelines/wait-step
// We can't use Step here since the value for Wait is always nil
// regardless of whether or not we want to include the key.
type WaitStep struct{}

func (WaitStep) MarshalYAML() (interface{}, error) {
	return map[string]interface{}{
		"wait": nil,
	}, nil
}

func (s Step) MarshalYAML() (interface{}, error) {
	if s.Group == "" {
		type Alias Step
		return (Alias)(s), nil
	}

	label := s.Group
	s.Group = ""
	stps := []Step{s}
	if s.Steps != nil {
		stps = s.Steps
	}
	return Group{Label: label, Steps: stps}, nil
}

func (n PluginNotify) MarshalYAML() (interface{}, error) {
	type Alias PluginNotify
	return (Alias)(n), nil
}

// PipelineGenerator generates pipeline file
type PipelineGenerator func(steps []Step, plugin Plugin) (*os.File, bool, error)

func uploadPipeline(plugin Plugin, generatePipeline PipelineGenerator) (string, []string, error) {
	diffOutput, err := diff(plugin.Diff)
	if err != nil {
		log.Fatal(err)
		return "", []string{}, err
	}

	if len(diffOutput) < 1 {
		log.Info("No changes detected. Skipping pipeline upload.")
		return "", []string{}, nil
	}

	log.Debug("Output from diff: \n" + strings.Join(diffOutput, "\n"))

	steps, err := stepsToTrigger(diffOutput, plugin.Watch)
	if err != nil {
		return "", []string{}, err
	}

	pipeline, hasSteps, err := generatePipeline(steps, plugin)
	if err != nil {
		return "", []string{}, err
	}
	defer func() {
		if removeErr := os.Remove(pipeline.Name()); removeErr != nil {
			log.Errorf("Failed to remove temporary pipeline file: %v", removeErr)
		}
	}()

	if !hasSteps {
		// Handle the case where no steps were provided
		log.Info("No steps generated. Skipping pipeline upload.")
		return "", []string{}, nil
	}

	cmd := "buildkite-agent"
	args := []string{"pipeline", "upload", pipeline.Name()}

	if !plugin.Interpolation {
		args = append(args, "--no-interpolation")
	}

	_, err = executeCommand("buildkite-agent", args)

	return cmd, args, err
}

func diff(command string) ([]string, error) {
	log.Infof("Running diff command: %s", command)

	output, err := executeCommand(
		env("SHELL", "bash"),
		[]string{"-c", strings.ReplaceAll(command, "\n", " ")},
	)
	if err != nil {
		return nil, fmt.Errorf("diff command failed: %v", err)
	}

	return strings.Fields(strings.TrimSpace(output)), nil
}

func stepsToTrigger(files []string, watch []WatchConfig) ([]Step, error) {
	steps := []Step{}
	var defaultStep *Step

	for _, w := range watch {
		if w.Default != nil {
			defaultStep = &w.Step
			continue
		}
		except := false

		for _, ex := range w.ExceptPaths {
			if except {
				break
			}

			for _, f := range files {
				exceptMatch, errExcept := matchPath(ex, f)
				if errExcept != nil {
					return nil, errExcept
				}
				if exceptMatch {
					log.Printf("excepted: %s\n", f)
					except = true
					break
				}
			}
		}

		if except {
			continue
		}

		for _, p := range w.Paths {
			for _, f := range files {
				match, err := matchPath(p, f)

				skip := false
				for _, sp := range w.SkipPaths {
					skipMatch, errSkip := matchPath(sp, f)

					if errSkip != nil {
						return nil, errSkip
					}

					if skipMatch {
						skip = true
					}
				}

				if err != nil {
					return nil, err
				}

				if match && !skip {
					if len(w.Step.Steps) > 0 {
						steps = append(steps, w.Step.Steps...)
					} else {
						steps = append(steps, w.Step)
					}
					break
				}
			}
		}
	}

	if len(steps) == 0 && defaultStep != nil {
		if len(defaultStep.Steps) > 0 {
			steps = append(steps, defaultStep.Steps...)
		} else {
			steps = append(steps, *defaultStep)
		}
	}

	return dedupSteps(steps), nil
}

// matchPath checks if the file f matches the path p.
func matchPath(p string, f string) (bool, error) {
	// If the path contains a glob, the `doublestar.Match`
	// method is used to determine the match,
	// otherwise `strings.HasPrefix` is used.
	if strings.Contains(p, "*") {
		match, err := doublestar.Match(p, f)
		if err != nil {
			return false, fmt.Errorf("path matching failed: %v", err)
		}
		if match {
			return true, nil
		}
	}
	if strings.HasPrefix(f, p) {
		return true, nil
	}
	return false, nil
}

func dedupSteps(steps []Step) []Step {
	unique := []Step{}
	for _, p := range steps {
		duplicate := false
		for _, t := range unique {
			if reflect.DeepEqual(p, t) {
				duplicate = true
				break
			}
		}

		if !duplicate {
			unique = append(unique, p)
		}
	}

	return unique
}

func generatePipeline(steps []Step, plugin Plugin) (*os.File, bool, error) {
	tmp, err := os.CreateTemp(os.TempDir(), "bmrd-")
	if err != nil {
		return nil, false, fmt.Errorf("could not create temporary pipeline file: %v", err)
	}

	yamlSteps := make([]yaml.Marshaler, len(steps))

	for i, step := range steps {
		yamlSteps[i] = step
	}

	if plugin.Wait {
		yamlSteps = append(yamlSteps, WaitStep{})
	}

	for _, cmd := range plugin.Hooks {
		yamlSteps = append(yamlSteps, Step{Command: cmd.Command})
	}

	yamlNotify := make([]yaml.Marshaler, len(plugin.Notify))
	for i, n := range plugin.Notify {
		yamlNotify[i] = n
	}

	pipeline := map[string][]yaml.Marshaler{
		"steps": yamlSteps,
	}

	if len(yamlNotify) > 0 {
		pipeline["notify"] = yamlNotify
	}

	data, err := yaml.Marshal(&pipeline)
	if err != nil {
		return nil, false, fmt.Errorf("could not serialize the pipeline: %v", err)
	}

	// Disable logging in context of go tests.
	if env("TEST_MODE", "") != "true" {
		fmt.Printf("Generated Pipeline:\n%s\n", string(data))
		for i, step := range steps {
			fmt.Printf("Step %d: Label=%s, Matrix=%v, Command=%v\n", i, step.Label, step.Matrix, step.Command)
		}
	}

	if err = os.WriteFile(tmp.Name(), data, 0o644); err != nil {
		return nil, false, fmt.Errorf("could not write step to temporary file: %v", err)
	}

	// Returns the temporary file and a boolean indicating whether or not the pipeline has steps
	if len(yamlSteps) == 0 {
		return tmp, false, nil
	} else {
		return tmp, true, nil
	}
}
