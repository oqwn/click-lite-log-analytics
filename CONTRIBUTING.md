# Contributing to Click-Lite Log Analytics

We love your input! We want to make contributing to Click-Lite Log Analytics as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Pull Request Process

1. Update the README.md with details of changes to the interface, if applicable.
2. Update the docs with any new environment variables, exposed ports, useful file locations, and container parameters.
3. Increase the version numbers in any examples files and the README.md to the new version that this Pull Request would represent.
4. You may merge the Pull Request once you have the sign-off of two other developers, or if you do not have permission to do that, you may request the second reviewer to merge it for you.

## Any contributions you make will be under the MIT Software License

In short, when you submit code changes, your submissions are understood to be under the same [MIT License](http://choosealicense.com/licenses/mit/) that covers the project. Feel free to contact the maintainers if that's a concern.

## Report bugs using GitHub's [issues](https://github.com/your-username/click-lite-log-analytics/issues)

We use GitHub issues to track public bugs. Report a bug by [opening a new issue](); it's that easy!

## Write bug reports with detail, background, and sample code

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

## Use a Consistent Coding Style

### Go Code Style

* Run `gofmt` and `goimports` on your code
* Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
* Use `golangci-lint` to check for common issues

### JavaScript/React Code Style

* Use ESLint with the provided configuration
* Follow the [Airbnb JavaScript Style Guide](https://github.com/airbnb/javascript)
* Use Prettier for code formatting

### General Guidelines

* 2 spaces for indentation in YAML and JSON files
* 4 spaces for indentation in Go files
* Use meaningful variable and function names
* Write self-documenting code and add comments only when necessary
* Keep functions small and focused on a single task

## Testing

* Write unit tests for new functionality
* Ensure all tests pass before submitting PR
* Aim for >80% code coverage
* Include integration tests for API endpoints

## License

By contributing, you agree that your contributions will be licensed under its MIT License.