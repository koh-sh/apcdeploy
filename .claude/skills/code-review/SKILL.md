---
name: code-review
description: Conduct a thorough Go code review of the apcdeploy project, covering idiomatic Go, testing conventions, and project-specific rules such as AWS List API usage and the Factory pattern
disable-model-invocation: true
allowed-tools: Bash(git diff:*) Bash(git log:*) Bash(git grep:*) Bash(git status:*) Bash(mise run lint:*) Bash(mise run test:*) Bash(go vet:*) Bash(go doc:*)
---

# Code Review

You are a Go language professional. Conduct a thorough code review of this project.

CLAUDE.md in the repository root holds the authoritative architecture and project conventions; consult it for rules referenced below (Development Rules, AWS List API Usage, Testing Patterns, etc.).

## Focus areas

### Code

- Is the code idiomatic Go, following the standard library's philosophy?
- Is the coding style consistent across subcommands? Flag any design drift.
- Does it follow DRY? Are duplicated patterns extracted into helpers?
- Is the code readable and maintainable?
- Are package and function responsibilities appropriate and single-purpose?
- Are any exports unnecessary? Identifiers used only within their own package should be unexported.
- Do function and variable names clearly describe their role?
- Are comments sufficient where logic is non-obvious? (Not for self-evident code.)

### Tests

- Are tests consistently written as table-driven tests?
- Are there any tests that exist only to boost coverage without verifying meaningful behavior?

### Documentation

- Is the README clear and simple for users?
- Does the README match the current implementation?

### Project-specific

- Is the Factory pattern used to keep executors testable?
- Are interfaces (`AppConfigAPI`, `ProgressReporter`, `Prompter`, …) used appropriately?
- Are Cobra commands thin, with logic in executors?
  - Exception: the `context` command is a self-contained utility (no `internal/context/` package).
- Do error messages follow the "lowercase start, no trailing period" convention?
- **AWS List API usage**: are List operations routed through `internal/aws/client_list_paginated.go`?
  - Use `ListAllApplications()`, `ListAllConfigurationProfiles()`, etc.
  - Direct SDK calls like `client.AppConfig.ListApplications()` are forbidden outside `client_list_paginated.go` itself.

## Output

Produce a structured review with:

1. Summary — overall health.
2. Findings — grouped by severity (must-fix / should-fix / nit), each with file:line references.
3. Suggested next steps.

ultrathink
