# startup.tape — verify jara launches in demo mode and renders the model view.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/startup.ascii

Type "./jara --demo"
Enter

# Wait for the model detail view — applications pane with mock data.
Wait+Screen /Applications/
Wait+Screen /grafana/

Sleep 2s

# Quit cleanly.
Type "q"

Sleep 500ms
