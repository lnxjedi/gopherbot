# Upgrading Existing Robots

For v3, extension compatibility is a higher priority than configuration compatibility.

That means many older plugins and jobs can keep working, while robot configuration often needs deliberate migration.

Until the first public v3 release, this is a priority rather than a guarantee:
extension APIs and operational behavior may still change when required to
complete the v3 model. Intentional breaks should fail clearly and be documented
in the root `UPGRADING-v3.md` migration guide and the Changelog.

## Highest-priority migration items

1. Move to `PrimaryProtocol` and optional `SecondaryProtocols`.
2. Put connector-local identity mapping inside each connector's `ProtocolConfig`.
3. Keep `UserRoster` as the canonical global user directory.
4. Move provider-specific settings into `conf/brains/<provider>.yaml` and `conf/history/<provider>.yaml`.
5. Expect `DefaultMessageFormat` to be `BasicMarkdown` unless you set something else explicitly.

## Identity changes to care about

- authorization is username-based
- `IgnoreUnlistedUsers` is username-based
- Slack `UserMap` is connector-local
- SSH user mapping belongs in `ProtocolConfig.UserKeys`

## Practical upgrade advice

The safest migration path is usually:

1. scaffold a fresh v3-style robot
2. compare it with your existing robot
3. port your custom config and extensions into the new layout
4. test locally before touching production

That approach usually goes faster than trying to rescue a heavily mutated old config tree in place.
