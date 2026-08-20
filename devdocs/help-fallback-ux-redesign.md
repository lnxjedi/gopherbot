# Help And Fallback UX Redesign

## Executive Summary

This branch pivots away from AI-first unmatched-command recovery and toward a deterministic help and fallback system that is easier to trust, easier to maintain, and better aligned with the actual failure patterns we saw in production.

The key product decision is simple:

- unmatched alias-addressed commands should recover through built-in help and command discovery
- AI should not be a core dependency of basic command recovery
- users should be guided from broad browsing to exact command syntax in small, readable steps

This work adds a clearer help ladder, a plugin-level help view, exact command help using `help plugin/command`, smarter keyword help when the keyword is also a plugin name, higher-quality deterministic suggestions grounded in real command surfaces, and more actionable fallback copy. The result is a help system that does a better job of getting a user from "I need help finding the right command" to "I found the exact command and syntax" without needing AI inference.

## Why We Changed Direction

We analyzed real unmatched-command traffic captured from production robots. The dominant error patterns were not open-ended "figure out my intent" prompts. They were things like:

- mistyped command names
- singular/plural mismatches
- wrong verb variants such as `show`, `get`, `list`, `reload`, or `refresh`
- users knowing the rough command family but not the exact syntax
- users needing help in the moment, not a long dump of every command

That made the product conclusion pretty clear:

- deterministic recovery can likely solve most real-world alias fallback cases
- AI fallback adds complexity faster than it adds user value
- the main UX problem is discoverability and scanability, not lack of semantic intelligence

## Design Principles

The redesign uses a few simple UX rules that fit chatops well:

1. Progressive disclosure beats wall-of-text help.
2. Recovery paths should be shown at the moment of failure.
3. Help should move from broad to narrow:
   `commands` -> `help plugin` -> `help plugin/command`
4. Users should never have to guess the next step after a fallback response.
5. The system should prefer short, actionable replies over exhaustive replies.

## What Changed

### 1. Quick Help Now Teaches The Help System

`help` now explicitly teaches three levels of discovery:

- search help: `help <keyword>`
- plugin help: `help <plugin>`
- exact command help: `help <plugin>/<command>`

This gives users a more understandable mental model than the older quick-help copy.

### 2. `commands` Is An Apropos-Style Command Index

The old command browsing path rendered plugin blocks and command previews. The current output uses one logical line per available command:

```text
plugin/command: `addressed usage` - summary
```

The engine divides very large indexes into ordered messages at command-line boundaries and ends with the exact-help path.

### 3. `help <plugin>` Now Works As A Real Plugin Overview

Users often know the command family they want, but not the exact command. The new plugin help view solves that by showing:

- the plugin name
- the plugin summary
- the commands in that plugin using the same one-line address, usage, and summary format
- the next exact step: `help <plugin>/<command>`

This is the missing middle layer between global browsing and exact command syntax.

### 4. `help <plugin>/<command>` Now Gives Exact Command Help

This was a requested feature, and it fits the final design very well.

It provides a stable, explicit path for users who already know the plugin and want the exact command. It also gives the help system a clean target to point users toward from fallback responses.

### 5. Keyword Help Is Smarter When A Plugin Name Matches

When `help <keyword>` exactly matches a plugin name, the response now does two jobs:

- it shows the plugin overview first
- it then shows other command matches from other plugins for the same keyword

This matters for terms like `help`, `list`, or `groups`, where users may want both:

- the plugin named by the keyword
- other commands elsewhere that also match that keyword

This makes the help response feel guided rather than arbitrary.

### 6. Multi-Match Help Uses Apropos Summaries

One important product decision in the second design pass was that multi-command help output should always reinforce the canonical command address.

The system consistently renders a plaintext `plugin/command`, copyable connector-aware usage, and purpose on one line, with a blank line between command entries. Public commands use alias addressing. Private-capable commands prefer the connector's hidden-command form; a required-private command without that transport uses the alias form and marks the summary `(direct message only)`. Full help is reserved for a single result or an exact `help plugin/command` request.

That address now shows up across:

- `commands` plugin previews
- `help <plugin>` command lists
- keyword search results
- exact-help links in multi-match responses

