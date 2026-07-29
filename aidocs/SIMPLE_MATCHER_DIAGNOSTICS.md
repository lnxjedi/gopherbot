# SimpleMatcher Diagnostic Decisions

The user-facing DSL is documented in `devdocs/SimpleMatcher.md`; parser and
matcher source are authoritative for syntax.

Directed `SimpleMatcher` commands retain a compiled matcher object, not merely a
regex, so the engine can distinguish:

- exact executable match;
- exact command skeleton with one invalid captured value; and
- no match.

A syntax diagnostic must never start a pipeline. Exact matches always win; a
diagnostic is shown only when one visible command produces an unambiguous
candidate. Multiple candidates fall through to ordinary unmatched-command
handling.

This strict skeleton rule is deliberate: diagnostics are field validation, not
fuzzy command matching. Literal structure, separators, option position, and
extra-token checks must all agree.

Visibility is a security boundary. Do not reveal diagnostics for commands the
current canonical user cannot confidently discover. Authorization and
elevation remain engine-owned; uncertainty suppresses the hint.
