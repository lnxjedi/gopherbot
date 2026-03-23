# Help And Fallback UX Redesign

## Executive Summary

This branch pivots away from AI-first unmatched-command recovery and toward a deterministic help and fallback system that is easier to trust, easier to maintain, and better aligned with the actual failure patterns we saw in production.

The key product decision is simple:

- unmatched alias-addressed commands should recover through built-in help and command discovery
- AI should not be a core dependency of basic command recovery
- users should be guided from broad browsing to exact command syntax in small, readable steps

This work adds a clearer help ladder, a plugin-level help view, exact command help using `help plugin/command`, smarter keyword help when the keyword is also a plugin name, high-confidence deterministic suggestions, and more actionable fallback copy. The result is a help system that does a better job of getting a user from "I need help finding the right command" to "I found the exact command and syntax" without needing AI inference.

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

### 2. `commands` Is Now Organized Around Plugin Groups

The old command browsing path leaned toward a very long command list. That is hard to skim in chat.

The new `commands` output is grouped by plugin and, for each plugin, shows:

- a short plugin summary
- a preview of representative commands using their full `plugin/command` address
- the next step: `help <plugin>`

This turns `commands` into a directory instead of a dump.

### 3. `help <plugin>` Now Works As A Real Plugin Overview

Users often know the command family they want, but not the exact command. The new plugin help view solves that by showing:

- the plugin name
- the plugin summary
- the commands in that plugin with short summaries
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

### 6. Multi-Match Help Teaches The Canonical Command Address

One important product decision in the second design pass was that multi-command help output should always reinforce the canonical command address.

The system now consistently teaches users to think in terms of:

- `plugin/command`

That address now shows up across:

- `commands` plugin previews
- `help <plugin>` command lists
- keyword search results
- exact-help links in multi-match responses

This is the text-chat equivalent of teaching a stable "man page address" for each command.

### 7. Deterministic Fallback Copy Is More Actionable

When a user sends an unmatched command, the built-in fallback now gives short next steps instead of only saying the command did not match.

Current recovery copy points users toward:

- `commands`
- `help <keyword>`
- `help <plugin>/<command>`

That makes the failure path more useful immediately.

### 8. High-Confidence Suggestions Are Now Explicit

When the matcher has a strong deterministic guess, the fallback now says so directly.

Example shape:

- `Did you mean [knock] knock?`

The system still follows that with broader help guidance, so the response is not a brittle one-shot guess. The intended user experience is:

- suggest the most likely command when confidence is high
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

## Live UX Testing With A Local Test Robot

I rebuilt and restarted a local UX test robot and tested the updated help flows over SSH using a separate test user.

At the time of writing, the throwaway UX test robot is:

- robot dir: `/tmp/clu-ux`
- SSH port: `4223`

If it is still running when you review this, you should be able to connect with your normal SSH testing flow against port `4223` and reproduce these transcripts directly.

The sample transcripts below are copied from live test sessions and are written just as the user would see them in chat. The exact robot name varies by deployment, so the examples focus on the command flow rather than a specific bot identity.

### Transcript: `!help`

User:

```text
!help
```

Robot:

```text
Quick help:
!help <keyword> - get help for the provided <keyword>
!help <keyword> brief - compact help for a likely command
!commands - browse command groups available in this channel
!help-all - help for all commands available in this channel, including global commands
Plugin help: !help <plugin>
Exact command help: !help <plugin>/<command>
Tip: !commands shows command groups in this channel.
```

Commentary:

- quick help now explicitly teaches plugin help and exact command help
- users are told that `commands` shows command groups in the current channel

### Transcript: `!commands`

User:

```text
!commands
```

Robot:

```text
Command groups available in this channel:
Skim a plugin first, then drill into one command if you need exact syntax.
- builtin-help: A plugin providing help for commands
  Commands: builtin-help/commands, builtin-help/help, builtin-help/help-all, builtin-help/info
  Try: !help builtin-help
- knock: An external Go plugin demonstrating Prompting and custom configuration
  Commands: knock/knock
  Try: !help knock
- lists: An external Go plugin for managing simple lists of strings
  Commands: lists/add, lists/delete, lists/empty, lists/help, +5 more
  Try: !help lists
...
For one command: !help <plugin>/<command>
Search by keyword: !help <plugin|command|keyword>
```

Commentary:

- response begins with "Command groups available in this channel"
- each plugin shows a summary, a command preview using the full `plugin/command` address, and a `!help <plugin>` next step
- the footer gives both `!help <plugin>/<command>` and keyword search

This is noticeably more scannable than a flat command list.

### Transcript: `!help help`

User:

```text
!help help
```

Robot:

