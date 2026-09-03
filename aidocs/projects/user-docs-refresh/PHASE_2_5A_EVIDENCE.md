# Phase 2.5A New-Robot Evidence Census

Date: 2026-09-01

This report establishes the evidence boundary for the supported new-Robot
path. It makes no source, scaffold, configuration, or onboarding behavior
change. Phase 2.5B turns these findings into the required Impact Surface Report
and an owner-approval proposal.

## Decision lens

A public launcher environment variable is justified when an operator may
reasonably need to override the value for one invocation and Gopherbot has no
equivalent CLI flag. For example:

```sh
GOPHER_ENVIRONMENT=production gopherbot pull-brain --force
```

That command must select the production environment's provider configuration
and variables without editing the Robot's normal local environment selection.
Durable Robot identity, behavior, provider settings, and secrets do not pass
this test merely because the current templates accept an environment variable.

The current `.env` design is the baseline, not a problem to replace. It is an
owner-readable bootstrap file for values needed before custom configuration
can be located or decrypted, plus the selected environment. A directly passed
launcher value overrides the corresponding `.env` value.

## Scope and method

The census covered:

- startup environment capture, private-file loading, scrubbing, restart
  handoff, startup-mode detection, configuration loading, CLI initialization,
  and extension-child environment construction;
- installed `conf/` defaults, `robot.skel/`, `new-robot`, `resume-setup`,
  `welcome-join`, the compiled bootstrap job, git and SSH helper tasks, and
  focused tests;
- the current configuration, secrets, execution-security, startup, testing,
  and compatibility decisions; and
- all seven locally available Robot configuration trees.

The instance comparison was read-only. Repositories were assigned temporary
anonymous identifiers during analysis. The findings below contain counts and
neutral operational patterns only; they contain neither non-public Robot
identities nor secret values.

## Principal findings

1. `.env` already has the correct narrow center of gravity. The generated file
   contains `GOPHER_ENCRYPTION_KEY` and `GOPHER_ENVIRONMENT`, then gains
   `GOPHER_CUSTOM_REPOSITORY` and `GOPHER_DEPLOY_KEY` for remote bootstrap. It
   is written with mode `0600`; startup tightens it to `0400`.
2. `GOPHER_ENVIRONMENT` is both a bootstrap value and a legitimate
   per-invocation operator selector. It affects environment-specific variables,
   the encrypted data-key file, provider configuration, and CLI commands that
   initialize Robot configuration, including brain maintenance commands.
3. Startup currently accepts every directly supplied `GOPHER_*` name, carries
   it across self-exec through a private file-descriptor handoff, and then
   scrubs it from the real process environment. The security handoff is sound,
   but acceptance is broader than the intended public launcher contract.
4. Installed defaults still express a large amount of durable Robot behavior
   through `env` template calls. The skeleton is partly modernized: it has
   environment files and a variables/secrets file, but Robot identity is still
   substituted directly into `robot.yaml`, while SSH listen address and port
   still use launcher environment variables.
5. The current onboarding flow does not generate a general-purpose outbound
   Robot SSH identity. It generates three distinct credentials: an SSH
   connector server host key, the initial human administrator's public login
   key, and a git deploy key that the instructions tell the owner to grant
   read-only access, used before the custom repository can be cloned.
6. The real-Robot evidence confirms a migration in progress, not a single
   settled pattern. Five of seven trees obtain Robot identity through named
   variables; two retain environment-based identity. Every tree has common
   variables/secrets plus two environment config files, but none has adopted
   environment-specific variables files yet.
7. No real-Robot config tree contains an active `{{ decrypt ... }}` template.
   Remaining matches in active configuration are descriptive words, not calls
   to the removed helper. Several imported user-doc pages still teach the
   removed helper and remain scheduled for rewrite or removal.
8. Four instance trees retain passphrase-backed outbound SSH automation. That
   is evidence for an optional production automation capability, not for
   putting a Robot private key back into the core new-Robot scaffold.
