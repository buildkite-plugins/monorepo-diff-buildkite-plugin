#!/usr/bin/env bash
#
# This script calculates the next semantic version and pushes a tag
#
set -euo pipefail

RELEASE_TYPE="$(buildkite-agent meta-data get "release-type")"

if [[ "${RELEASE_TYPE}" == "major" ]]; then
  echo "🚨 Major releases require manual tagging to prevent accidents."
  echo "Please run: git tag vX.0.0 && git push origin vX.0.0"
  exit 1
fi

# Get latest tag matching v*.*.* pattern
LATEST_TAG=$(git describe --tags --match "v[0-9]*" --abbrev=0 2>/dev/null) || {
  echo "Error: No existing version tags found. Cannot calculate next version."
  exit 1
}

echo "Latest tag: ${LATEST_TAG}"

# Parse version (strip 'v' prefix and any pre-release suffix)
VERSION="${LATEST_TAG#v}"
IFS='.' read -r MAJOR MINOR PATCH <<< "${VERSION%%-*}"

# Calculate new version
case "${RELEASE_TYPE}" in
  minor)
    TAG="v${MAJOR}.$((MINOR + 1)).0"
    ;;
  patch)
    TAG="v${MAJOR}.${MINOR}.$((PATCH + 1))"
    ;;
  *)
    echo "Error: Unknown release type: ${RELEASE_TYPE}"
    exit 1
    ;;
esac

echo "New tag: ${TAG}"

if git ls-remote --exit-code --tags origin "refs/tags/${TAG}" >/dev/null 2>&1; then
  echo "Error: Tag ${TAG} already exists at origin"
  exit 1
fi

echo "${TAG} does not exist at origin. Proceeding... 🚀"

echo "--- Downloading gh"
GH_VERSION="2.57.0"
ARCH=$(uname -m)
case "${ARCH}" in
  x86_64)  GH_ARCH="amd64" ;;
  aarch64) GH_ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: ${ARCH}"
    exit 1
    ;;
esac

GH_DIR="gh_${GH_VERSION}_linux_${GH_ARCH}"
curl -sL "https://github.com/cli/cli/releases/download/v${GH_VERSION}/${GH_DIR}.tar.gz" | tar xz

echo "--- Logging in to gh"
"${GH_DIR}/bin/gh" auth setup-git

echo "+++ Tagging ${BUILDKITE_COMMIT} with ${TAG}"
git tag "${TAG}"
git push origin "${TAG}"