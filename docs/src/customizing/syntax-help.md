# Writing Good Help Text

Help in v3 is command-linked, so good help starts with good command metadata.

Most directed commands should use `SimpleMatcher`; see the [SimpleMatcher reference](../reference/SimpleMatcher.md) for the full command grammar. Help text should show what users type, not every detail of the matcher grammar.

## Recommended fields

- `Usage`: the command body only
- `Summary`: one short sentence
- `Examples`: a few realistic examples
- `Keywords`: the search terms users will actually try

## Example

```yaml
Commands:
- Command: deploy
  SimpleMatcher: "deploy <branch:token>"
  Usage: "deploy <branch>"
  Summary: "deploy the named git branch to the selected environment"
  Examples:
  - "(alias) deploy main"
  - "(alias) deploy release/2026-03-13"
  Keywords: [ "deploy", "release", "ship" ]
```

## Good habits

- Use placeholders in examples, not your real bot name.
- Keep `Usage` short and command-line-like.
- Put the detailed explanation in follow-up help or normal prose, not inside `Usage`.
- If a command has a common invalid form, handle it deliberately and reply with the correct syntax.
