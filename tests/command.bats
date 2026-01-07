#!/usr/bin/env bats

load "$BATS_PLUGIN_PATH/load.bash"

# Uncomment the following line to debug stubbed commands
# export COMMAND_STUB_DEBUG=/dev/tty

setup() {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_BUILDKITE_PLUGIN_TEST_MODE=true
  # Set a minimal plugin config (only used by Go binary, not the bash hook's download logic)
  export BUILDKITE_PLUGINS='[{"monorepo-diff": {}}]'
  
  # Create a mock binary in the current directory for download mode tests
  cat > "$PWD/monorepo-diff-buildkite-plugin" << 'MOCKBIN'
#!/bin/bash
echo "Mock binary executed with args: $@"
MOCKBIN
  chmod +x "$PWD/monorepo-diff-buildkite-plugin"
}

teardown() {
  # Clean up the mock binary
  rm -f "$PWD/monorepo-diff-buildkite-plugin"
}

@test "download=false with binary in PATH succeeds" {
  # Buildkite sets this env var based on plugin config
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=false
  export PATH="$PWD:$PATH"

  run "$PWD/hooks/command"

  assert_success
  assert_output --partial "Mock binary executed"
}

@test "download=false without binary in PATH fails with error" {
  # Buildkite sets this env var based on plugin config
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=false
  
  # Remove mock binary from current directory and don't add to PATH
  rm -f "$PWD/monorepo-diff-buildkite-plugin"
  export PATH="/usr/bin:/bin"

  run "$PWD/hooks/command"

  assert_failure
  assert_output --partial "Binary 'monorepo-diff-buildkite-plugin' not found in PATH"
  assert_output --partial "Please install it or set download: true"
}

@test "download=true executes binary in test mode" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=true

  # In test mode, it skips download but still executes the mock binary we created
  run "$PWD/hooks/command"

  assert_success
  assert_output --partial "Mock binary executed"
}

@test "download defaults to true when not set" {
  # Don't set BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD - should default to true

  # Should default to true, skip download in test mode, and execute mock binary
  run "$PWD/hooks/command"

  assert_success
  assert_output --partial "Mock binary executed"
}

@test "download=false passes arguments to preinstalled binary" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=false
  export PATH="$PWD:$PATH"

  run "$PWD/hooks/command" arg1 arg2

  assert_success
  assert_output --partial "Mock binary executed with args: arg1 arg2"
}

@test "download=true in test mode skips actual download" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=true

  # Stub curl to verify it's NOT called in test mode
  stub curl \
    ": echo 'ERROR: curl should not be called in test mode'; exit 1"

  run "$PWD/hooks/command"

  # Should succeed without calling curl
  assert_success
  
  # Clean up stub (it should not have been called)
  unstub curl || true
}

# Tests for download_with_retry() function (SUP-5615)

@test "download retries on transient failure and succeeds" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=true
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_BUILDKITE_PLUGIN_TEST_MODE=false
  # Use pinned version to skip get_latest_version API call
  export BUILDKITE_PLUGINS='[{"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#v1.0.0": {}}]'
  
  # Remove mock binary so it actually needs to download
  rm -f "$PWD/monorepo-diff-buildkite-plugin"
  rm -f "$PWD/monorepo-diff-buildkite-plugin.version"
  
  # Stub curl: fail twice on download, then succeed and create executable
  stub curl \
    "-fL * -o * : exit 1" \
    "-fL * -o * : exit 1" \
    "-fL * -o * : echo '#!/bin/bash' > \"\${4}\"; echo 'echo test' >> \"\${4}\"; exit 0"

  run "$PWD/hooks/command"

  assert_output --partial "Downloading binary (attempt 1/3)"
  assert_output --partial "Download failed, retrying"
  assert_output --partial "Downloading binary (attempt 2/3)"
  assert_output --partial "Downloading binary (attempt 3/3)"
  assert_output --partial "Download successful"

  unstub curl
}

@test "download fails after max retries exhausted" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=true
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_BUILDKITE_PLUGIN_TEST_MODE=false
  # Use pinned version to skip get_latest_version API call
  export BUILDKITE_PLUGINS='[{"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#v1.0.0": {}}]'
  
  # Remove mock binary so it actually needs to download
  rm -f "$PWD/monorepo-diff-buildkite-plugin"
  
  # Stub curl to always fail on download
  stub curl \
    "-fL * -o * : exit 1" \
    "-fL * -o * : exit 1" \
    "-fL * -o * : exit 1"

  run "$PWD/hooks/command"

  assert_failure
  assert_output --partial "Downloading binary (attempt 1/3)"
  assert_output --partial "Downloading binary (attempt 2/3)"
  assert_output --partial "Downloading binary (attempt 3/3)"
  assert_output --partial "Failed to download"

  unstub curl
}
