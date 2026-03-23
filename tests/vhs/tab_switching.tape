# tab_switching.tape — navigate from model view to each sub-view and back.
# The U/A/R keys work from the model detail view (split pane).
# Each sub-view is standalone — use Esc to return to the model view.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/tab_switching.ascii

Type "./jara --demo"
Enter

# Wait for model view — starts on applications pane.
Wait+Screen /Applications/
Sleep 1s

# Switch to standalone units view (Shift+U).
Type "U"
Wait+Screen /UNIT/
Sleep 2s

# Go back to model view.
Escape
Wait+Screen /Applications/
Sleep 1s

# Switch to standalone relations view (Shift+R).
Type "R"
Wait+Screen /ENDPOINT/
Sleep 2s

# Go back to model view.
Escape
Wait+Screen /Applications/
Sleep 1s

# Switch to standalone applications view (Shift+A).
Type "A"
Wait+Screen /SCALE/
Sleep 2s

Type "q"

Sleep 500ms
