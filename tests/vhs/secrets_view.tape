# secrets_view.tape — navigate to secrets view and drill down to detail.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/secrets_view.ascii

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

Wait+Screen /Revisions/

Sleep 2s

Type "q"
