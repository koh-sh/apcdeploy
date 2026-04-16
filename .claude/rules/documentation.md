---
paths:
  - "README.md"
  - "CLAUDE.md"
  - "llms.md"
---

# Documentation Rules

Shared principles for `README.md`, `CLAUDE.md`, and `llms.md`:

- Add missing content where information is lacking.
- Update discrepancies with the current implementation.
- Maintain existing structure and formatting. Do not overemphasize content or overuse emojis.

## `README.md`

- Audience: users of the `apcdeploy` command.
- Prioritize simple, clear descriptions so users can run the tool smoothly.
- Do not add sections unilaterally. If no existing section fits, propose a new section and wait for approval.
- **When adding new command documentation, strictly follow the existing format and length of other command sections.**
  - Read existing command sections first.
  - Match the structure: command name, brief description, code example, optional note.
  - Keep descriptions concise (1–2 sentences).
  - Do **not** add extra subsections like "Key characteristics", "When to use", bullet lists, or detailed explanations.
  - Do **not** compare commands with each other (e.g., "Unlike X, this command…").
  - If uncertain about the format, ask before writing.

## `CLAUDE.md`

- Audience: developers of the `apcdeploy` project (and Claude Code).
- Do not unilaterally rewrite the Development Rules; these are quality gates.
- Cover the overall design and anything an implementer should know, without excess.
- Favor `@` imports into `.claude/rules/*.md` over inlining long sections.

## `llms.md`

- Audience: AI assistants (and their human operators) using `apcdeploy`. Accessed via `apcdeploy context`.
- Be thorough on command specifications and usage.
- Document overlooked features, common mistakes, and execution risks.
- `apcdeploy` has interactive and long-running (`--wait-*`) features that are poorly suited to AI use; bias AI-oriented guidance away from them. For human-oriented guidance, cover them.
