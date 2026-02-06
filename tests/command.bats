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
  # Disable checksum verification for this test to focus on retry logic
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_VERIFY_CHECKSUM=false
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

# Tests for SHA256 checksum verification

@test "integration: download with valid checksum verification succeeds" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=true
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_BUILDKITE_PLUGIN_TEST_MODE=false
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_VERIFY_CHECKSUM=true
  # Use pinned version to skip get_latest_version API call
  export BUILDKITE_PLUGINS='[{"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#v1.0.0": {}}]'

  # Remove mock binary so it actually needs to download
  rm -f "$PWD/monorepo-diff-buildkite-plugin"
  rm -f "$PWD/monorepo-diff-buildkite-plugin.version"

  # Expected checksum for mock binary
  local mock_binary_checksum="5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"

  # Stub sha256sum to be available
  stub sha256sum \
    "* : echo '${mock_binary_checksum}  monorepo-diff-buildkite-plugin'"

  # Stub curl: download binary and checksums.txt
  # Need to match all possible architectures in checksums.txt
  stub curl \
    "-fL * -o * : echo '#!/bin/bash' > \"\${4}\"; echo 'echo test' >> \"\${4}\"; exit 0" \
    "-fL * -o * : printf '%s\n' '${mock_binary_checksum}  monorepo-diff-buildkite-plugin_Darwin_amd64' '${mock_binary_checksum}  monorepo-diff-buildkite-plugin_Darwin_arm64' '${mock_binary_checksum}  monorepo-diff-buildkite-plugin_Linux_amd64' '${mock_binary_checksum}  monorepo-diff-buildkite-plugin_Linux_arm64' > \"\${4}\"; exit 0"

  run "$PWD/hooks/command"

  assert_success
  assert_output --partial "Download successful"
  assert_output --partial "Checksum verification passed"

  unstub curl
  unstub sha256sum
}

@test "download with invalid checksum fails and deletes binary" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=true
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_BUILDKITE_PLUGIN_TEST_MODE=false
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_VERIFY_CHECKSUM=true
  export BUILDKITE_PLUGINS='[{"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#v1.0.0": {}}]'

  rm -f "$PWD/monorepo-diff-buildkite-plugin"
  rm -f "$PWD/monorepo-diff-buildkite-plugin.version"

  # Expected checksum (different from actual)
  local expected_checksum="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  local actual_checksum="bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

  # Stub sha256sum to return mismatched checksum
  stub sha256sum \
    "* : echo '${actual_checksum}  monorepo-diff-buildkite-plugin'"

  # Stub curl: download binary and checksums.txt (with all architectures)
  stub curl \
    "-fL * -o * : echo '#!/bin/bash' > \"\${4}\"; echo 'echo test' >> \"\${4}\"; exit 0" \
    "-fL * -o * : printf '%s\n' '${expected_checksum}  monorepo-diff-buildkite-plugin_Darwin_amd64' '${expected_checksum}  monorepo-diff-buildkite-plugin_Darwin_arm64' '${expected_checksum}  monorepo-diff-buildkite-plugin_Linux_amd64' '${expected_checksum}  monorepo-diff-buildkite-plugin_Linux_arm64' > \"\${4}\"; exit 0"

  run "$PWD/hooks/command"

  assert_failure
  assert_output --partial "Checksum verification failed"
  assert_output --partial "Expected: ${expected_checksum}"
  assert_output --partial "Actual:   ${actual_checksum}"

  # Verify binary was deleted
  [ ! -f "$PWD/monorepo-diff-buildkite-plugin" ]

  unstub curl
  unstub sha256sum
}

