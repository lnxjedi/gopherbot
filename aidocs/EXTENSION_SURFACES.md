# Extension Boundaries

- `robot/` defines contracts; modular packages depend on it, never on `bot/`.
- Connectors, brain/history/queue providers, and compiled extensions register
  statically. The engine consumes registrations and owns lifecycle/policy.
- Compiled Go extensions are trusted in-process code. File-backed extensions
  are child-process code selected by configured path and extension.
- A provider-scoped handler exposes only that provider's configuration.
  `ReadEncryptedFile` is for engine-owned providers/connectors, not a generic
  extension secret API.
- Credentialed shipped extensions remain explicit custom-robot opt-ins.

Build quirk: `modules.go` has a test build constraint but production builds
explicitly name it in the Makefile. This avoids double registration during
package tests. Do not remove it based solely on its build tag.

Use source registration calls and config loading code for the current inventory.
