# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
* Accept both `artifact_paths` (preferred) and `artifacts` (alternative) field names in plugin configuration for backward compatibility (#120)

### Fixed
* Fix `artifact_paths` field being ignored when specified in plugin configuration. Both `artifact_paths` and `artifacts` now work correctly (#120)
* Fix typo in test case: "artifiact" → "artifact"
* Fix BATS test failures caused by unbound `BUILDKITE_PLUGINS_PATH` variable in hooks/command

## [v1.6.0]

### Added
* Add support for `key` in watch config steps by @toadzky in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/133

### Changed
* Perform an update check instead of downloading the binary every run by @JanSF in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/127

## [v1.5.2]

### Added
* Add support for `depends_on` in step definitions by @jasonwbarnett in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/123

### Changed
* Perform an update check instead of downloading the binary every run by @JanSF in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/122
* Revert the update check change by @pzeballos in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/126
* Maintenance updates to Go/tooling dependencies by @renovate[bot]

### Fixed
* Fix unicode handling in git diff output by @scadu in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/131

## [v1.5.1]

### Fixed
* Fix nested envs parsing as rawenv by @mcncl in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/108

## [v1.5.0]

### Added
* Add env and metadata attributes to steps by @jykingston in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/74
* Add compatibility table by @mcncl in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/81
* Add OSSF scanning by @mcncl in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/90
* Add support for conditionals (`if`) in steps by @jasonwbarnett in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/92

## [v1.4.0]

### Added
* Preserve `plugins:` blocks in watched steps by @dstrates in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/50
* Support `branches` in steps by @toote in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/52
* Support multiple steps by @toote in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/53
* Add `except_path` functionality by @lswith in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/72

### Changed
* Reconfigure tests and remove BATS by @petetomasik in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/48
* Update dependencies by @toote in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/73
* Remove pull request action by @pzeballos in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/70
* Update broken link to original author by @adikari in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/49

## [v1.3.0]

### Fixed
* Use regex for version matching by @petetomasik in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/45

### Changed
* Update release version by @pzeballos in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/39
* Contribution docs by @stephanieatte in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/36
* Update README for v1.3.0 by @petetomasik in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/46

## [v1.2.0]

### Fixed
* Fix typo in README.md by @greenled in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/37

### Changed
* External contributions by @toote in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/35
* feat: add `skip_path` option in `watch` by @jamietanna in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/27
* Default config structure by @toote in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/38

## [v1.1.0]

### Fixed
* Updates to CI, testing and bug fixes by @pzeballos in #7
* Fix default env test by @sj26 in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/33

### Changed
* Skip pipeline upload when no steps by @stephanieatte in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/16
* Use a more portable shebang by @mcncl in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/25
* Support using default config on no path match by @mcncl in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/29
* Escape pipeline interpolation by @sj26 in https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/26

## [v1.0.1]

### Changed
* e6c92d3 Update command

## [v1.0.0]

### Changed
* Update to README.md  ([#1](https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/pull/1)) @stephanieatte
