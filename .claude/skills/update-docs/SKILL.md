---
name: update-docs
description: Review recent code changes and update README.md, CLAUDE.md, and llms.md to match. Use after implementing features or changing CLI behavior
disable-model-invocation: true
argument-hint: "[target-file]"
allowed-tools: Bash(git diff:*) Bash(git log:*) Bash(git status:*) Bash(git show:*)
---

# Update Documentation

Review recent source changes (or the local git diff) and update the project's documentation to match.

If an argument is provided (`$ARGUMENTS`), limit the update to that file. Otherwise, review all three targets below.

## Target files

- `README.md` — apcdeploy users (humans)
- `llms.md` — apcdeploy users (AI agents), served by `apcdeploy context`
- `CLAUDE.md` — apcdeploy developers (incl. Claude Code)

`llms.md` is **not** a substitute for `CLAUDE.md`: different audiences. Do not move content between them without checking the audience matches.

## Rules

The shared documentation conventions live in `.claude/rules/documentation.md` and auto-activate when you edit these files. The essentials:

- Add missing content; fix discrepancies with current behavior.
- Preserve existing structure, formatting, and tone. Do not overemphasize or overuse emojis.
- `README.md` — user-facing. Match the existing command-section format **exactly**. Do not invent subsections such as "Key characteristics" or "When to use". Do not compare commands with each other. If uncertain about format, ask first.
- `CLAUDE.md` — developer-facing. Follow Claude Code best practices: keep it short, do not duplicate source/README/llms.md/rules, prefer `@` imports over inline copies. For each line ask "would removing this cause Claude to make a mistake?" — if not, cut it.
- `llms.md` — AI-oriented context delivered by `apcdeploy context`. Cover specs, common mistakes, and execution risks. For AI-oriented guidance, bias away from interactive / `--wait-*` features; for human-oriented guidance, include them. Operational rules (per-command template, Errors/Exit Codes policy, compression criteria) live in `.claude/rules/llms-md.md` — consult it before any non-trivial edit.

## Workflow

1. Inspect recent changes: `git diff`, `git log -n 20 --oneline`.
2. For each target file (or the one passed as argument), look for: drift (out of date), audience mismatch (in the wrong file), and bloat (could be removed without losing information).
3. **Default to subtraction over addition.** Propose deletions and moves before additions. Do not grow a doc unless the new content earns its place.
4. Propose a plan before editing if the change is non-trivial. Apply edits preserving existing tone and structure.

ultrathink
