# _setup.tape — shared setup sourced by all VHS test tapes.
# The jara binary is already built by the Makefile before tapes run.

Require vhs

Set Width 1200
Set Height 800
Set FontSize 14
Set TypingSpeed 50ms

Hide
Type "cd $JARA_ROOT"
Enter
Type "clear"
Enter
Sleep 500ms
Show
