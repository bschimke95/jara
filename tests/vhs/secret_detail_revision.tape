Source tests/vhs/_setup.tape

Output tests/vhs/golden/secret_detail_revision.ascii

# Launch jara in demo mode.
Hide
Type "./jara --demo"
Enter
Sleep 2s
Show

# Navigate to secrets via command mode.
Type ":"
Sleep 300ms
Type "secrets"
Enter
Sleep 1s

# Drill into the first secret.
Enter
Sleep 1s

# We should see the detail view with "Revision: 3" (newest first) in the left pane.
Wait+Screen /Revision: 3/

# Take a visual snapshot before moving.
Sleep 500ms

# Move cursor down to next revision.
Type "j"
Sleep 1s

# The left pane should now show "Revision: 2".
Wait+Screen /Revision: 2/
Sleep 500ms

# Move cursor down again.
Type "j"
Sleep 1s

Wait+Screen /Revision: 1/
Sleep 500ms