```text
Help for keyword: help

Plugin help: help
give general help on the help system and using the robot
Commands:
- `help/help` - give general help on the help system and using the robot
Exact help: !help help/help

Other command matches:

Command: builtin-help/help
Summary: find help for commands matching <keyword>
Usage: help <keyword>
Examples:
- !help ping
- !help
Availability: all channels + direct messages
Exact help: !help builtin-help/help

Command: groups/help
...
Exact help: !help groups/help
```

Commentary:

- the plugin named by the keyword is shown first
- other matching commands from other plugins are still shown afterward
- every secondary result teaches the exact `plugin/command` address

This is the new "plugin match plus broader search" behavior.

### Transcript: `!help knock`

User:

```text
!help knock
```

Robot:

```text
Help for keyword: knock

Plugin help: knock
An external Go plugin demonstrating Prompting and custom configuration
Commands:
- `knock/knock` - Starts an interactive knock-knock joke.
Exact help: !help knock/knock
```

Commentary:

- the robot now returns a plugin overview for `knock`
- the response lists the plugin summary and the `knock/knock` command address
- it points the user to `!help knock/knock`

This is exactly the kind of "middle step" the old UX was missing.

### Transcript: `!help knock/knock`

User:

```text
!help knock/knock
```

Robot:

```text
Command help: knock/knock

[knock] knock
Usage: tell me a knock-knock joke
Summary: Starts an interactive knock-knock joke.
Examples:
- !tell me a knock-knock joke
- <bot>, tell me another joke
- <bot>, tell me a (knock-knock) joke
Availability: channels: (many)
```

Commentary:

- the robot returns exact command help for the fully qualified path
- the response includes usage, summary, examples, and availability

This makes exact help addressable and predictable.

### Transcript: `!knok`

User:

```text
!knok
```

Robot:

```text
I couldn't match `knok`. Did you mean [knock] `knock`? Starts an interactive knock-knock joke. Try `!help knock/knock` in #botdev, #chat, #clu-jobs, or other configured channels, or run `!tell me a knock-knock joke` there. If not, try `!commands`, `!help <keyword>`, or `!help <plugin>/<command>`.
```

Commentary:

- deterministic fallback now makes a high-confidence suggestion when the intended command is obvious
- the reply still includes broader help escape hatches
- the response can mention likely target channels when the command family is not valid where the user asked

This is already a better failure experience than the earlier baseline.

## Configuration And Metadata Impact

The good news is that the first phase does **not** require a major plugin help metadata redesign.

The current system can produce a much better UX by reusing the metadata we already have:

- command summary
- usage
- examples
- plugin/task description

That means we can improve user experience significantly without forcing a broad migration of plugin YAML.

### Recommended Metadata Direction

For v3, I would keep the metadata shape mostly stable in the short term and tighten expectations instead of inventing a large new schema.

The most useful improvements are:

- make plugin summaries more consistent
- make command summaries short and action-oriented
- ensure examples are realistic and copy-paste-friendly
- keep usage lines short enough for chat

If we later decide we need richer metadata, we should add it to solve a specific UX problem, not because we assume "more structure" will help.

## Product Recommendation

The current branch direction is the right one:

- remove AI-first unmatched-command recovery from the core product
- keep deterministic fallback built in
- use help metadata to improve navigation and recovery
- reserve AI for optional robot-owner customization only if a real long-tail use case appears later

In other words, the product should treat help and fallback as a navigation system, not an inference system.

## Recommended Next Slices

The current work establishes the right help architecture. The next best product slices would be:

1. Tune the new high-confidence suggestion layer.
   The first pass is in place now, but it should be tuned so obvious near-matches are recovered aggressively without creating noisy bad guesses.

2. Improve family-level matching.
   Handle common operator substitutions such as `show/get/list`, `reload/refresh/rebuild`, and singular/plural variants.

3. Tighten plugin summaries and examples.
   This is likely to deliver a lot of UX value with little engine complexity.

4. Tune fallback copy by situation.
   Differentiate between:
   - obvious near-match
   - wrong channel
   - broad no-match
   - command family likely found but syntax incomplete

5. Consider a compact mode for long plugin overviews.
   Some plugins may still need shorter chat-first rendering when command counts are large.

## Bottom Line

This redesign moves the product in a stronger direction for v3.

It reduces system complexity, removes AI pressure from the core help path, and gives users a clearer way to discover commands and recover from mistakes. Most importantly, it is grounded in observed user behavior from real unmatched-command data rather than a hypothetical belief that AI must be involved.

The new interaction model is:

- `help` teaches the system
- `commands` shows command groups
- `help plugin` narrows to a family
- `help plugin/command` gives exact syntax
- fallback points users back into that path

That is a much better user journey for chatops.