The former `help <keyword> brief` mode is gone because every multi-command result is compact.

### 7. Exact Help Supports Man-Page-Like Details

Full help renders summary, usage, SimpleMatcher-derived options, optional multiline BasicMarkdown `Details`, examples, and availability. `Details` is additive metadata and is neither rendered in multi-command results nor used for ranking.

This is the text-chat equivalent of teaching a stable "man page address" for each command.

### 8. Deterministic Fallback Copy Is More Actionable

When a user sends an unmatched command, the built-in fallback now gives short next steps instead of only saying the command did not match.

Current recovery copy points users toward:

- `commands`
- `help <keyword>`
- `help <plugin>/<command>`

That makes the failure path more useful immediately.

### 9. High-Confidence Suggestions Are Now Explicit, But Narrower

When the matcher has a strong deterministic guess based on the actual command surface, the fallback now says so directly.

Example shape:

- `Best guess: [knock] knock?`

If the engine only has a weaker directional hint, it now uses softer wording such as "Closest help I found..." instead of pretending certainty.

The system still follows that with broader help guidance, so the response is not a brittle one-shot guess. The intended user experience is:

- suggest the most likely command when confidence is high
- soften the wording when the match is only directionally helpful
- still provide the broader help path if the suggestion is wrong

## Why This Is Better For Users

### Less Cognitive Load

Users do not need to read a giant command inventory up front. They can skim plugins first, then drill down only when needed.

### Better Recovery From Mistakes

The fallback response no longer stops at "no match." It tells the user exactly how to continue.

### Better Match For Real User Behavior

The production command-miss report suggested users usually know roughly what they want. They need orientation, not a chatbot therapist.

### Better Learnability

A repeat user can now learn a durable navigation pattern:

- browse with `commands`
- inspect with `help plugin`
- learn the command address from `plugin/command`
- execute with `help plugin/command`

That is much easier to internalize than relying on fuzzy search alone.

## Live UX Testing

I tested the updated help flows with a local UX robot and the integration harness.

The sample transcripts below are representative of the current behavior and can be reproduced with the local test configuration or an interactive robot using the same engine/config combination. The exact robot name varies by deployment, so the examples focus on the command flow rather than a specific bot identity.

### Transcript: `!help`

User:

```text
!help
```

Robot:

```text
Quick help:
!help <keyword> - get help for the provided <keyword>
!commands - browse commands available in this channel
!help-all - summarize all commands available in this channel, including global commands
Plugin help: !help <plugin>
Exact command help: !help <plugin>/<command>
Tip: !commands shows commands available in this channel.
```

Commentary:

- quick help now explicitly teaches plugin help and exact command help
- users are told that `commands` shows commands in the current channel

### Transcript: `!commands`

User:

```text
!commands
```

Robot:

```text
Commands available in this channel:

builtin-help/commands: `/acme-bot commands` - browse commands available in this channel

builtin-help/help: `/acme-bot help <keyword>` - find help for commands matching <keyword>

knock/knock: `!tell me a knock-knock joke` - start an interactive knock-knock joke

lists/add: `!add <item> to <name>` - add an item to a named list
...

For full command help, use `!help <plugin>/<command>`.
```

Commentary:

- every command has one logical address, usage, and summary line
- the footer gives the exact-help path
- large output is split only at command-line boundaries

This is noticeably more scannable than a flat command list.

### Transcript: `!help help`

User:

```text
!help help
```

Robot:

```text
Command matches for keyword: help

help/help: `!help with robot` - give general help on the help system and using the robot

builtin-help/help: `/acme-bot help <keyword>` - find help for commands matching <keyword>

groups/help: `!help with groups` - give general help for using groups
...

For full command help, use `!help <plugin>/<command>`.
```

Commentary:

- the plugin named by the keyword is shown first
- other matching commands use the same compact format

This is the new "plugin match plus broader search" behavior.

### Transcript: `!help knock`

User:

```text
!help knock
```

Robot:

```text
Command help: knock/knock

Summary: Starts an interactive knock-knock joke.
Usage: tell me a knock-knock joke
Examples:
- !tell me a knock-knock joke
Availability: #general, #random
```

Commentary:

- a search or plugin lookup with exactly one result goes directly to full help

