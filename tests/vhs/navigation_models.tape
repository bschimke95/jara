# navigation_models.tape — navigate to controllers via command bar, then into models.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/navigation_models.ascii

Type "./jara --demo"
Enter

# Wait for the model detail view to render.
Wait+Screen /Applications/

Sleep 1s

# Open command bar and navigate to controllers.
Type ":"
Sleep 300ms
Type "controllers"
Enter

# Wait for the controllers table.
Wait+Screen /prod-aws/

Sleep 1s

# Select prod-aws to open its models list.
Enter

# Wait for the models list.
Wait+Screen /staging/

Sleep 2s

# Go back to controllers.
Escape

Wait+Screen /prod-aws/

Sleep 1s

Type "q"

Sleep 500ms
