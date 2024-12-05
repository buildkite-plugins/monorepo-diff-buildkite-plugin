# Contributing

Thank you for your interest in contributing to this project!

Before opening a pull request, please review and follow the guidelines outlined in this document. Additionally, be sure to read our [Code of Conduct](https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/blob/master/CODE_OF_CONDUCT.md) to understand expected behavior within the project.

If you plan to submit a pull request, we ask that you first create an [issue](https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/issues). For new features or modifications to existing functionality, please start a discussion with the maintainers. For straightforward bug fixes, an issue is enough without a preliminary discussion.

## Developing

To get started with development:

- [Install Go](https://golang.org/doc/install).
- Ensure you have `make` installed, as this project uses a Makefile.
- Fork this repository and clone your fork locally.
- Make your changes (see [Formatting](https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/blob/master/CONTRIBUTING.md#formatting)) and commit to your fork. Use [Conventional Commits](https://www.conventionalcommits.org/) style for commit messages.
- Add relevant unit tests (see [Testing](https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/blob/master/CONTRIBUTING.md#testing)) for your changes.
- Update documentation if necessary.
- Open a pull request, a maintainer will assist in reviewing the pull request
- After review and approval, a maintainer will merge your pull request and create a release (see [Releasing](https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/blob/master/CONTRIBUTING.md#releasing)).

## Testing

All changes must be unit tested, and test coverage should meet the project's minimum threshold of 73%. Run `make test` to execute all tests and generate coverage reports before submitting a pull request.

For `bats` plugin tests:
1. Modify the tests as needed.
2. Run `make build-docker-test && make test-docker` to build the Docker image and execute the tests within the Docker container.

## Formatting

Please format all code with `gofmt` using the latest version of Go and ensure it passes `go vet`. Additionally, the plugin must be linted with the [buildkite-plugin-linter](https://github.com/buildkite-plugins/buildkite-plugin-linter).

## Releasing

Once a pull request is merged, a maintainer will create a new release:
- Confirm that documentation is updated as needed.
- Update all plugin version references in [README.md](https://github.com/buildkite-plugins/monorepo-diff-buildkite-plugin/blob/master/README.md).
- Create and push a new version tag.
