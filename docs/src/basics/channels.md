# Channels, DMs, and Threads

Availability is a first-class part of Gopherbot command design.

## Channels

Many commands are intentionally limited to specific channels. That is normal for operational robots. For example:

- build or deploy commands may only belong in a job channel
- support commands may only belong in a support channel
- noisy or playful commands may be kept out of primary work channels

## Direct messages

Some commands are better in DM, especially when they:

- prompt for follow-up input
- reveal sensitive data
- would create too much channel noise

## Threads

Threads matter in Gopherbot. The engine tracks thread context and many plugins use it to keep long workflows contained. On connectors that support threads well, replying in-thread is usually the cleanest way to run multi-step operations.

## Practical takeaway

If a command works somewhere else but not here, it is often a channel-placement issue rather than a typo. Use `help <keyword>` or `commands` in the current context to see what is actually available.
