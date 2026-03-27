# Help Sample Transcripts

This document is a transcript-driven UX spec for the v3 help system and deterministic unmatched-command fallback.

In product/UX terms, this is closest to a conversation design spec or transcript matrix:
- conversation design spec, because it defines the wording and interaction flow the user should experience
- transcript matrix, because it groups target replies by scenario instead of by implementation detail

The goal is to agree on the user-facing conversations before tuning ranking, matching, and rendering code.

## Principles

- Prefer one strong recovery path over a noisy list of weak guesses.
- Always show commands in their canonical `plugin/command` form when listing multiple possibilities.
- When there is a good direct retry, show the literal retry the user should type.
- Mention the current channel when channel availability matters.
- Do not leak internal matcher structure into replies.
- Keep replies short enough to scan in chat.
- Use `BasicMarkdown` formatting so the same messages work across connectors.

## Canonical Help Journey

This is the desired path for a user who starts broad and then drills down to exact help.

### Journey 1: Starting from `help`

User:
```text
!help
```

Robot:
```text
Quick help:
- `!commands` shows plugins and command groups available here
- `!help <keyword>` searches by plugin, command, or topic
- `!help <plugin>` shows commands from one plugin
- `!help <plugin>/<command>` shows exact help for one command
```

Notes:
- Keep this compact.
- Teach the four main wayfinding commands immediately.

### Journey 2: Browsing plugins, then drilling into one plugin

User:
```text
!commands
```

Robot:
```text
Plugins and command groups available in this channel:

**weather**: A plugin that uses OpenWeatherMap to report the weather
Commands: `weather/weather`
Help: `!help weather`

**lists**: An external Go plugin for managing simple lists of strings
Commands: `lists/add`, `lists/delete`, `lists/empty`, `lists/help`, `lists/list`, `lists/remove`
Help: `!help lists`

Exact help: `!help <plugin>/<command>`
Search by keyword: `!help <plugin|command|keyword>`
```

Notes:
- Plugin names are first-class.
- Command previews always use `plugin/command`.

Reviewer Notes:
- I like the spacing/separation here between plugins; that wasn't present in the last QA test I did.

### Journey 3: Keyword search that is also a plugin name

User:
```text
!help lists
```

Robot:
```text
Help for keyword: `lists`

Plugin help: `lists`
An external Go plugin for managing simple lists of strings
Commands:
- `lists/add` - Adds an item to a named list, prompting to create the list if needed.
- `lists/remove` - Removes an item from a named list.
- `lists/list` - Shows items from a named list.
- `lists/delete` - Deletes an entire named list.

Other command matches:
- `builtin-help/commands` - browse plugins and command groups available in this channel

Exact help: `!help <plugin>/<command>`
```

Notes:
- When the keyword is also a plugin name, show the plugin section first.
- Then show other matches, if any.

### Journey 4: Exact command help

User:
```text
!help weather/weather
```

