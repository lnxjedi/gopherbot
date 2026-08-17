# Primary Pipelines

The primary stage is where the real work happens.

## Key behaviors

- tasks run in sequence
- a failure stops normal primary-stage execution
- plugins often begin as a single task and then add more work dynamically
- privileged tasks can only be added by a pipeline that is already privileged

## Important API calls

- `AddTask`
- `AddJob`
- `AddCommand`
- `SetParameter`
- `SetWorkingDirectory`

## v3 nuance

- `AddJob` starts a child pipeline
- `AddCommand` stays inside the current pipeline and is not a fake new inbound chat message
- tasks added during execution can run before later originally queued tasks, which is how setup tasks can expand into more detailed work on the fly

## Privilege and AddTask

Marking a task with `Privileged: true` does not make the pipeline privileged. It means the task requires a privileged pipeline. This is useful for reusable operations that depend on robot-owned credentials, such as a `kubectl` task that uses a robot-managed kubeconfig. A privileged job or plugin can add that task; an unprivileged plugin cannot.
