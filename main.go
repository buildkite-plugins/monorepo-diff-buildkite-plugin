package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func setupLogger(logLevel string) {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	ll, err := log.ParseLevel(logLevel)
	if err != nil {
		ll = log.InfoLevel
	}

	log.SetLevel(ll)
}

// Version of plugin
var version string = "dev"

func main() {
	log.Infof("--- running monorepo-diff-buildkite-plugin %s", version)

	plugins := env("BUILDKITE_PLUGINS", "")

	log.Debugf("received plugin: \n%v", plugins)

	plugin, err := initializePlugin(plugins)
	if err != nil {
		log.Debug(err)
		log.Fatal(err)
	}

	setupLogger(plugin.LogLevel)

	if env("BUILDKITE_PLUGIN_MONOREPO_DIFF_BUILDKITE_PLUGIN_TEST_MODE", "false") == "true" {
		return
	}

	_, _, pipelinePath, err := uploadPipeline(plugin, generatePipeline)
	if pipelinePath != "" {
		defer os.Remove(pipelinePath)
	}
	if err != nil {
		log.Fatalf("+++ failed to upload pipeline: %v", err)
	}
}
