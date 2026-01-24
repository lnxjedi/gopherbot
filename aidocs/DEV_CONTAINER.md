# Dev Container Notes

These notes capture how the dev IDE container is built and how editor tooling
discovers the Gopherbot libraries.

## Container images

- `resources/containers/containerfile.base` builds the base image used for dev
  and production robots. It installs OpenVSCode Server, Go, mdBook, and language
  tooling (gopls, staticcheck, Ruby/Lua/Python extensions), then exposes port
  `7777`.
- `resources/containers/containerfile.dev` layers on top of the base image. It
  copies the repos into `/opt/gopherbot` and `/opt/gopherbot-doc`, installs
  shell helpers, and runs `make` in `/opt/gopherbot`.

## IDE workspace + JS completion

The dev container copies:

- `resources/containers/assets/gopherbot.code-workspace` → `/home/bot/gopherbot.code-workspace`
  (opens `/home/bot`, `/opt/gopherbot`, `/opt/gopherbot-doc`).
- `resources/containers/assets/jsconfig.json` → `/home/bot/jsconfig.json`.

`jsconfig.json` sets:

- `baseUrl: "./custom"`
- `paths: ["./lib/*", "/opt/gopherbot/lib/*"]`

This is what enables VSCode to resolve Gopherbot JS helpers under `lib/`
for code completion when editing custom scripts.
