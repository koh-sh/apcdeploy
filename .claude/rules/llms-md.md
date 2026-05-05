---
paths:
  - "llms.md"
---

# llms.md authoring rules

Operational rules for editing `llms.md`. Read alongside `.claude/rules/documentation.md` (audience-level conventions). When the two conflict, `documentation.md` is the higher-level rule; this file is the concrete how-to.

## What `llms.md` is

`llms.md` is the AI-facing manual for `apcdeploy`. It is a Markdown file in the repository root that documents every CLI command's specification, behavior, error semantics, and gotchas, at a level of detail tailored to LLM-based agents (and the humans directing them).

It is embedded into the `apcdeploy` binary at build time — `main.go` uses `//go:embed llms.md` — and surfaced through the `apcdeploy context` subcommand, which prints the document verbatim to stdout. The expected consumption flow:

1. A user (or their AI agent) runs `apcdeploy context` once at the start of a session.
2. The output is fed into the AI's conversation context.
3. The AI now has authoritative, version-matched guidance for the exact `apcdeploy` binary installed on the user's machine — independent of whatever (possibly stale or hallucinated) knowledge happened to be in its training data.

## Why it exists

This file solves three problems specific to AI-assisted CLI use:

- **Training-data drift.** Published CLIs change faster than model training cutoffs. A binary-embedded manual is always in sync with the binary that ships it.
- **Hallucinated flags and behaviors.** Agents sometimes invent plausible-looking flags or invariants. A canonical source closes that gap for the small surface area `apcdeploy` exposes.
- **Pitfall avoidance.** TTY-only prompts, billable Data API calls (`get`), one-active-deployment-per-environment, `--wait-*` blocking, idempotency rules around FeatureFlags metadata — these are not obvious from `--help` and are exactly the failure modes a first-pass agent walks into. Documenting them centrally lets agents skip them.

## Constraints these properties impose

These three properties dictate every rule in this file:

- **Embedded / single file.** The doc cannot reference `README.md`, `CLAUDE.md`, or anything else in the repo — consumers reading via `apcdeploy context` do not have access to those. Every fact a reader needs must be inside `llms.md`. External links to AWS documentation are fine.
- **Loaded into AI context.** Tokens cost the consumer real budget per session. But coverage of execution risks costs them real *failures*. The trade-off is asymmetric: cut redundancy aggressively, but never cut coverage of a behavior or error condition. "Be thorough" beats "be short" when in doubt.
- **Two audiences, both first-class.** AI assistants and the humans directing them both read this file. Content cannot be cut on the basis that "only one audience needs it" without checking the other has what it needs elsewhere in the doc. AI-targeted guidance and human-targeted guidance can co-exist; they often share the same section with inline parallels.

## Per-command section template

Every `### <name> command` section follows this layout. Sections marked optional are governed by the rules below.

```
### <name> command

<one-sentence description; surface non-obvious framing like "no apcdeploy.yml required" or "billed per call">

#### Usage

```bash
<one canonical invocation>
```

#### Flags

| Flag | Description |
|---|---|
...

#### Operation Details                                 [optional — see rule]

<numbered steps>

#### Behavior

<idempotency, side-effect ordering, AI-bias notes, special cases>

#### Errors                                            [optional — see rule]

| Cause | Resolution |
|---|---|

#### Exit Codes

- `0`: ...
- `1`: ...

#### Examples

```bash
# flag-specific demonstrations only
```
```

### When to include "Operation Details"

Include a numbered step list **only** when the ordering encodes a user-visible contract that is not stated elsewhere. Drop it for simple commands.

| Command | Operation Details? | Why |
|---|---|---|
| `init` | yes | interactive prompt sequence is user-facing |
| `run`, `pull` | yes (compressed) | diff/compare-before-write encodes idempotency |
| `rollback` | yes (compressed) | TTY check fires before any AWS write |
| `ls-resources`, `status`, `diff`, `get`, `context` | no | simple operations; covered by Behavior/Errors |
| `edit` | no | minimal-coverage special case — see "edit command (special case)" below |

### When to include an "Errors" entry

Include an entry **only** when the resolution is not auto-evident from the error message itself. Criterion: an AI agent reading the error text alone could (or could not) figure out the next action.

