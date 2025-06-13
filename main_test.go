package main

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)

	// set some env variables for using in tests
	// given this is a test we can use `_` for the return; our tests will check values
	_ = os.Setenv("BUILDKITE_COMMIT", "123")
	_ = os.Setenv("BUILDKITE_MESSAGE", "fix: temp file not correctly deleted")
	_ = os.Setenv("BUILDKITE_BRANCH", "go-rewrite")
	_ = os.Setenv("env3", "env-3")
	_ = os.Setenv("env4", "env-4")
	_ = os.Setenv("TEST_MODE", "true")

	run := m.Run()

	os.Exit(run)
}

func TestSetupLogger(t *testing.T) {
	setupLogger("debug")
	assert.Equal(t, log.GetLevel(), log.DebugLevel)
	setupLogger("weird level")
	assert.Equal(t, log.GetLevel(), log.InfoLevel)
}
