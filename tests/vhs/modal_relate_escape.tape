# modal_relate_escape.tape — open the relate modal from model view and dismiss it with Esc.
# Verifies that Esc closes the modal and stays on the model view (does not navigate back).

Source tests/vhs/_setup.tape

Output tests/vhs/golden/modal_relate_escape.ascii

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
Wait+Screen /Applications/

Sleep 1s

Type "q"

Sleep 500ms