This is exactly the kind of "middle step" the old UX was missing.

### Transcript: `!help knock/knock`

User:

```text
!help knock/knock
```

Robot:

```text
Command help: knock/knock

Summary: Starts an interactive knock-knock joke.
Usage: tell me a knock-knock joke
Examples:
- !tell me a knock-knock joke
- <bot>, tell me another joke
- <bot>, tell me a (knock-knock) joke
Availability: channels: (many)
```

Commentary:

- the robot returns exact command help for the fully qualified path
- the response includes summary, usage, generated options when applicable, optional details, examples, and availability

This makes exact help addressable and predictable.

### Transcript: `!tell me a jok`

User:

```text
!tell me a jok
```

Robot:

```text
I couldn't match `tell me a jok`.
Best guess: [knock] `knock`? Starts an interactive knock-knock joke.
Try `!help knock/knock` in #general or #random, or run `!tell me a knock-knock joke` there.
Possible commands to check:
- `knock/knock` - Starts an interactive knock-knock joke.
If not, try `!commands`, `!help <keyword>`, or `!help <plugin>/<command>`.
```

Commentary:

- deterministic fallback now makes a high-confidence suggestion when the intended phrase is actually close to a supported command surface
- the reply still includes broader help escape hatches and exact-help addresses
- the response can mention likely target channels when the command family is not valid where the user asked
- this is intentionally better than guessing from an internal command name alone

This is already a better failure experience than the earlier baseline.

### Transcript: bare identifier typo for a phrase command

User:

```text
!knok
```

Robot:

```text
I couldn't match `knok` here.
This looks more likely to belong in #general or #random.
Try `!help knock/knock` in #general or #random, or run `!tell me a knock-knock joke` there.
If not, try `!commands`, `!help <keyword>`, or `!help <plugin>/<command>`.
```

Commentary:

- this is an intentional behavior change from the earlier prototype
- the engine no longer over-claims that a one-word typo is a confident near-match for a phrase-shaped command
- instead it offers softer recovery guidance and points the user back to exact help

## Configuration And Metadata Impact

The good news is that the first phase does **not** require a major plugin help metadata redesign.

The current system can produce a much better UX by reusing the metadata we already have:

- command summary
- usage
- examples
- plugin/task description
- optional multiline command details

That means we can improve user experience significantly without forcing a broad migration of plugin YAML.

### Recommended Metadata Direction

For v3, I would keep the metadata shape mostly stable in the short term and tighten expectations instead of inventing a large new schema.

The most useful improvements are:

- make plugin summaries more consistent
- make command summaries short and action-oriented
- ensure examples are realistic and copy-paste-friendly
- keep usage lines short enough for chat

The additive `Details` field solves the specific need for man-page-like exact help without migrating existing metadata. Robots opt in command by command after upgrading.

## Product Recommendation

The current branch direction is the right one:

- remove AI-first unmatched-command recovery from the core product
- keep deterministic fallback built in
- use help metadata to improve navigation and recovery
- reserve AI for optional robot-owner customization only if a real long-tail use case appears later

In other words, the product should treat help and fallback as a navigation system, not an inference system.

## Recommended Next Slices

The current work establishes the right help architecture. The next best product slices would be:

1. Improve family-level matching.
   Handle common operator substitutions such as `show/get/list`, `reload/refresh/rebuild`, and singular/plural variants.

2. Tighten plugin summaries and examples.
   This is likely to deliver a lot of UX value with little engine complexity.

3. Tune fallback copy by situation.
   Differentiate between:
   - obvious near-match
   - wrong channel
   - broad no-match
   - command family likely found but syntax incomplete

4. Continue tuning exact-help details where complex commands need option semantics or operational notes.

## Bottom Line

This redesign moves the product in a stronger direction for v3.

It reduces system complexity, removes AI pressure from the core help path, and gives users a clearer way to discover commands and recover from mistakes. Most importantly, it is grounded in observed user behavior from real unmatched-command data rather than a hypothetical belief that AI must be involved.

The new interaction model is:

- `help` teaches the system
- `commands` shows one-line command summaries
- `help plugin` narrows to a family
- `help plugin/command` gives exact syntax
- fallback points users back into that path

That is a much better user journey for chatops.