- Auto-evident → omit. Examples: "max 1024 characters", "deployment timeout after 1800 seconds", "interactive mode requires a TTY: use --yes to skip confirmation".
- Not auto-evident → include. Examples: "another deployment is in progress" (resolution requires knowing `rollback` exists), "no prior deployment exists" (resolution requires running `run` first; exit code 2).

Cross-cutting errors (resource not found, AWS auth failure) live in the top-level `## Troubleshooting` section, not per-command.

### Exit Codes

Always include explicit Exit Codes per command, even when only `0` / `1`. When a command has a command-specific code (e.g. `pull` and `edit` use `2` for "no prior deployment"), document that case in the bullet list.

### Examples

Examples must demonstrate something the Usage block doesn't. Never re-show the basic `apcdeploy <name> -c apcdeploy.yml` form if it is already in Usage. 1–4 examples per command.

## Cross-cutting rules

### Single source of truth

The following live in **one** place only and are forbidden per-command:

- "AWS credentials required" → Overview Constraints.
- TTY/`--yes` requirements → Overview "TTY Requirements" table. Per-command, only mention the bypass flag in the Flags row (or an Errors row when not auto-evident).
- Hosted configuration size limits (2 MB / 4 MB) → Overview Constraints.
- FeatureFlags metadata exclusion (`_createdAt` / `_updatedAt`) → Overview Supported Content Types. Per-command Behavior may reference it briefly when relevant to that command's idempotency.

### Format conventions

- Flags: Markdown table (not bullet list).
- Errors: Markdown table with Cause | Resolution.
- Exit Codes: bullet list.
- Output samples (JSON, human-readable): show **once** with the most informative variant; describe diffs from other variants in prose. Do not repeat full samples for each flag combination.

### `edit` command (special case)

AI must not run `edit` (no non-interactive mode), but the section must still cover what it does so an AI can describe it to humans. Keep the section minimal:

- AI-warning paragraph (must say "AI agents must not run this command" and point at the `pull` → edit → `run` alternative).
- Brief human-targeted description with key flags inline (no flag table).
- One example.
- Exit Codes (including the `2` case).

Do **not** add a full Operation Details, flag table, or expanded examples to `edit`.

### Recommended Usage Flows

Keep AI/human parallels but express them inline (e.g. "AI agents: pass all four flags; human users may omit them for prompts"). Do not duplicate command blocks under separate Human/AI labels. Do not litter with "Human users: vim ..." / "AI agents: Use Edit tool" footnotes — state the convention once at the top of the section if needed.

### Bias for AI vs human

For interactive or long-running features (`init` prompts, `edit`, `--wait-deploy`, `--wait-bake`):

- Document the feature for human users.
- AI-oriented guidance MUST explicitly steer AI away (e.g. "For AI assistants: prefer the no-wait default and poll with `status`").
- Do not promote these features in AI-targeted workflows or examples.

## Compression criteria

A piece of content can be **deleted** if **all** of these hold:

- It is a verbatim or near-verbatim duplicate of content elsewhere in the same doc.
- Its meaning is auto-evident without the duplicate.
- Both AI and human audiences lose nothing meaningful.

A piece of content **must be retained** if any of these hold:

- It encodes a user-visible behavior contract not stated elsewhere (e.g. side-effect ordering, idempotency rules, exit code distinctions).
- It documents a risk, common mistake, or execution gotcha.
- Its resolution is not auto-evident from any error message the user might see.

## Defaults

- "Be thorough" > "be short" when in doubt about whether to cover a behavior.
- "No redundancy" > "be thorough" when there is verbatim duplication.
- Drop before add. Audience is context-budget constrained when read via `apcdeploy context`.
- When unsure whether to cut, look up the audience: if removing serves only one of {AI, human}, do not cut unilaterally.

## Workflow when restructuring

When a change touches more than one command or restructures the file:

1. Pick one command and rewrite it in the new shape first.
2. Verify against this rules file (template followed, Errors filtered correctly, Operation Details judgment correct).
3. Show the draft for approval before propagating.
4. Apply the same template mechanically to the rest. Do not redesign mid-propagation.
