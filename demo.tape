# VHS tape — render with: vhs demo.tape  (produces demo.gif)
# Supply ETHERSCAN_API_KEY in the environment that runs vhs; never type it here.

Output demo.gif

Set Shell "bash"
Set FontSize 17
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms
Set Padding 22

# Help
Type "nodrain --help"
Sleep 600ms
Enter
Sleep 2s

# Interactive scan of a well-known wallet (vitalik.eth)
Type "nodrain scan 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
Sleep 600ms
Enter
Sleep 5s

# Navigate and select a couple of approvals
Down
Sleep 400ms
Space
Sleep 400ms
Down
Sleep 400ms
Space
Sleep 600ms

# Review the revoke plan
Enter
Sleep 3s

# Back out and quit
Escape
Sleep 600ms
Type "q"
Sleep 1s

# Scriptable JSON output
Type "nodrain scan --json 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045 | jq '.[0]'"
Sleep 600ms
Enter
Sleep 3s
