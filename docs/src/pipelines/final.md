# Final Pipelines

Final tasks always run after the primary stage, whether the primary stage succeeded or failed.

## Ordering

Final tasks run in reverse order of addition. Think of them as a cleanup stack.

That behavior is intentional because it lets you pair setup and teardown naturally:

- start an `ssh-agent`
- later stop that same agent in a final task

## Good use cases

- cleanup
- closing sessions
- deleting temporary state
- sending a final summary after work is complete
