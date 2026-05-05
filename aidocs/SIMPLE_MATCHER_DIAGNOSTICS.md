# SimpleMatcher Diagnostics

This document describes the target engine design for `SimpleMatcher` command diagnostics. It is an implementation handoff for the v3 UX goal in root `GOALS_v3.md`: when a user supplies the right command shape but a captured field is invalid, the engine should explain that specific field instead of falling through to generic help.

## Scope

This design applies only to directed plugin `Commands` configured with `SimpleMatcher`.

Out of scope:
- raw `Regex` command diagnostics
- ambient `MessageMatchers`
- reply matchers
- job argument matchers
- connector-local command parsing

## Current Anchors

- Config loads matchers into `InputMatcher` in `bot/tasks.go`.
- `compileInputMatcher` and the parser/compiler live in `bot/simple_matcher.go`.
- Directed command dispatch loops over `plugin.Commands` in `bot/dispatch.go` method `checkPluginMatchersAndRun`.
- Fallback and wrong-location guidance are downstream of directed command matching in `bot/dispatch.go` method `handleMessage`.
- Human authoring contract: `devdocs/SimpleMatcher.md`.
- Pipeline flow summary: `aidocs/PIPELINE_LIFECYCLE.md`.

## First-Class Matcher Object

`SimpleMatcher` should be represented as a compiled object, not only as a compiled regex string.

At load time, the object should:
- parse the SimpleMatcher spec into an AST
- compile exact matching regex data for current behavior
- retain capture metadata: labels, types, choice values, and positional argument order
- retain enough command-skeleton metadata to distinguish `NoMatch` from `SyntaxMatch`

Regex-backed commands may keep the existing regex behavior. They only need `NoMatch` and `ExactMatch`.

The runtime contract should be shaped like:

```go
type MatchKind int

const (
    NoMatch MatchKind = iota
    SyntaxMatch
    ExactMatch
)

type MatchResult struct {
    Kind       MatchKind
    Args       []string
    Diagnostic string
}
```

Exact matches return positional args exactly as plugin handlers receive them today. Syntax matches return no executable args and must never start a pipeline.

## Command Skeleton Rule

False positives are controlled by an exact command-skeleton rule.

A `SyntaxMatch` is allowed only when:
- required literal and synonym terms match exactly, using normal SimpleMatcher case and separator forgiveness
- optional noise/capturing groups are either absent or present in a grammar-valid position
- all separators and token boundaries line up with the grammar
- there is one invalid captured value that explains the failure
- there are no extra unmatched terms before or after the command

There is no fuzzy matching in this path. If the skeleton is not exact, the matcher returns `NoMatch`, allowing the normal deterministic help fallback to make suggestions.

Example:

```yaml
SimpleMatcher: "set loglevel {to} (level:trace|debug|info|warn|error)"
```

Input:

```text
set loglevel to fine
```

Result:

```text
SyntaxMatch: Invalid value 'fine' for 'level'; valid values: trace, debug, info, warn, error.
```

Input:

```text
set logging to fine
```

Result:

```text
NoMatch
```

The second input should flow to normal help/fallback because the command skeleton did not match exactly.

## Labelled Choices

Capturing choices should require a label prefix:

```text
(label:a|b|c)
[label:a|b|c]
```

The label may be empty:

```text
(:foo:bar|baz|frotz)
```

The first top-level colon separates the label from the values. In the empty-label example, the valid values are `foo:bar`, `baz`, and `frotz`.

Unlabelled capturing choices such as `(trace|debug)` and `[disabled]` should be migrated to `(:trace|debug)` / `[:disabled]` or to a labelled form such as `(level:trace|debug)`.

This is a v3 config syntax migration. It does not change plugin argument order.

## Typed Capture Diagnostics

Typed captures already support labels:

```text
<siding:ident>
```

The matcher object should retain the label and type. If the skeleton matches and the user supplies an invalid value, the diagnostic should name the label and explain the type:

```text
Invalid value '9round' for 'siding'; expected an identifier starting with a letter, followed by letters, numbers, '_' or '-'.
```

Types should have short human descriptions alongside their regex patterns. These descriptions must be stable enough for tests.

## Dispatch Behavior

Directed command dispatch should evaluate visible command matchers and keep the best result:

1. Exact regex command match: current behavior.
2. Exact SimpleMatcher match: current behavior, including duplicate-match protection.
3. SyntaxMatch: retain the diagnostic candidate, but do not stop scanning for exact matches.
4. NoMatch: ignore.

After all available directed command matchers are checked:
- if one or more exact matches exist, run the existing exact-match path
- if no exact match exists and exactly one syntax diagnostic candidate exists, send that diagnostic and treat the message as handled
- if no exact match exists and multiple syntax candidates exist, do not guess; continue into the normal unmatched-command flow
- if no syntax candidate exists, continue into the normal unmatched-command flow

Syntax diagnostics should run before ambient `MessageMatchers`, job commands, wrong-location hints, catch-alls, and thread subscriptions only when the diagnostic candidate is unique and visible. Otherwise existing routing order should continue.

## Visibility And Security

Syntax diagnostics must not reveal command surfaces the user should not know about.

Candidate diagnostics should be considered only for commands visible to the user in the current routing context. At minimum this means:
- task-level availability passes for user and location
- command-level admin restrictions are respected
- command-level authorization visibility follows the same conservative rules used by help/wrong-location hints
- if visibility cannot be determined confidently, suppress the syntax diagnostic and let normal fallback continue

Authorization and elevation remain engine-owned. Connectors must not participate in matcher diagnostics.

## Connector And Routing Boundaries

Connectors remain responsible for transport, identity mapping, and message context. The engine remains responsible for command parsing, policy, and diagnostics.

Diagnostic replies should use the same routing convention as existing unmatched-command guidance:
- direct-message commands reply in the DM
- channel commands reply in-thread when a thread-capable context exists

No connector-specific diagnostic behavior is required.

## TDD Plan

Use a narrow test-first slice.

Unit tests in `bot/simple_matcher_test.go` should cover:
- labelled required choices parse and exact-match without changing positional args
- empty-label choices permit values containing `:`
- unlabelled capturing choices are rejected
- typed capture labels are retained for diagnostics
- exact match beats syntax diagnostics
- exact skeleton plus invalid choice returns `SyntaxMatch`
- exact skeleton plus invalid typed capture returns `SyntaxMatch`
- non-exact skeleton returns `NoMatch`
- optional groups do not create false positives when omitted
- multiple capture fields either diagnose one clear field or return `NoMatch`/ambiguous according to the implementation contract

Focused routing tests should cover:
- a unique syntax diagnostic is sent and no pipeline starts
- multiple syntax diagnostics do not guess
- an exact match elsewhere wins over a syntax diagnostic
- command-level visibility suppresses diagnostics for commands the user should not see
- skeleton mismatch falls through to existing help/catchall behavior

Integration coverage should be small: one representative directed command through the test connector is enough after unit coverage exercises the matcher object.

## Token-Conservative Implementation Order

1. Introduce matcher result types and a compiled SimpleMatcher object behind `InputMatcher`.
2. Keep the current exact regex output working before adding diagnostics.
3. Add labelled choice parsing and migrate shipped configs/tests from `(a|b)` to `(:a|b)` or `(label:a|b)`.
4. Add `Match` diagnostics for labelled choices.
5. Add typed capture diagnostics.
6. Wire unique `SyntaxMatch` handling into directed command dispatch.
7. Run focused unit tests first, then the smallest applicable integration slice.

This order keeps behavior-preserving refactors separate from user-visible diagnostics and makes failures easier to classify.
