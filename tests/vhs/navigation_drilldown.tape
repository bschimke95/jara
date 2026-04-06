# navigation_drilldown.tape — full drill-down: controllers → models → model view.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/navigation_drilldown.ascii

Hide
Type "./jara --demo"
Enter

# Wait for model detail view.
Wait+Screen /Applications/
Sleep 1s

# Navigate to controllers via command bar.
Type ":"
Sleep 300ms
Type "controllers"
Enter

Wait+Screen /prod-aws/
Sleep 1s

# Enter controller → models list.
Enter

Wait+Screen /staging/
Sleep 1s

# Enter model → model detail view.
Enter

# Wait for applications pane to render.
Wait+Screen /grafana/
Wait+Screen /Relations/
Show

Sleep 2s

Hide
Type "q"
Sleep 500ms
