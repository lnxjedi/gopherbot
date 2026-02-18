This directory contains default configuration for Gopherbot, and defines a **default robot**, *Floyd*. Your robot's custom configuration overrides configuration here.

Layout overview:
- `robot.yaml`: global robot policy and runtime selection (protocols, brain provider, history provider, etc.)
- `protocols/<protocol>.yaml`: connector-local protocol configuration (`ProtocolConfig`)
- `brains/<provider>.yaml`: brain-provider-specific settings (`BrainConfig`)
- `history/<provider>.yaml`: history-provider-specific settings (`HistoryConfig`)
