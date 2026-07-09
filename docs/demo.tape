# VHS tape for the README hero GIF — records the live TUI with a key tour.
# Regenerate: vhs docs/demo.tape  (needs a Prometheus with pilot metrics;
# swap the config/prom flag for any live setup).

Output docs/demo.gif

Set FontSize 14
Set Width 1400
Set Height 580
Set Padding 12
Set WindowBar Colorful

Hide
Type "go build -o grom ./cmd/grom && clear"
Enter
Sleep 3s
Show

Type "./grom run --config examples/pilot.yaml --prom http://localhost:9093"
Sleep 500ms
Enter
Sleep 4s

# Focus tour: right, right, down.
Type "l"
Sleep 800ms
Type "l"
Sleep 800ms
Type "j"
Sleep 1.2s

# Zoom the focused chart, then back.
Enter
Sleep 3s
Escape
Sleep 1.2s

# Help overlay.
Type "?"
Sleep 3s
Escape
Sleep 1s

# Cycle themes: tokyo-night, catppuccin-mocha, back to pilot.
Type "t"
Sleep 2s
Type "t"
Sleep 2s
Type "t"
Sleep 2s

Type "q"
Sleep 500ms
