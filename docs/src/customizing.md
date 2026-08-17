# Extending Your Robot

Gopherbot is most useful when your team adds its own automation. In v3, the recommended extension path is:

1. start with the built-in interpreters or Go
2. keep extension config in `custom/conf/`
3. keep extension code in `custom/plugins/`, `custom/jobs/`, or `custom/tasks/`
4. test locally and reload often

## Choose the right extension type

- **plugin**: a user-facing command or ambient matcher
- **job**: scheduled or triggered automation that assembles work
- **task**: a reusable unit of work inside pipelines

## Language guidance

The current project preference is:

1. Lua for approachable built-in scripting
2. Go for safety and performance
3. JavaScript where it fits well

Bash, Python, and Ruby are still supported, especially when they are the most practical fit for an existing integration.
