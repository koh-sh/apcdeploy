# VHS tape for the README demo GIF. Run with `mise run demo`.
# See vhs/README.md for prerequisites (AWS session, `fastest` strategy,
# ttyd + ffmpeg).

Output vhs/demo.gif

Set Shell zsh
Set FontSize 16
Set Width 1100
Set Height 720
Set Theme "Catppuccin Mocha"
# PlaybackSpeed 4.0 compresses the ~110s recording into a ~28s GIF.
# TypingSpeed and each Sleep below are pre-multiplied so the on-screen
# pacing (value / 4) stays comfortable.
Set PlaybackSpeed 4.0
Set TypingSpeed 160ms
Set Padding 20

# --- Setup (not recorded) -------------------------------------------
# Pin AWS-side state to the "before" fixture using `fastest` (0s deploy /
# 0s bake), then switch the strategy to Linear50PercentEvery30Seconds so
# the recorded run shows a visible deploy progress bar.
Hide
Type "setopt interactive_comments"
Enter
Sleep 100ms
Type `export PATH="$PWD:$PATH"`
Enter
Sleep 100ms
Type "cp vhs/fixtures/apcdeploy.before.yml apcdeploy.yml"
Enter
Sleep 200ms
Type "cp vhs/fixtures/data.before.json data.json"
Enter
Sleep 200ms
Type "apcdeploy run --wait-bake --force"
Enter
Sleep 15s
Type "sed -i '' 's/^deployment_strategy:.*/deployment_strategy: AppConfig.Linear50PercentEvery30Seconds/' apcdeploy.yml"
Enter
Sleep 200ms
Type "clear"
Enter
Sleep 200ms
Show
# --------------------------------------------------------------------

Type "cat data.json"
Enter
Sleep 6s

Type "# Enable new_search and raise the upload limit"
Enter
Sleep 2.5s

Type `sed -i '' 's/"new_search": false/"new_search": true/; s/"upload_mb": 10/"upload_mb": 25/' data.json`
Enter
Sleep 2s

Type "cat data.json"
Enter
Sleep 6s

# Visible reset so the next command starts at the top of the viewport.
Type "clear"
Sleep 1.2s
Enter
Sleep 1.5s

Type "apcdeploy diff"
Enter
Sleep 10s

Type "apcdeploy run --wait-deploy"
Enter
Sleep 70s

# Hold the final frame so the completion line stays readable.
Sleep 12s
