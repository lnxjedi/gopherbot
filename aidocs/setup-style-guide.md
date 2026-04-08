# Setup Style Guide

This guide defines the preferred conversational style for setup-oriented flows such as `new-robot`, `setup-slack`, and future guided onboarding helpers.

## Goals

- make setup feel calm, friendly, and readable
- reduce repeated user-mention noise in setup conversations
- keep prompts predictable so users understand what is happening next
- prefer explanatory progress over checkpoint-style "type `done` to continue" handoffs

## Messaging Pattern

The default rhythm for setup questions is:

1. explain what value is about to be requested and why it matters
2. pause long enough for the user to read
3. prompt for the value
4. confirm the accepted value in plain language
5. move on to the next topic

Preferred shape:

```text
robot: Your robot will need a 'given name' something like "Floyd", but unique to *your* custom robot. For maximum compatibility and portability across chat platforms, the robot will recognize messages that start with its name as a command it should try to interpret, for example "Floyd, ping".
robot: What name would you like to give your robot?
user: Clu
robot: Perfect. Your robot will respond to messages that start with "Clu".
```

## Addressing And Targeting

- greet the user once when they connect or reconnect
- do not keep repeating `@username` in every follow-up setup message
- for channel-based setup flows, prefer normal channel messages after the initial greeting
- when a reply must be bound to a specific user in a shared channel, use a targeted prompt API such as `PromptUserChannelForReply(...)`; the prompt machinery already scopes the reply to the intended user

## Paragraphing

- send one message per paragraph, not one message per sentence
- keep descriptive paragraphs compact but complete
- if a concept needs two distinct ideas, split them into two separate sends with a read pause between them

## Pauses

- use a short initial pause before the first greeting so the robot feels like a fast typist
- use a noticeably longer pause between descriptive paragraphs the user is expected to read
- keep these pauses tunable through named constants in shared setup libraries rather than scattered numeric literals

Current shared onboarding constants live in `lib/newrobotflow/onboarding.go`:

- `SetupInitialGreetingPauseSeconds`
- `SetupParagraphReadPauseSeconds`
- `SetupRestartTransitionPauseSeconds`

## Confirmation Style

- after accepting an answer, confirm it explicitly
- prefer confirmations that restate the operational meaning of the value, not just "ok" or "saved"
- avoid repeating the same confirmation opener in every step; vary the phrasing naturally while keeping the operational meaning clear
- examples:
  - "Got it. Scheduled job messages will go to `#clu-jobs`."
  - "Perfect. I'll configure bootstrap to use `git@github.com:...`."

## Manual Step Handoffs

- do not require the user to type `done` merely to acknowledge that they have read instructions
- if the next robot action does not actually depend on immediate confirmation, explain the expected manual steps and continue
- reserve explicit confirmation prompts for cases where the robot truly needs new information before proceeding

Repository/setup example:

- good: explain how to create the empty upstream repository, add the deploy key, and push `custom/`, then restart if the local scaffold is already sufficient for runtime
- avoid: "reply `done` once the push succeeds" when the robot can safely continue without that message

## Source Material

When rewriting or creating setup prompts:

- follow the robot's tone from built-in help output
- reuse descriptive content from setup answer files under `resources/answerfiles/` when it remains accurate
- prefer wording that explains purpose and expected outcome over implementation jargon