9. The onboarding state machine and friendly copy are tightly interleaved in
   one large Go file. The proposed English message catalog is therefore useful
   for owner voice editing, but moving text must not move validation, security,
   defaults, state transitions, or message targeting out of code.
10. The `.env` contract is sound, but the current two-stage writer conflicts
    with startup permissions: onboarding writes `.env` as `0600`, startup
    changes it to `0400`, and resumed onboarding later calls `os.WriteFile` on
    that same file to add repository credentials without first making it
    writable. The repository-handoff path therefore needs a focused fix and a
    process-backed test; this is not a reason to broaden `.env`.

## Current authority and precedence

The observed startup sequence is:

1. capture directly passed `GOPHER_*` values;
2. self-exec with those values in a private file-descriptor payload and remove
   them from the ordinary process environment;
3. load `private/environment` for legacy test fixtures or `.env` for normal
   operation, without overwriting values passed by the launcher;
4. select `GOPHER_ENVIRONMENT`, defaulting to `production`;
5. initialize encryption, preferring
   `custom/binary-encrypted-key.<environment>` and falling back to the shared
   production key file where allowed;
6. load `conf/variables/common.yaml`, then
   `conf/variables/<environment>.yaml` as an override;
7. expand installed defaults followed by the custom delta; and
8. construct a scoped environment/parameter set for each extension child.

The engine-created `GOPHER_*` context passed to extensions is not launcher
configuration. It includes paths, message identity and privacy context,
pipeline/task names, branch state, queue context, and final/failure metadata.
Those values remain engine-owned even where historical documentation calls
them environment variables.

## Launcher environment classification

This is the Phase 2.5A evidence classification. Phase 2.5B must propose the
exact supported list and migration behavior for owner approval.

### Strongly justified public launcher values

| Value | Reason it exists before or outside custom configuration |
|---|---|
| `GOPHER_ENVIRONMENT` | Selects one environment for normal starts and one-off CLI operations; there is no general CLI environment option. |
| `GOPHER_ENCRYPTION_KEY` | Outer key required before encrypted Robot data and named secrets can be read. |
| `GOPHER_CUSTOM_REPOSITORY` | Locates custom configuration when `custom/` does not exist. |
| `GOPHER_DEPLOY_KEY` | Authenticates the initial private-repository clone before a Robot config or parameter set exists. |

These four values are exactly the generated `.env` set after repository
handoff. `GOPHER_ENVIRONMENT` is also appropriate as a direct one-command
override.

### Operator overrides that need an explicit 2.5B decision

| Value | Evidence and conflict |
|---|---|
| `GOPHER_CUSTOM_BRANCH` | A useful initial-clone branch selector with no CLI equivalent, but the engine later overwrites the same name with observed runtime branch state. Input and output authority are conflated. |
| `GOPHER_PROTOCOL` | A plausible one-run connector override for demos, recovery, or diagnostics, but durable connector choice belongs in the selected environment. |
| `GOPHER_HTTP_DEBUG` | A genuine one-run diagnostic switch with no CLI flag, but it enables unsanitized HTTP logging and forces debug/file logging. Its security warning must remain prominent if retained. |
| `GOPHER_SSH_HOST` | A possible host-level bind override with no CLI flag; normal listen configuration belongs in the selected environment or protocol config. |
| `GOPHER_PORT` | A possible host-level override for the local extension API listener; normal configuration belongs in Robot config. |

The likely resolution is either a deliberately small documented escape-hatch
set or equivalent CLI flags. Accidental historical availability is not enough
to make a value public.

### Environment inputs that should cease being durable Robot configuration

The installed defaults and real Robot trees still use environment calls for
some or all of these concerns:

- Robot name, full name, email, alias, job channel, time zone, and message
  format;
- brain and history providers, regions, cache/state/history/workspace
  directories, and provider-specific options;
