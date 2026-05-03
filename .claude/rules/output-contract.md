# Output Contract

This contract governs how every `apcdeploy` command produces output. It exists so
human-facing presentation stays uniform across commands while machine-readable
data remains script-friendly.

The contract is enforced by the `internal/reporter` interface and its
`internal/cli` implementations. Executors MUST NOT call `fmt.Fprint*` directly;
all output flows through the `Reporter`.

The visual model and per-command screen designs live in
`docs/design/output.md`. This file is the implementation-level contract — the
"rules" — while `output.md` is the "picture".

## Channels

| Channel | Purpose | `--silent` |
|---|---|---|
| stdout | Machine-readable payload that scripts/pipes consume | Always emitted |
| stderr | Human-readable progress, structure, errors | Most kinds suppressed (see below) |

A command must pick exactly one stdout payload (or none). Examples: `get` writes
the configuration body, `diff` writes the unified diff, `ls-resources --json`
writes the JSON tree (the human-readable view goes through Reporter primitives
on stderr), `status --silent` writes the deployment state, `context` writes
`llms.md`. Everything else goes to stderr.

## Output kinds

The Reporter exposes the following methods. Each kind has a fixed channel,
silent-mode behavior, and visual treatment.

| Kind | Channel | `--silent` | Visual (TTY) | Use for |
|---|---|---|---|---|
| `Targets(ids) Targets` | stderr | suppressed; non-TTY: line per phase transition / threshold | identifier-aligned multi-row block with state icon, phase, optional progress bar | All deployment-target lifecycles (`run` / `diff` / `pull` / `rollback` / `edit` / `get --yes` / `status`) |
| `Header(title)` | stderr | suppressed | bold + rule line | Section heading (e.g. `init` / `ls-resources`) |
| `Box(title, lines)` | stderr | suppressed | bordered card | Multi-line panel (`init`'s "Next steps", `status`'s no-deployment guidance) |
| `Table(headers, rows)` | stderr | suppressed | lipgloss table | Structured key/value or row data (`status` detail, `ls-resources`) |
| `Warn(msg)` | stderr | suppressed | `⚠` + yellow | Non-fatal anomaly the user should notice (`get` cost notice) |
| `Error(msg)` | stderr | **always shown** | `✗` + red | Fatal error (used only by `cmd/root.go`) |
| `Step(msg)` | stderr | suppressed | `⏳` + dim | **`init` only** — sequential workflow step announcement |
| `Success(msg)` | stderr | suppressed | `✓` + green | **`init` only** — sequential workflow step completion |
| `Info(msg)` | stderr | suppressed | `ℹ` + cyan | **`init` only** — neutral information ("no deployment found, creating without data") |
| `Spin(msg) Spinner` | stderr | suppressed; non-TTY: silent until `Done`/`Fail` emits the completion line | animated frames | **`init` only** — single-phase live indicator wrapping a long call |
| `Data(p []byte)` | **stdout** | **always shown** | none | Machine-readable payload |
| `Diff(p []byte)` | **stdout** | **always shown** | colorized when TTY | Unified diff payload |

`Targets` is the primary primitive (see `docs/design/output.md` §4). Every
deployment-target command tracks its lifecycle through a `Targets` block where
each `id` is one row and the row's state icon, phase label, and optional
progress bar are driven from the executor. `Step` / `Success` / `Info` /
`Spin` are retained only for `init`, which is fundamentally a sequential
interactive workflow that does not fit the target-centric model
(`docs/design/output.md` §11 Q-1).

`Targets` is `interface { SetPhase(id, phase, detail string); SetProgress(id
string, percent float64, eta time.Duration); Done(id, summary string);
Fail(id string, err error); Skip(id, reason string); Close() }`. All
identifiers MUST be supplied to the constructor up front — the implementation
precomputes column widths and cannot accept new rows mid-flight. `Done` /
`Fail` / `Skip` are sticky: subsequent calls against the same id are
ignored. Callers MUST `defer tg.Close()` immediately after construction;
forgetting Close leaks the rendering goroutine.

`Spinner` (returned by `Spin`) is `interface { Update(msg string); Done(msg
string); Fail(msg string); Stop() }`. `Update` swaps the animated label
without changing the running state; on non-TTY output it is silent. `Done`
emits a `Success`-equivalent line; `Fail` emits an `Error`-equivalent line;
`Stop` terminates silently.

## Phases and state icons (Targets)

Each Targets row progresses through these states (`docs/design/output.md` §3):

