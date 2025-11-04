# PR Closure Notice

Thank you for your contribution and for taking the time to work on this pull request! We appreciate your effort and interest in improving the monorepo-diff-buildkite-plugin.

## Why We're Closing This PR

After careful review, we've determined that this PR needs significant additional work before it can be merged. Additionally, we wanted to bring to your attention that Buildkite now natively supports the `if_changed` property directly from the agent as an agent-applied attribute.

You can read more about this feature in the official Buildkite documentation:
https://buildkite.com/docs/pipelines/configure/step-types/trigger-step#agent-applied-attributes

## Moving Forward

If you're still interested in contributing to this project, we encourage you to:

1. Consider whether the native `if_changed` functionality meets your needs
2. If you still believe this enhancement is valuable, please open a new PR that incorporates:
   - The feedback and suggestions from this PR review
   - Consideration of how it works with the native `if_changed` feature
   - Updated tests and documentation
   - A clear explanation of the use case that isn't covered by the native functionality

We value your contributions and look forward to any future PRs you may submit. If you have questions or would like to discuss this further, please feel free to open an issue or reach out to the maintainers.

Thank you again for your interest in the project!
