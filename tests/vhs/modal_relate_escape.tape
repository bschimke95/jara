# modal_relate_escape.tape — open the relate modal from model view and dismiss it with Esc.
# Verifies that Esc closes the modal and stays on the model view (does not navigate back).

Source tests/vhs/_setup.tape

Output tests/vhs/golden/modal_relate_escape.ascii

Hide
Type "./jara --demo"
Enter

# Wait for model detail view to render.
Wait+Screen /Applications/
Sleep 1s

# Open the relate modal with the 'r' key.
Type "r"

# Wait for the modal to appear.
Wait+Screen /Add Relation/
Sleep 1s

# Press Esc — should close the modal and stay on the model view.
Escape

# The model view should still be visible (not the models list).
# Wait for the Relations panel to confirm the full view rendered after modal close.
Wait+Screen /grafana-source/
Wait+Screen /Relations/
Show

Sleep 1s

Hide
Type "q"
Sleep 500ms
