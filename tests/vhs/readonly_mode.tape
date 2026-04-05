# readonly_mode.tape — verify write operations are blocked in read-only mode.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/readonly_mode.ascii

Hide
Type "./jara --demo --readonly"
Enter

# Wait for model detail view.
Wait+Screen /Applications/
Wait+Screen /grafana/
Sleep 1s

# Attempt to scale up (+ key) — should show read-only error briefly.
Type "+"
Show

Sleep 2s

Hide
Type "q"
Sleep 500ms
