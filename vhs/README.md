# VHS demo

Tape file for the README usage GIF, generated with
[VHS](https://github.com/charmbracelet/vhs).

## Prerequisites

- `vhs` itself is provisioned by `mise install` (pinned in `.mise.toml`).
  Install its runtime dependencies once:

  ```bash
  brew install ttyd ffmpeg
  ```

- A valid AWS session for the resources declared in
  `vhs/fixtures/apcdeploy.before.yml` (`test-app` / `test-prof` / `test-env`
  in `ap-northeast-1`).
- A custom deployment strategy named `fastest` (0s deploy + 0s bake) in the
  same region. The tape uses it to seed the "before" state quickly.

## Generate

```bash
mise run demo
```

Builds `apcdeploy` and runs `vhs vhs/demo.tape`. Output: `vhs/demo.gif`.

## How the tape works

Inside a hidden setup block:

1. Restores `apcdeploy.yml` and `data.json` at the repo root from
   `vhs/fixtures/`.
2. Runs `apcdeploy run --wait-bake --force` with the `fastest` strategy so
   the AWS-side state matches the fixture before recording starts.
3. Rewrites `deployment_strategy` to
   `AppConfig.Linear50PercentEvery30Seconds` so the recorded run shows a
   visible deploy progress bar.

The recorded section then runs:

```
cat → # comment → sed → cat → clear → apcdeploy diff → apcdeploy run --wait-deploy
```

`PATH` is prepended with the repo root inside the recording shell so commands
appear as `apcdeploy ...` rather than `./apcdeploy ...`.

## Notes

- Each invocation issues two real deployments (one in setup, one recorded).
  `apcdeploy.yml` and `data.json` at the repo root are git-ignored, so the
  per-run rewrites do not leak into commits.
- `Set PlaybackSpeed 4.0` compresses the ~110s recording into a ~28s GIF.
  Tweak `PlaybackSpeed`, `TypingSpeed`, and the per-step `Sleep` values in
  `vhs/demo.tape` if the pacing feels off.
