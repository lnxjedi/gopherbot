# Pipeline Methods

Gopherbot pipelines are built in code rather than declared in a separate pipeline language. That is why the Robot API includes methods for adding tasks, jobs, commands, cleanup steps, and failure handlers while a pipeline is being assembled.

## Pipeline-building methods

- `GetParameter(name)`
- `SetParameter(name, value)`
- `AddTask(name, args...)`
- `FinalTask(name, args...)`
- `FailTask(name, args...)`
- `AddJob(name, args...)`
- `SpawnJob(name, args...)`
- `AddCommand(plugin, command)`
- `FinalCommand(plugin, command)`
- `FailCommand(plugin, command)`
- `Exclusive(tag, queueTask)`

## Important behavior notes

- `GetParameter(...)` reads explicit task/plugin/job and pipeline parameters
  first, then falls back to engine runtime metadata such as `GOPHER_USER`,
  `GOPHER_CHANNEL`, `GOPHER_PROTOCOL`, and `GOPHER_ENVIRONMENT`.
- `GOPHER_ENVIRONMENT=development` is the common v3 signal for dry-run or
  prove-it behavior in extensions that would otherwise change host state,
  external systems, or persistent robot memory.
- `AddCommand(...)` composes another plugin command into the current pipeline. It is not the same thing as injecting a fresh user message.
- `FinalTask(...)` runs during cleanup even if the main pipeline fails.
- `FailTask(...)` and `FailCommand(...)` only run when the primary pipeline fails.
- `SpawnJob(...)` starts work in parallel, while `AddJob(...)` adds a child job into the current pipeline flow.
- `Exclusive(...)` is the advisory lock for protecting shared resources such as worktrees or deployment targets.

## Privileged tasks

`Privileged: true` on a task means the task requires the current pipeline to already be privileged. `AddTask(...)`, `FinalTask(...)`, and `FailTask(...)` do not elevate an unprivileged pipeline; if unprivileged code tries to add a privileged task, the engine rejects the request.

Use this for tasks that need robot-owned credentials or tools that should not be exposed to ordinary plugin code. For example, a task that runs `kubectl` with a robot-managed kubeconfig should be declared privileged. Only a privileged job, a privileged plugin, or code already running inside a privileged pipeline can add that task:

```yaml
ExternalTasks:
  "kubectl-apply":
    Path: tasks/kubectl-apply.gsh
    Description: Apply Kubernetes manifests with the robot kubeconfig
    Privileged: true
```

```go
// This must run from a privileged job/plugin pipeline.
r.AddTask("kubectl-apply", "clusters/prod/payment-api.yaml")
```

For plugins and jobs, `Privileged: true` also controls whether that plugin or job starts a privileged pipeline. For tasks, it is a requirement on the owning pipeline.

`SetWorkingDirectory(...)` exists in the public Go API, but it is currently omitted from this user-facing reference until that area is revisited.

## Examples

### Go
```go
if !r.Exclusive("deploy-prod", false) {
    r.Say("Another production deploy is already running.")
    return robot.Normal
}

r.SetParameter("TARGET_ENV", "production")
r.AddTask("build-artifact", "payments")
r.AddTask("deploy-service", "payments")
r.FinalTask("cleanup-worktree")
r.FailTask("notify-deploy-failure")
```

### Lua
```lua
local gopherbot = require("gopherbot_v1")
local task = gopherbot.task
local Robot = gopherbot.Robot

local bot = Robot:new()
if not bot:Exclusive("deploy-prod", false) then
  bot:Say("Another production deploy is already running.")
  return task.Normal
end

bot:SetParameter("TARGET_ENV", "production")
bot:AddTask("build-artifact", "payments")
bot:AddTask("deploy-service", "payments")
bot:FinalTask("cleanup-worktree")
bot:FailTask("notify-deploy-failure")
```

### JavaScript
```javascript
const { Robot, task } = require("gopherbot_v1")();

const bot = new Robot();
if (!bot.Exclusive("deploy-prod", false)) {
  bot.Say("Another production deploy is already running.");
  return task.Normal;
}

bot.SetParameter("TARGET_ENV", "production");
bot.AddTask("build-artifact", "payments");
bot.AddTask("deploy-service", "payments");
bot.FinalTask("cleanup-worktree");
bot.FailTask("notify-deploy-failure");
```

### Bash
```bash
if ! Exclusive "deploy-prod" false
then
  Say "Another production deploy is already running."
  exit 0
fi

SetParameter "TARGET_ENV" "production"
AddTask "build-artifact" "payments"
AddTask "deploy-service" "payments"
FinalTask "cleanup-worktree"
FailTask "notify-deploy-failure"
```

### Python
```python
if not bot.Exclusive("deploy-prod", False):
    bot.Say("Another production deploy is already running.")
else:
    bot.SetParameter("TARGET_ENV", "production")
    bot.AddTask("build-artifact", ["payments"])
    bot.AddTask("deploy-service", ["payments"])
    bot.FinalTask("cleanup-worktree", [])
    bot.FailTask("notify-deploy-failure", [])
```

### Ruby
```ruby
if !bot.Exclusive("deploy-prod", false)
  bot.Say("Another production deploy is already running.")
else
  bot.SetParameter("TARGET_ENV", "production")
  bot.AddTask("build-artifact", ["payments"])
  bot.AddTask("deploy-service", ["payments"])
  bot.FinalTask("cleanup-worktree", [])
  bot.FailTask("notify-deploy-failure", [])
end
```
