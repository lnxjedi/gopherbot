# Dev Container Decisions

The base container supplies the production/dev runtime and editor tooling; the
dev layer adds source repositories and builds Gopherbot.

Editor resolution for custom JavaScript depends on the workspace and
`jsconfig.json` assets under `resources/containers/assets/`, which map custom
`lib/` before installed `lib/`. Preserve that ordering so robot-local helpers
override the installed baseline during development.

Inspect containerfiles and assets for the current package/tool inventory.
