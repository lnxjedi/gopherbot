This directory contains the installed default configuration for Gopherbot, and
defines a **default robot**, *Floyd*. Custom robot configuration under
`custom/conf/` overrides these installed defaults. This installed-default plus
custom-override layering is a core repository pattern, not a Google Chat
special case.

Layout overview:
- `robot.yaml`: global robot policy and runtime selection (protocols, brain provider, history provider, etc.)
- `protocols/<protocol>.yaml`: installed connector defaults (`ProtocolConfig`) that custom robots override from `custom/conf/protocols/`
- `*.yaml.sample`: inert owner/setup templates that are not loaded until copied or renamed to an active `*.yaml` path
- `brains/<provider>.yaml`: brain-provider-specific settings (`BrainConfig`)
- `history/<provider>.yaml`: history-provider-specific settings (`HistoryConfig`)
