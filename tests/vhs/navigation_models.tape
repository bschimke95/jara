# navigation_models.tape — navigate to controllers via command bar, then into models.

Source tests/vhs/_setup.tape

Output tests/vhs/golden/navigation_models.ascii

Hide
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

# Wait for staging-gce which is unique to the controllers table (not in any breadcrumb).
Wait+Screen /staging-gce/
Show

Sleep 1s

Hide
Type "q"
Sleep 500ms