- primary/default protocol and connector settings;
- integration credentials and feature policy; and
- `GOPHER_LOGLEVEL`, `GOPHER_LOGDEST`, and `GOPHER_SSH_PORT`, which already
  have top-level `-level`, `-log`, and `-ssh-port` flags.

Stable nonsecret identity and behavior belong under `Variables`; encrypted
values belong under `Secrets`; environment-specific differences belong in
`variables/<environment>.yaml` or the selected environment config. Extension
configuration must reach the extension through an explicitly attached
namespace or parameter set, not broad process inheritance.

### Engine-owned context, not user configuration

The following families are constructed by the engine or its execution
boundary and must not be presented as launcher knobs:

- `GOPHER_HOME`, `GOPHER_CONFIGDIR`, `GOPHER_INSTALLDIR`, and
  `GOPHER_WORKSPACE`;
- `GOPHER_USER*`, `GOPHER_CHANNEL*`, `GOPHER_MESSAGE_ID`,
  `GOPHER_THREAD*`, `GOPHER_PROTOCOL`, and command/privacy context when they
  describe the triggering message;
- `GOPHER_START_*`, pipeline/job/task/final/failure/queue fields, and caller
  identifiers;
- runtime git branch observations and default-branch status; and
- private self-exec and privilege-separation handoff values.

One spelling can currently serve more than one authority—for example,
`GOPHER_PROTOCOL` can be a launcher input and later message context. Phase 2.5B
must preserve the semantic distinction even if compatibility temporarily
preserves the spelling.

## Skeleton and generated configuration

The scaffold is intentionally delta-only in several important respects:

- installed plugin, job, brain, and history defaults remain authoritative;
- development and production environment files select startup providers;
- connector configuration is kept under `conf/protocols/`;
- the SSH server private host key is encrypted in `Secrets`; and
- optional integrations remain samples rather than copied active config.

The remaining reconciliation work is concrete:

1. Put Robot name, full name, email, alias, and other stable Robot-owned values
   in `conf/variables/common.yaml`, and reference them from `robot.yaml` and
   connector files.
2. Add `conf/variables/development.yaml` and
   `conf/variables/production.yaml` when environment-specific values are
   needed, and teach the scaffold/readme the difference between environment
   config and environment variables files.
3. Stop using `GOPHER_SSH_PORT` in the skeleton because `-ssh-port` already
   supplies the supported launch override. Decide the bind-host treatment in
   Phase 2.5B.
4. Replace the installed `welcome-join` and `resume-setup` trigger's
   `GOPHER_BOTNAME` lookup with authoritative configured Robot identity.
5. Keep custom config delta-only; do not copy installed provider or extension
   inventories into a new Robot.

## Credential roles and legacy SSH evidence

| Credential | Current purpose | Core new-Robot disposition |
|---|---|---|
| Outer encryption key | Unlocks the Robot's encrypted data key and named secrets | Keep outside git in `.env` or deployment secret injection. |
| SSH connector host key | Proves the SSH server's identity to connecting users | Keep; generate during onboarding and store encrypted under `Secrets`. |
| Human SSH public key | Authorizes the initial canonical administrator on the SSH connector | Keep; public material belongs in connector config. |
| Git deploy key | Allows clone/pull before or through the managed update pipeline | Keep as narrow bootstrap credential; private key remains in owner-readable `.env`, public half goes to the repository host with read-only access requested by the setup instructions. |
| General outbound Robot keypair and passphrase | Optional automation identity used by some production pipelines | Do not generate in the core scaffold; retain the capability as an explicit production-automation setup. |

The `ssh-admin` plugin and the general `ssh-agent start` path still implement
the optional outbound identity. The `ssh-agent deploy` path is separately tied
to `GOPHER_DEPLOY_KEY`. Documentation and configuration must not blur those
two uses.

## Onboarding flow and state findings

The current nominal flow is:

