# Documentation Update Command

Review recent source code updates or local git diffs in detail, and update the relevant documentation files.

## Target Files

- README.md
- CLAUDE.md
- llms.md

## Basic Rules

- Add missing content to documentation files where information is lacking
- Update documentation content where there are discrepancies with the current specifications
- Maintain consistency with existing document structure and formatting. Do not overemphasize content or overuse emojis

## Document-Specific Rules

### README.md

- This is documentation for users of the apcdeploy command
- Strive for simple and clear descriptions that allow users to use the tool smoothly
- Do not add sections indiscriminately. Respect the existing format. If there is no appropriate place to add content within existing sections, first propose adding a new section and obtain permission before executing

### CLAUDE.md

- This is documentation for developers of the apcdeploy project
- Do not arbitrarily rewrite the Development Rules. These are critical rules for ensuring project quality
- Ensure that developers have comprehensive information without excess or omission
- Emphasize the overall design of the project and carefully document other points that should be known during implementation

### llms.md

- This is AI-oriented context information that should be utilized when command users leverage LLM AI agents to use the apcdeploy command
- By executing `apcdeploy context`, users and their AI agents can access the contents of this file
- Write as much detailed information as possible about command specifications and usage
- Carefully and thoroughly describe features that people unfamiliar with the command often overlook, common mistakes, and risks related to execution
- apcdeploy has some interactive features and long-running features like wait, but these are poorly suited for use via AI. Therefore, for AI-oriented information, avoid using these features as much as possible. Conversely, for information assuming human execution, provide guidance about these features as well

ultrathink
