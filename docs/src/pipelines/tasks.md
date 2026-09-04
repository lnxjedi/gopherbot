# Included Tasks

Gopherbot ships with a small set of useful stock tasks. Common examples include:

- `send-message`
- `notify-admins`
- `restart-robot`
- `robot-quit`
- `pause`
- `pause-brain`
- `resume-brain`
- `rotate-log`
- `tail-log`
- `ssh-agent`
- `ssh-git-helper`
- `git-command`
- `email-log`

These tasks are useful both directly and as examples of how the pipeline model is meant to be used.

## `notify-admins`

Usage: `AddTask notify-admins <message> (...)`

Sends the message as a direct message to every canonical username in
`AdminUsers`. The task attempts every administrator and fails if any delivery
fails. It is privileged in the installed defaults, so only privileged
pipelines can add it unless custom `GoTasks` configuration explicitly
overrides that setting.
