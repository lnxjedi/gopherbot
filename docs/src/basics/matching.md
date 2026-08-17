# Command Matching

Gopherbot commands are matched by regular expressions, not fuzzy AI intent matching. That is a strength: the behavior is predictable, testable, and safe for automation.

Most normal directed commands are authored with `SimpleMatcher`, a small command grammar for words, choices, typed captures, and CLI-style dash options. Plugin authors should use the [SimpleMatcher reference](../reference/SimpleMatcher.md) for the full syntax and argument-passing rules.

## What happens when a command does not match

When you address the robot directly but no command matches, the robot usually replies with a message that points you back to help.

That failure can mean one of three things:

- you mistyped the command
- the command exists but is not available in the current channel or DM context
- the command is not visible to you because of policy or authorization

## v3 quality-of-life behavior

If a command is clearly valid somewhere else, the engine can often tell you that it belongs in another channel or in DM. That happens before generic catch-all behavior, so users get a more useful answer than a plain "no command matched".

## Best next step

Use `help <keyword>` instead of guessing. The help system understands command metadata and current visibility better than trial and error.