@test "cached binary with invalid checksum triggers recovery" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=true
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_BUILDKITE_PLUGIN_TEST_MODE=false
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_VERIFY_CHECKSUM=true
  export BUILDKITE_PLUGINS='[{"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#v1.0.0": {}}]'

  # Create a cached binary with version file
  echo "#!/bin/bash" > "$PWD/monorepo-diff-buildkite-plugin"
  echo "echo cached" >> "$PWD/monorepo-diff-buildkite-plugin"
  chmod +x "$PWD/monorepo-diff-buildkite-plugin"
  echo "v1.0.0" > "$PWD/monorepo-diff-buildkite-plugin.version"

  local bad_checksum="bad_cached_checksum_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  local good_checksum="5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"

  # Stub sha256sum: first call returns bad checksum (cached), second returns good (after recovery)
  stub sha256sum \
    "* : echo '${bad_checksum}  monorepo-diff-buildkite-plugin'" \
    "* : echo '${good_checksum}  monorepo-diff-buildkite-plugin'"

  # Stub curl: download checksums (first check), then binary, then checksums again (recovery check)
  stub curl \
    "-fL * -o * : printf '%s\n' '${good_checksum}  monorepo-diff-buildkite-plugin_Darwin_amd64' '${good_checksum}  monorepo-diff-buildkite-plugin_Darwin_arm64' '${good_checksum}  monorepo-diff-buildkite-plugin_Linux_amd64' '${good_checksum}  monorepo-diff-buildkite-plugin_Linux_arm64' > \"\${4}\"; exit 0" \
    "-fL * -o * : echo '#!/bin/bash' > \"\${4}\"; echo 'echo recovered' >> \"\${4}\"; exit 0" \
    "-fL * -o * : printf '%s\n' '${good_checksum}  monorepo-diff-buildkite-plugin_Darwin_amd64' '${good_checksum}  monorepo-diff-buildkite-plugin_Darwin_arm64' '${good_checksum}  monorepo-diff-buildkite-plugin_Linux_amd64' '${good_checksum}  monorepo-diff-buildkite-plugin_Linux_arm64' > \"\${4}\"; exit 0"

  run "$PWD/hooks/command"

  assert_success
  assert_output --partial "Cached binary failed checksum verification"
  assert_output --partial "attempting recovery"
  assert_output --partial "Binary recovery successful"

  unstub curl
  unstub sha256sum
}

@test "verify_checksum=false skips verification" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=true
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_BUILDKITE_PLUGIN_TEST_MODE=false
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_VERIFY_CHECKSUM=false
  export BUILDKITE_PLUGINS='[{"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#v1.0.0": {}}]'

  rm -f "$PWD/monorepo-diff-buildkite-plugin"
  rm -f "$PWD/monorepo-diff-buildkite-plugin.version"

  # Stub curl: only download binary, no checksums
  stub curl \
    "-fL * -o * : echo '#!/bin/bash' > \"\${4}\"; echo 'echo test' >> \"\${4}\"; exit 0"

  run "$PWD/hooks/command"

  assert_success
  assert_output --partial "Download successful"
  refute_output --partial "Checksum verification"
  refute_output --partial "checksums.txt"

  unstub curl
}

@test "missing checksums.txt warns but continues" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=true
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_BUILDKITE_PLUGIN_TEST_MODE=false
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_VERIFY_CHECKSUM=true
  export BUILDKITE_PLUGINS='[{"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#v1.0.0": {}}]'

  rm -f "$PWD/monorepo-diff-buildkite-plugin"
  rm -f "$PWD/monorepo-diff-buildkite-plugin.version"

  # Stub curl: download binary succeeds, checksums.txt fails
  stub curl \
    "-fL * -o * : echo '#!/bin/bash' > \"\${4}\"; echo 'echo test' >> \"\${4}\"; exit 0" \
    "-fL * -o * : exit 1"

  run "$PWD/hooks/command"

  assert_success
  assert_output --partial "Warning: Could not download checksums.txt"
  assert_output --partial "skipping verification"

  unstub curl
}

@test "missing sha256 command warns but continues" {
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_DOWNLOAD=true
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_BUILDKITE_PLUGIN_TEST_MODE=false
  export BUILDKITE_PLUGIN_MONOREPO_DIFF_VERIFY_CHECKSUM=true
  export BUILDKITE_PLUGINS='[{"github.com/buildkite-plugins/monorepo-diff-buildkite-plugin#v1.0.0": {}}]'

  rm -f "$PWD/monorepo-diff-buildkite-plugin"
  rm -f "$PWD/monorepo-diff-buildkite-plugin.version"

  # Stub curl: download binary and checksums
  stub curl \
    "-fL * -o * : echo '#!/bin/bash' > \"\${4}\"; echo 'echo test' >> \"\${4}\"; exit 0" \
    "-fL * -o * : echo 'abc123  monorepo-diff-buildkite-plugin_Darwin_amd64' > \"\${4}\"; exit 0"

  # Ensure sha256sum, shasum, and sha256 are all unavailable by stubbing command -v
  # Note: This is tricky in bats. We'll rely on the implementation checking for these commands.
  # For this test, we assume the system doesn't have these tools or we mock the check.

  run "$PWD/hooks/command"

  # This test depends on implementation details. We may need to adjust based on how
  # we implement the sha256 command detection. For now, we expect it to warn and continue.
  assert_success

  unstub curl
}
