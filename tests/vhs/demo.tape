# demo.tape — generate the hero GIF for the README.
# Run: make demo

Source tests/vhs/_setup.tape

Output doc/demo.gif

Set Width 1200
Set Height 700
Set Padding 16
Set FontSize 13
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Hide
Type "./jara --demo"
Enter

# ── Model overview ─────────────────────────────────────────────────────────────
# Start on the model detail view: split pane with applications + relations.

Wait+Screen /Applications/
Wait+Screen /grafana/
Wait+Screen /Relations/
Show

Sleep 2s

# Browse applications with vim keys.
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "k"
Sleep 300ms

# ── Units view ─────────────────────────────────────────────────────────────────

Type "U"
Wait+Screen /UNIT/
Sleep 2s

Escape
Wait+Screen /Applications/
Sleep 800ms

# ── Relations view ─────────────────────────────────────────────────────────────

Type "R"
Wait+Screen /ENDPOINT/
Sleep 2s

Escape
Wait+Screen /Applications/
Sleep 800ms

# ── Machines view ──────────────────────────────────────────────────────────────

Type "M"
Wait+Screen /INSTANCE/
Sleep 2s

Escape
Wait+Screen /Applications/
Sleep 800ms

# ── Secrets view ───────────────────────────────────────────────────────────────

Type "S"
Wait+Screen /Secrets/
Sleep 2s

# Drill into a secret.
Enter
Sleep 1s

Escape
Sleep 500ms
Escape
Wait+Screen /Applications/
Sleep 800ms

# ── Debug log ──────────────────────────────────────────────────────────────────

Type "L"
Wait+Screen /Waiting for log/
Sleep 500ms
Wait+Screen /juju.worker/
Sleep 2s

# Inline search.
Type "/"
Wait+Screen /Search/
Sleep 200ms
Type "hook"
Enter
Wait+Screen /match/
Sleep 1s

# Close search and return.
Escape
Wait+Screen /Applications/
Sleep 800ms

# ── Controllers ────────────────────────────────────────────────────────────────

Type ":"
Sleep 200ms
Type "controllers"
Enter
Wait+Screen /prod-aws/
Wait+Screen /staging-gce/
Wait+Screen /dev-local/
Sleep 2s

# Drill into a controller → models list.
Enter
Wait+Screen /staging/
Sleep 2s

# Enter model → back to model detail view.
Enter
Wait+Screen /Applications/
Wait+Screen /Relations/
Sleep 2s


# ── Help modal ─────────────────────────────────────────────────────────────────

Type "?"
Sleep 2s

Escape
Wait+Screen /Applications/
Sleep 1s

Hide
Type "q"
Sleep 500ms
