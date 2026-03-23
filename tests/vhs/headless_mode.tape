# headless_mode.tape — verify the header panel is hidden in headless mode.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/headless_mode.ascii

Type "./jara --demo --headless"
Enter

# Wait for model view — header box should be absent.
Wait+Screen /Applications/
Wait+Screen /grafana/

Sleep 2s

Type "q"

Sleep 500ms