Robot:
```text
Command help: `weather/weather`

Usage: `weather in <city>`
Summary: Fetches current weather from OpenWeatherMap.
Examples:
- `!weather in Boston`
- `!weather in London`
Availability: `#general`, `#weather`, `#ops`
```

Notes:
- Exact help is the destination state.
- This should show all relevant help for that command.
- Show explicit channel availability here when the list is still readable.
- Reserve shortened availability summaries like `(many)` for broader browse/search views.

Reviewer Notes:
- Is "[weather] weather" really needed? Eliminate that? Also, when getting help for a single plugin/command, list all the channels where it's available - the shortened "(many)" was intended only for long lists of > 2 matching commands where long channel lists would only further bloat the output.

## Mismatched Command Replies

Each scenario below describes the intended reply shape for deterministic fallback.

### Scenario 1: No strong guess

When to use:
- the unmatched text does not point clearly at one command
- ranked candidates are weak or noisy

User:
```text
!say something
```

Robot:
```text
I couldn't match `say something` in channel `#general`.
Try `!commands`, `!help <keyword>`, or `!help <plugin>/<command>`.
```

Notes:
- Mention the current channel.
- Do not fabricate a guess.
- The implementation may internally try a strong first-word help lookup here.
- Only surface a concrete help suggestion such as `!help say` if that lookup produces a genuinely useful result.

Reviewer Notes:
- The engine should try the first word as a help term internally, but should not speculate in the copy with a tentative `!help say?`.
- If first-word help is actually useful, the reply should recommend it directly.
- This is a good example of where the new SimpleMatcher syntax could have / should have helped - the deterministic code should have been able to quickly find that the first word, "say", was an exact match for the first word in the SimpleMatcher for "say anything"; note that if that plugin doesn't use the SimpleMatcher, we should correct that, in gopherbot or Clu
- Question: where, exactly, is the best place to take advantage of the SimpleMatcher syntax? `help <keyword>`? Mismatches? Both?

### Scenario 2: One strong near-match in the current channel

When to use:
- one command is a clear match
- the command is available in the current channel

User:
```text
!tell me a jok
```

Robot:
```text
`!tell me a jok` looks close to `knock/knock`.
Try: `!tell me a knock-knock joke`
More help: `!help knock/knock`
```

Notes:
- Use plain language, not “best guess.”
- Show one concrete retry and one exact-help path.

### Scenario 3: One strong near-match, but not available here

When to use:
- one command is a clear match
- the command is not available in the current channel

User:
```text
!tell me a jok
```

Robot:
```text
`!tell me a jok` looks close to `knock/knock`.
Try: `!tell me a knock-knock joke`
That command isn't available in channel `#general`; try `#botdev`, `#chat`, `#clu-jobs`, or another configured channel.
More help: `!help knock/knock`
```

Notes:
- Do not bury the availability note in one long sentence.
- Use channel guidance only when it is useful and specific.

### Scenario 4: Shared verb or overloaded command family

When to use:
- the first word or core phrase matches several command families
- the user likely needs disambiguation, not typo correction

User:
```text
!add bananas
```

Robot:
```text
`add` matches several commands.
You may be looking for:
- `lists/add` - `!add bananas to the groceries list`
- `groups/add` - `!add alice to the deployers group`
- `links/add` - `!add golang https://go.dev`
- `links/save` - save a link, then add keywords interactively
More help: `!help add`
Exact help: `!help <plugin>/<command>`
```

Notes:
- Group around the shared verb.
- Prefer one practical example per command family.

### Scenario 5: Phrase-shaped command with one plausible retry

When to use:
- the unmatched text is close to one phrase-shaped command
- there is one practical retry worth suggesting

User:
```text
!say something
```

Robot:
```text
I couldn't match `say something` in channel `#general`.
Did you mean `rubytest/sayit`?
Try: `!say anything`
More help: `!help rubytest/sayit`
```

Notes:
- This should come from ordinary near-match logic, not session-state special casing.
- Phrase-shaped retries should lean on `SimpleMatcher`, `Usage`, and `Examples`.

Reviewer Notes:
- Agreed: do not design special-case fallback around recent exact-help viewing.
- Keep this scenario only as a normal phrase-shaped near-match case.

### Scenario 6: Wrong channel, clear command match

When to use:
- the text clearly matches one command
- the problem is channel availability, not command recognition

User:
```text
!deploy staging
```

Robot:
```text
`deploy/staging` isn't available in channel `#general`.
Try `#deployments`.
More help: `!help deploy/staging`
```

Notes:
- This is a location problem, not an unmatched-command problem.
- Keep it short and direct.

### Scenario 7: Hidden command on a connector that does not support hidden commands

When to use:
- the command is hidden-only
- the active protocol does not support hidden commands

User:
```text
/clu admin-only
```

Robot:
```text
This command isn't supported over `some-connector`; that connector doesn't support hidden commands. Check with the robot administrator.
```

Notes:
- This comes from hidden-command policy, but it belongs in the overall help/recovery experience.

### Scenario 8: Hidden command supported, but addressed incorrectly

When to use:
- the active protocol supports hidden commands
- the command must be addressed with connector-specific hidden syntax

User:
```text
ping
```

Robot:
```text
Use `/(bot) <command>` to address a hidden command.
```

Notes:
- Connector owns the exact hidden-command syntax.
- This should be a single message.

## Copy Guardrails

These are wording constraints the implementation should follow.

- Do not say “best guess.”
- Do not say “possible commands to check.”
- Do not include obviously irrelevant low-confidence commands just to fill space.
- Do not expose internal-looking labels like `[knock] knock` outside exact help views. (Reviewer note: why would we ever want this over knock/knock?)
- Prefer:
  - “looks close to”
  - “you may be looking for”
  - “more help”
  - “exact help”
- Mention `channel #name` when the current location matters.
- Use backticks for commands, plugin names, channels, and placeholders.

## Implementation Implications

This document implies several ranking/rendering behaviors:

- fallback should choose among reply shapes, not just print ranked candidates
- shared-verb cases should be grouped and rendered as families
- channel availability guidance should only appear when it materially helps
- deterministic fallback should prefer examples, usage, and `SimpleMatcher` surfaces over raw regex text
- `help <keyword>` and mismatched-command recovery should share the same ranking surface built from:
  - `SimpleMatcher`
  - `Usage`
  - `Examples`
  - command name and plugin name
- first-word help lookups are a valid internal ranking technique, but should only be shown to the user when they produce a clearly useful suggestion

This document is intentionally transcript-first. Code changes should be judged by how closely they produce these conversations.
