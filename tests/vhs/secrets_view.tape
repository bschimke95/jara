# secrets_view.tape — navigate to secrets view and drill down to detail.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/secrets_view.ascii

Hide
Type "./jara --demo"
Enter

# Wait for model detail view.
Wait+Screen /Applications/

Sleep 1s

# Navigate to secrets via Shift+S.
Type "S"

Wait+Screen /db-password/

Sleep 1s

# Drill down into the first secret.
Enter

# Wait for content unique to the secret detail view.
Wait+Screen /Revision: 3/
Show

Sleep 2s

Hide
Type "q"
Sleep 500ms