1. start in the default SSH Robot and invoke `new robot`;
2. create or supply the outer encryption key, write `.env`, and restart;
3. resume on the SSH connector's join event for the initiating canonical
   username;
4. collect Robot identity, channels, contact information, canonical username,
   and a human SSH public key;
5. copy `robot.skel/`, create the SSH server host key, and append initial
   authorization and roster config;
6. collect the upstream repository URL, generate a deploy key, and update
   `.env`;
7. tell the user how to create/push the repository, mark the session complete,
   and restart; and
8. on the next join, show clean-directory bootstrap verification instructions
   and clear that user's session.

State is stored atomically in a mode-`0600` `.setup-state` JSON file. Version 4
stores multiple sessions keyed by canonical username and contains progress,
contact/config answers, public keys, and repository URL, but no encryption key
or private SSH key. An exclusive tag serializes updates.

The reconciliation risks are:

- version is written but not validated or migrated explicitly on load;
- three obsolete stage names remain as compatibility branches without a
  defined pre-v3 removal decision;
- only a small subset of helper and state behaviors has unit coverage; there
  is no process-backed complete-wizard, interruption, restart, or bootstrap
  test;
- the first restart makes `.env` mode `0400`, while the resumed repository
  handoff attempts to rewrite it directly; normal unprivileged operation
  cannot rely on that succeeding;
- after the encryption-key prompt, onboarding removes the existing `custom/`
  tree before restart; the supported precondition and recovery behavior must
  be explicit before retaining that destructive step;
- an existing scaffold causes the flow to continue to repository handoff
  without proving that its contents match the collected answers;
- repository handoff is marked complete before the upstream push or a clean
  bootstrap is verified; and
- user-visible messages, pauses, prompt behavior, and transition logic are
  interleaved, making safe voice editing unnecessarily difficult.

## Message-catalog boundary for Phase 2.5B

The evidence supports a locale-ready English catalog with semantic keys, such
as `prompt_for_robot_name`, rather than stage-number keys. Catalog entries may
own:

- friendly user-facing paragraphs and prompt text;
- explicit named placeholders;
- fixed-width command blocks;
- ordered paragraph/list structure; and
- named pacing cues selected from code-defined pause classes.

Code must continue to own validation regexes, retry counts, defaults,
canonicalization, privacy-aware targeting, state transitions, persistence,
credential generation, filesystem changes, and security decisions. Catalog
loading must fail clearly for a missing key, unknown placeholder, or invalid
message shape. BasicMarkdown remains the portable default format.

## Real-Robot comparison

Across the seven anonymous trees:

- all seven use common variables/secrets and have two environment config
  files;
- zero have environment-specific variables files;
- five put Robot identity behind named variables and two still depend on
  environment-based identity;
- five `.env` files explicitly select an environment, while two older files
  contain only the other three bootstrap values;
- all seven use `GOPHER_ENVIRONMENT` during config selection;
- environment calls remain common for directories, providers, listener
  settings, logging, and older identity fields;
- no active removed decrypt template remains; and
- four retain optional outbound SSH passphrase configuration for operational
  automation.

These are migration samples, not defaults to copy. The strongest production
pattern is named identity variables plus explicitly scoped secrets. The
strongest remaining gap is that common and per-environment data have not yet
been separated into the two-layer variables-file model.

## Validation and evidence gaps

Passing during this census:

- `go test ./bot ./jobs/go-resume-setup ./jobs/go-welcome-join`
- `GOWORK=off go test ./...` from `lib/`

The first attempted root-module glob for `./lib/newrobotflow` failed because
`lib/` is intentionally a separate Go module; rerunning it through the
documented module-local command passed.

Phase 2.5B should now produce the Impact Surface Report and proposed contract.
No implementation should begin until the owner approves the launcher-variable
exceptions, destructive/recovery behavior, credential boundary, state
migration policy, repository verification boundary, and catalog shape.
