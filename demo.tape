# fedit demo - generates demo.gif
# Regenerate with: vhs demo.tape

Output demo.gif

Set FontSize 16
Set Width 1100
Set Height 650
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set Padding 20

# Show structure of a Go file
Type "fedit -file demo_sample.go -op map -lang go"
Sleep 500ms
Enter
Sleep 2500ms

# Find a specific line by content
Type "fedit -file demo_sample.go -op find -match 'usersHandler'"
Sleep 500ms
Enter
Sleep 2s

# Surgical insert - content-matched, no line numbers
Type "fedit -file demo_sample.go -op insertafter -match 'api/orders' -text '// TODO: add products endpoint here' -v"
Sleep 500ms
Enter
Sleep 3s

# Global rename - one operation, all occurrences
Type "fedit -file demo_sample.go -op replaceall -match 'Handler' -text 'Endpoint' -v"
Sleep 500ms
Enter
Sleep 3s

Sleep 1s