| State | Icon | Color | Meaning |
|---|---|---|---|
| pending | `○` | dim | initial state before `SetPhase` |
| running | `⠋` (spinner) | step blue | active sub-phase (preparing / comparing / creating-version / deploying / baking) |
| running with progress | `█░` bar | green/dim | deploying with quantified rollout % |
| done | `✓` | green | terminal success (`Targets.Done`) |
| failed | `✗` | red | terminal failure (`Targets.Fail`) |
| skipped | `→` | dim | terminal early-exit / no-op (`Targets.Skip`) |

Phase verbs are limited to: `preparing`, `comparing`, `creating-version`,
`deploying`, `baking`, `fetching`, `stopping`. New verbs require an entry in
`output.md` §3.2.

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

- Silent mode suppresses Targets / Step / Success / Info / Warn / Header / Box
  / Table / Spin entirely. `Targets.Fail` is the lone exception — its
  underlying error is forwarded through `Error` so fatal failures still
  surface in scripts.
- Silent mode preserves Error (always to stderr) and Data / Diff (always to
  stdout) so scripts still receive errors and payloads.
- Silent mode does NOT change confirmation behavior — the user still must pass
  `--yes` to bypass.
- Executors MUST NOT branch on `opts.Silent`. Reporter selection in
  `cmd/root.go` is the single source of truth. (The `Silent` field is kept on
  Options structs only because Cobra binds the flag there.)

## TTY degradation

When stderr is not a TTY (CI, pipes, redirects), the Reporter degrades:

- `Targets`:
  - In TTY mode the rows redraw in place on every state / progress change.
  - In non-TTY mode each phase transition emits a new `<id>: <phase>
    [<detail>]` line and progress is decimated to 25 / 50 / 75 / 100 %
    thresholds. Done / Fail / Skip emit a single terminal line per row.
- `Spin` stays silent until the caller invokes `Done`/`Fail`, which emit a
  single `Success`/`Error` line. The starting message is dropped so logs only
  record terminal states.
- `Header` / `Box` / `Table` drop borders and color but keep structure (plain
  text with the same content).
- `Diff` drops ANSI color so piped consumers get clean text.
- `Data` is always raw bytes, regardless of TTY.

## Color and accessibility

- All color comes from `lipgloss` styles defined in `internal/cli/style.go`.
- `lipgloss` honors `NO_COLOR`; setting it disables color globally.
- Symbols (`✓ ✗ ⊘ ⚠ ℹ ⠋ → █ ░ ⏳ ○`) MUST be the only emoji-like glyphs used.
  `→` and `○` are reserved for `Targets` (skip / pending), `█` and `░` for
  the progress bar, and the rest are line prefixes for their corresponding
  kinds. `⏳` survives only for `init`'s `Step` lines. No other emoji in CLI
  output.

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
  `cmd/`. The only legal direct write is via Reporter.
- No raw `\033[...m` ANSI escape codes anywhere except inside `internal/cli`.
- No emoji or symbol prefixes in messages passed to Reporter — the Reporter
  prepends them. Pass the message text only.
- No bespoke "section header" or "table" string-builders in feature packages.
  Use `Reporter.Header` / `Reporter.Table` / `Reporter.Box`.

## Adding a new command

1. Decide what (if anything) goes to stdout — that is your `Data` or `Diff`
   payload.
2. Pick Reporter kinds for everything else. The default shape for a
   deployment-target command is:
   - Build the canonical identifier with `config.Identifier(region, cfg)`.
   - Open a `Targets` block, `defer tg.Close()`, drive the row through
     `SetPhase` / `SetProgress` and finalise with `Done` / `Fail` / `Skip`.
   - Optional `Header` + `Table` / `Box` for a final summary or guidance.
   - Do not emit any phase transition for instant operations (file reads,
     validation, content-type detection). Failures surface via the returned
     error; the absence of an error is the success signal.
   - For wait phases, use `run.MakeTargetsDeployTick` /
     `run.MakeTargetsBakeTick` to drive the row from the AWS polling tick
     callbacks.
3. For a non-target sequential workflow (`init`-style), use `Step` /
   `Success` / `Info` / `Spin` instead. New commands should justify why they
   need this path rather than `Targets` — most deployment-flavoured commands
   fit `Targets`.
4. Wire the command to `cli.GetReporter(silent)` in `cmd/<name>.go`.
5. Do not branch on `opts.Silent` inside the executor.

## Resolution hints

When emitting a `Targets.Fail` whose underlying error is an AWS API error,
the command may consult `internal/errors.Resolution(err)` to look up a
short user-facing remediation hint (e.g. "wait for the current deployment to
complete or run 'apcdeploy rollback'"). Hints exist only for the small set of
AWS error codes documented in `internal/errors/resolution.go`; callers MUST
NOT invent new hints inline. To add a hint, add an entry to
`resolutionHints` and document the rationale in
`docs/design/output.md` §8.3.
