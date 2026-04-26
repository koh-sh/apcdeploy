# Output Contract

This contract governs how every `apcdeploy` command produces output. It exists so
human-facing presentation stays uniform across commands while machine-readable
data remains script-friendly.

The contract is enforced by the `internal/reporter` interface and its
`internal/cli` implementations. Executors MUST NOT call `fmt.Fprint*` directly;
all output flows through the `Reporter`.

## Channels

| Channel | Purpose | `--silent` |
|---|---|---|
| stdout | Machine-readable payload that scripts/pipes consume | Always emitted |
| stderr | Human-readable progress, structure, errors | Most kinds suppressed (see below) |

A command must pick exactly one stdout payload (or none). Examples: `get` writes
the configuration body, `diff` writes the unified diff, `ls-resources` writes
the formatted tree (or JSON), `status --silent` writes the deployment state,
`context` writes `llms.md`. Everything else goes to stderr.

## Output kinds

The Reporter exposes the following methods. Each kind has a fixed channel,
silent-mode behavior, and visual treatment.

| Kind | Channel | `--silent` | Visual (TTY) | Use for |
|---|---|---|---|---|
| `Step(msg)` | stderr | suppressed | `⏳` + dim | Starting a long-running step |
| `Success(msg)` | stderr | suppressed | `✓` + green | Step completion |
| `Info(msg)` | stderr | suppressed | `ℹ` + cyan | Neutral information ("no deployment found, creating without data") |
| `Warn(msg)` | stderr | suppressed | `⚠` + yellow | Non-fatal anomaly the user should notice |
| `Error(msg)` | stderr | **always shown** | `✗` + red | Fatal error (used only by `cmd/root.go`) |
| `Header(title)` | stderr | suppressed | bold + rule line | Section heading (e.g. "Configuration Diff") |
| `Box(title, lines)` | stderr | suppressed | bordered card | Multi-line panel (e.g. init's "Next steps") |
| `Table(headers, rows)` | stderr | suppressed | lipgloss table | Structured key/value or row data |
| `Spin(msg) Spinner` | stderr | suppressed; non-TTY: `Step` once | animated frames | Live indicator wrapping a long call |
| `Data(p []byte)` | **stdout** | **always shown** | none | Machine-readable payload |
| `Diff(p []byte)` | **stdout** | **always shown** | colorized when TTY | Unified diff payload |

`Spinner` is `interface { Done(msg string); Fail(msg string) }`. `Done` emits a
`Success`-equivalent line; `Fail` emits an `Error`-equivalent line on stderr.

## Confirmations

Confirmation prompts (e.g. `rollback`, `get` cost confirmation) are NOT a
Reporter kind — they live in `internal/prompt`. The contract for confirmations:

- A `--yes` style flag MUST be available to bypass the prompt.
- When stdin is not a TTY and `--yes` was not supplied, the command MUST exit
  with `prompt.ErrNoTTY` (which the user message points at the bypass flag).
- The prompt MUST be shown via `prompt.Prompter`; the surrounding context
  (deployment summary, cost notice, etc.) MUST be rendered through Reporter
  primitives so silent mode behaves consistently.

## Silent mode contract

`--silent` (alias `-s`) is a global flag that swaps the concrete Reporter for a
silent variant. Rules:

- Silent mode suppresses Step / Success / Info / Warn / Header / Box / Table /
  Spin entirely.
- Silent mode preserves Error (always to stderr) and Data / Diff (always to
  stdout) so scripts still receive errors and payloads.
- Silent mode does NOT change confirmation behavior — the user still must pass
  `--yes` to bypass.
- Executors MUST NOT branch on `opts.Silent`. Reporter selection in
  `cmd/root.go` is the single source of truth. (The `Silent` field is kept on
  Options structs only because Cobra binds the flag there.)

## TTY degradation

When stderr is not a TTY (CI, pipes, redirects), the Reporter degrades:

- `Spin` collapses to a single `Step` line; `Spinner.Done`/`Fail` emit a
  matching `Success`/`Error` line.
- `Header` / `Box` / `Table` drop borders and color but keep structure (plain
  text with the same content).
- `Diff` drops ANSI color so piped consumers get clean text.
- `Data` is always raw bytes, regardless of TTY.

## Color and accessibility

- All color comes from `lipgloss` styles defined in `internal/cli/style.go`.
- `lipgloss` honors `NO_COLOR`; setting it disables color globally.
- Symbols (`⏳ ✓ ℹ ⚠ ✗`) MUST be the only emoji-like glyphs used. No other
  emoji in CLI output.

## Documented exceptions

Exceptions are limited and each one must be justified inline at the call site
with a `CONTRACT EXCEPTION` comment that links back to this section.

### diff in-progress warning

`internal/diff/display.go::displayDeploymentWarning` writes its notice
directly to `os.Stderr` (via a package-level writer that tests can swap)
instead of going through `Reporter.Warn`.

Why: the original `diff` implementation deliberately surfaced this notice
under `--silent` because an in-flight deployment can be rolled back
mid-rollout, changing what the diff is taken against. Scripts in CI/automation
need to see this risk even when they otherwise want machine-readable output.
Routing through `Reporter.Warn` would suppress it under `--silent`, so the
notice bypasses the Reporter for this single case.

This is the only sanctioned bypass. New callers must not introduce more
without adding a similar entry here.

## What MUST NOT happen

- No `fmt.Fprint*` to `os.Stderr` or `os.Stdout` from `internal/<cmd>/` or
  `cmd/`. The only legal direct write is via Reporter (or via a writer the
  caller injected, e.g. the `io.Writer` parameter on `lsresources`).
- No raw `\033[...m` ANSI escape codes anywhere except inside `internal/cli`.
- No emoji or symbol prefixes in messages passed to Reporter — the Reporter
  prepends them. Pass the message text only.
- No bespoke "section header" or "table" string-builders in feature packages.
  Use `Reporter.Header` / `Reporter.Table` / `Reporter.Box`.

## Adding a new command

1. Decide what (if anything) goes to stdout — that is your `Data` or `Diff` payload.
2. Pick Reporter kinds for everything else. A typical command issues:
   `Step` → `Success` for each phase, optional `Header` + `Table` for a final
   summary, `Box` for next-step guidance.
3. Wire the command to `cli.GetReporter(silent)` in `cmd/<name>.go`.
4. Do not branch on `opts.Silent` inside the executor.
