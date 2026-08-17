# Addressing the Robot

Every robot has both:

- a regular name, such as `floyd`
- a one-character alias, such as `;`

Both are valid ways to address a command.

## Common patterns

```text
;ping
floyd, ping
floyd: ping
ping, floyd
```

`ping` is the canonical first command because it is usually available everywhere the robot is present.

## A useful routing nuance

If you send what looks like a command but forget to address the robot, many connectors support the follow-up pattern of sending just the bot name or alias as your next message. That tells the robot to interpret the previous message as if it had been addressed.

## Hidden commands

Some connectors support hidden messages, such as Slack slash commands or SSH hidden messages. Hidden messages are not a free bypass:

- the command still has to be explicitly allowed as hidden by the plugin
- the message still has to be treated as addressed to the robot
