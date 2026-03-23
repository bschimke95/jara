# readonly_mode.tape — verify write operations are blocked in read-only mode.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/readonly_mode.ascii

Type "./jara --demo --readonly"
Enter

# Wait for model detail view.
Wait+Screen /Applications/
Wait+Screen /grafana/
Sleep 1s

# Attempt to scale up (+ key) — should show read-only error briefly.
Type "+"
Sleep 2s

Type "q"

Sleep 500ms
