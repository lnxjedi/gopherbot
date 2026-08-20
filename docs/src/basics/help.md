# Finding Commands with Help

The built-in help plugin is the fastest way to discover what a robot can do in your current context.

## Most useful commands

- `help`
- `help <keyword>`
- `commands`
- `help-all`
- `info`

## What makes v3 help better

Help in v3 is tied more closely to command metadata:

- `Usage`
- `Summary`
- optional multiline `Details`
- `Examples`
- `Keywords`

That means users see help that is closer to the actual command surface rather than a detached block of prose.

When more than one command matches, help shows one line per command:

```text
lists/list: `;list <name>` - list the contents of the <name> list
```

The `plugin/command` address is plain text, and a blank line separates each entry. The usage includes the robot's configured alias so it can be copied directly. Commands allowed or required in private use instead show the connector's private-command form when available. If a required-private command has no private-command transport, its alias form is marked `(direct message only)`.

Use `help <plugin>/<command>` for full help. Full help shows the summary, usage, any options derived from `SimpleMatcher`, optional detailed documentation, examples, and availability. A search that finds only one command goes directly to that full view.

## Context-aware behavior

Help is filtered by the current channel or DM context, and where possible by what the user should actually be allowed to see. If a command exists somewhere else, help can point you there instead of pretending the command does not exist.

## Advice for users

- Use `commands` when you want to browse.
- Use `help <keyword>` when you know roughly what you want.
- Use `help <plugin>/<command>` when you want full help for one command.
- Use `info` when you need operational details about the robot itself.
