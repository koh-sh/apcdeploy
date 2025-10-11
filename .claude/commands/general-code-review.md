# General Code Review Command

You are a Go language professional.
As a professional, please conduct a thorough code review of this project.

Please review with particular attention to the following aspects:

## Code

- Does the code follow Go best practices and philosophy, making it idiomatic Go code?
- Is the coding style consistent? Specifically, are there any design differences between subcommands?
- Does it follow the DRY principle and eliminate duplicate code?
- Is the code sufficiently readable and maintainable?
- Are the responsibilities of packages and functions appropriate? Do they avoid having multiple responsibilities?
- Are there any unnecessary function exports? Functions not currently referenced externally should be kept private
- Are function and variable names clear and representative of their responsibilities?
- Are code comments sufficient? Especially for code or logic that is difficult to understand at first glance

## Tests

- Are test codes consistently written in table-driven test format?
- Are there any meaningless tests that exist only to increase coverage?

## Documentation

- Is the README clear and simple for users?
- Is the README up to date with the latest code specifications?

## Project-Specific

- Is the Factory pattern utilized to ensure testability?
- Are interfaces (AppConfigAPI, ProgressReporter, Prompter, etc.) used appropriately?
- Is there proper separation of responsibilities between Cobra commands and executors? Cobra commands should contain minimal logic
- Do error messages follow the convention of lowercase start and no trailing period?

ultrathink
