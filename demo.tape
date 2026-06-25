# VHS tape — render with: vhs demo.tape  (produces demo.gif)
# Needs ETHERSCAN_API_KEY in the environment that runs vhs (never typed here).

Output demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 680
Set Theme "Catppuccin Mocha"
Set TypingSpeed 25ms
Set Padding 20

# Scan a wallet for active token approvals (vitalik.eth on Arbitrum)
Type "nodrain scan --chain arbitrum 0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
Sleep 500ms
Enter
Sleep 5s

# Navigate the table and tick approvals to revoke
Down
Sleep 700ms
Space
Sleep 600ms
Up
Sleep 500ms
Space
Sleep 800ms

# Review the revoke plan (revoke.cash link + unsigned calldata)
Enter
Sleep 2500ms

# Write the unsigned revoke bundle to a file
Type "w"
Sleep 1800ms

# Quit
Type "q"
Sleep 800ms
