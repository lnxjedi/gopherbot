# Unresolved AI Follow-ups

- Add plain-language suite and case purposes to process-backed integration YAML
  so failures communicate intent without source archaeology.
- Require every job to have a real non-DM operator channel.
- Reproduce secondary connector activation failure on reload and make failed
  activation visible without sacrificing secondary isolation.
- Make thread-subscription TTL configurable while preserving the current
  seven-day default and protocol-scoped keys.
- Audit compiled Go privilege configuration so trusted/in-process status cannot
  be mistaken for configurable unprivileged execution.
- Perform pre-v3 pilot migrations across real robots: environment selection,
  secret migration, connector identity/reload, and hidden commands.
- Define a safer operator-guided stale brain-lock recovery for shutdowns killed
  before release; do not infer ownership from age alone.
- Add hardened systemd/Terraform guidance for root-owned launch secrets and
  parent-process memory protection.
- Reconcile SSH and terminal slash parsing while preserving the distinction
  between transport-hidden and robot-addressed input.
- Clarify or narrow runtime bot-ID lookup so it is not misused as a generic
  connector self-identity heuristic.

Completed work does not belong here; Git history and tests record it.
