# Status

Gopherbot is in the middle of the v3 transition. The important point for users is that the current engine behavior is real and usable now, even though some long-tail features and connectors are still evolving.

The docs in this book were reorganized in March 2026 around the workflow the project actually supports today:

- develop and run the engine locally on a Linux workstation
- keep robot configuration in git
- prefer built-in interpreters and current v3 config structure
- deploy the same robot cleanly to long-running environments

## What is current

- The startup and configuration model documented here matches the current v3 engine defaults and robot skeleton.
- The SSH connector is the default local-development connector.
- Slack remains the primary production team-chat connector.
- Multi-protocol runtime support, `PrimaryProtocol`, `SecondaryProtocols`, and username-authoritative identity are part of the current model.
- `BasicMarkdown` is the default outgoing message format unless you set another one explicitly.

## What is still moving

- Some included plugins and older integrations are still catching up to v3 expectations.
- Connector coverage is still uneven across platforms; this manual focuses on what is supported and maintained today.
- The engine still evolves faster than the user manual at times, so the shipped defaults and example repositories remain useful companion references.

## How to read this book

If you are new to Gopherbot, start with Part I and build a local robot first.

If you already run older robots, jump to [Configuration Reference](Configuration.md) and [Upgrading Existing Robots](Upgrading.md).

If you are writing automation, Part III is the modern starting point after you are comfortable with the local workflow.
