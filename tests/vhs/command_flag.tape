# command_flag.tape — verify --command flag starts directly on the specified view.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/command_flag.ascii

Type "./jara --demo --command model"
Enter

# Should start directly on the model detail view, skipping controllers.
Wait+Screen /Applications/

Sleep 2s

Type "q"

Sleep 500ms
