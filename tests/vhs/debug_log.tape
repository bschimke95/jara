# debug_log.tape — verify the debug log view, inline search, and filter modal.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/debug_log.ascii

Hide
Type "./jara --demo"
Enter

# Wait for model view to load with data.
Wait+Screen /Applications/
Wait+Screen /grafana/
Sleep 1s

# Open debug log with 'l'.
Type "l"
Wait+Screen /Waiting for log/
Sleep 500ms

# Wait for the first log entries to arrive (mock sends every 500ms).
Wait+Screen /juju.worker/
Sleep 1s

# ── Inline search ──────────────────────────────────────────────────────────────

# Open search bar with '/'.
Type "/"
Wait+Screen /Search/
Sleep 300ms

# Type a search query that matches known mock messages.
Type "hook"
Enter
Wait+Screen /match/
Sleep 1s

# Navigate to next match.
Type "n"
Sleep 500ms

# Navigate to previous match.
Type "N"
Sleep 500ms

# ── Filter modal ───────────────────────────────────────────────────────────────

# Open filter modal with 'F'. The search indicator stays in the background.
Type "F"
Wait+Screen /Debug Log Filter/
Sleep 500ms

# 'Level' pane is selected by default — move right to the level list.
Type "l"
Wait+Screen /TRACE/
Sleep 300ms

# Select WARNING level (down three times from TRACE: DEBUG → INFO → WARNING).
Down
Down
Down
Enter
Sleep 300ms

# Apply the filter with 'F'.
Type "F"
Wait+Screen /level=WARNING/
Sleep 1s

# ── Dismiss modal with Escape ──────────────────────────────────────────────────

# Re-open the filter modal and dismiss it with Escape (right pane → left pane →
# close). Each Escape should stay in the debug log view, not navigate back.
Type "F"
Wait+Screen /Debug Log Filter/
Sleep 300ms
Type "l"
Wait+Screen /TRACE/
Sleep 300ms
Escape
Wait+Screen /Debug Log Filter/
Sleep 300ms
Escape
Wait+Screen /level=WARNING/
Sleep 500ms

# ── Clear filter ───────────────────────────────────────────────────────────────

# Clear the active filter with 'D'.
Type "D"
Sleep 1s

# ── Back ───────────────────────────────────────────────────────────────────────

Escape
Wait+Screen /Applications/
Wait+Screen /Relations/
Show

Sleep 500ms

Hide
Type "q"
Sleep 500ms
