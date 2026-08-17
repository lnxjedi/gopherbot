# CLI Tools for Authors

The `gopherbot` binary is also a CLI utility. The most useful commands for authors and operators are:

- `gopherbot --help`
- `gopherbot check`
- `gopherbot match`
- `gopherbot syntax`
- `gopherbot script`
- `gopherbot encrypt`
- `gopherbot decrypt`
- `gopherbot validate <path>`
- `gopherbot dump installed <path>`
- `gopherbot dump configured <path>`
- `gopherbot list`
- `gopherbot fetch <key>`
- `gopherbot version`

## Examples

Encrypt a secret:

```bash
gopherbot encrypt MyLousyPassword
```

Validate a robot repository:

```bash
gopherbot validate ~/robots/acme/custom
```

Check one SimpleMatcher without loading a robot:

```bash
gopherbot check 'spot (type:rails|devops) up [<branch:token>]' spot-devops-up
```

Check one or more script files without loading a robot:

```bash
gopherbot syntax custom/plugins/hello.lua
gopherbot syntax custom/plugins/hello.lua custom/plugins/report.js
```

`syntax` supports Lua, JavaScript, Gopherbot shell (`.gsh`), and interpreted Go
(`.go`) files. The language is normally inferred from the file extension. Use
`-language lua|js|gsh|go` when checking an unusual path, and `-json` when you
want structured diagnostics for editor tooling or automation.

Run a plugin script locally with a fixture-backed Robot API:

```bash
gopherbot script custom/plugins/hello.lua -- hello alice
```

In plugin mode, the first argument after `--` is the plugin command and the
remaining arguments are the captured command arguments. For job or task scripts,
pass `-kind job` or `-kind task`; then the arguments after `--` are passed
directly to the script:

```bash
gopherbot script -kind task custom/tasks/render.lua -- daily-report
```

`script` starts only the built-in interpreter child process. It does not load
robot configuration, start connectors, connect to the real brain, or run
pipelines. The local Robot API uses a fixture for message context, task config,
parameters, prompts, attributes, and local memory.

When `-fixture` is omitted, `script` uses the installed
`conf/default-fixture.yaml`. Copy it before editing local test data:

```bash
gopherbot script -new-fixture hello-fixture.yaml
gopherbot script -fixture hello-fixture.yaml custom/plugins/hello.lua -- hello alice
```

The default fixture sets `GOPHER_ENVIRONMENT=development`, so plugins can use
`GetParameter("GOPHER_ENVIRONMENT")` or the process environment to enter a
dry-run or prove-it mode. Fixture `parameters:` are visible through
`GetParameter(...)` and as child-process environment variables, and can override
runtime metadata for local checks.

Inline scripts are useful for quick API checks:

```bash
gopherbot script -language lua -c 'local gopherbot = require("gopherbot_v1"); local bot = gopherbot.Robot:new(); bot:Say(bot:GetParameter("GOPHER_ENVIRONMENT")); return gopherbot.task.Normal' -- check
```

Match command text against the configured robot's plugin commands:

```bash
gopherbot match spot-devops-up
```

`match` uses the configured robot's command metadata. If the robot encryption key is available, template secrets are decrypted normally; if it is not available, secret template values are replaced with redacted placeholders and a note is written to stderr.

For repeated tests, load the command metadata once and prompt for command text until EOF:

```bash
gopherbot match -interactive
```

Dump the final merged robot config:

```bash
gopherbot dump configured robot.yaml
```

Secret template values are redacted by default when dumping expanded config. Use `--unredacted-secrets` only when you deliberately want decrypted secret values printed:

```bash
gopherbot dump --unredacted-secrets configured plugins/example.yaml
```

These commands are especially useful when configuration reload fails and you want to understand what the engine thinks the world looks like.
