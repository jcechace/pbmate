# PBMate Hero Demo
# Run: vhs demo.tape
# Prerequisites: PBM cluster reachable via current context or --uri flag.

Output docs/tapes/demo.gif

Set Shell bash
Set Width 1200
Set Height 700
Set FontSize 14
Set Theme "Catppuccin Mocha"
Set Padding 10
Set TypingSpeed 60ms

# Launch PBMate.
Type "pbmate --theme mocha"
Enter
Sleep 4s

# Overview: cluster topology, agent health, live logs.
Sleep 2s

# Toggle log follow — watch entries stream in real time.
Type "f"
Sleep 4s

# Stop follow.
Type "f"
Sleep 1s

# Switch to Backups tab.
Type "2"
Sleep 2s

# Navigate the backup list — detail panel updates with each selection.
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Type "j"
Sleep 2s

# Open the full backup form — type, profile, compression, namespaces.
Type "S"
Sleep 2s

# Advance past the core selectors (Type, Profile, Compression).
Enter
Sleep 1s

# Type a namespace filter.
Set TypingSpeed 40ms
Type "mydb.orders, mydb.users"
Sleep 2s

# Let the viewer see the filled form.
Sleep 1s

# Dismiss form.
Set TypingSpeed 60ms
Escape
Sleep 1s

# Switch to Config tab — syntax-highlighted YAML, storage profiles.
Type "3"
Sleep 3s

# Open help overlay — all keybindings at a glance.
Type "?"
Sleep 3s

# Dismiss help.
Escape
Sleep 500ms

# Quit (double-press).
Type "q"
Sleep 500ms
Type "q"
Sleep 1s
