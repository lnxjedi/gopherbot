# Updating and Reloading

There are three related admin actions you will use a lot:

- `reload`
- `update`
- `switch-branch <branch>` or `default-branch`

## `reload`

Use `reload` when you have already changed files on disk and want the running robot to re-read configuration.

## `update`

Use `update` when the running robot should `git pull` its custom repository and then reload.

This is the common production flow:

1. make and test changes locally
2. commit and push them
3. tell the production robot to `update`

## Branch-aware workflows

`switch-branch <branch>` and `default-branch` are useful when you want a fast test-and-rollback cycle on a live robot without manually logging into the host and editing its checkout.

## Good habit

Prefer testing significant changes on a local robot first. `update` is great for safe rollouts, not for discovering syntax mistakes in front of your whole team.
