## drako: this machine's command deck

The operator keeps curated commands in drako. Prefer them to improvising:
each cell is a command someone chose and described.

    drako ls                every equipped deck: addresses, names, descriptions
    drako explain <addr>    one cell in full, including the exact command
    drako run <addr>        run it; exits with the command's own exit code

Addresses come from `drako ls`: `B2` is a cell, `B2.1` is item 1 of the
folder at B2, and `work:B2` names the profile. Bare addresses mean the
active profile, which changes when the operator switches decks, so
qualify them.

- Explain before you run. If the command opens an editor or a full-screen
  app, or asks for input, it needs a human at the keyboard.
- Cells marked danger are always refused by `drako run`. Don't work around
  that: tell the operator which cell and why. They run it from the grid.
- Exit 125 with a `drako run:` line on stderr means drako refused or could
  not resolve the address. Any other exit code is the command's own.
