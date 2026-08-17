# Built-in Commands

Every robot ships with a small set of foundational commands, and many robots also keep some stock plugins enabled.

## Core commands most users will see

- `ping`: check that the robot is listening
- `help`, `commands`, `help-all`: discover commands
- `info`: inspect runtime and admin-facing basics
- `whoami`: see how the robot identifies you

## Commonly enabled stock plugins

- `links`: store and search shared bookmarks
- `lists`: keep simple shared lists
- onboarding helpers on the default robot, such as `new robot`

## Why these matter

These commands are more than demos:

- `info` helps confirm which robot, branch, and environment you are talking to
- `whoami` helps debug identity mapping and `UserRoster` issues
- `links` and `lists` are small but real examples of persistent memory usage
