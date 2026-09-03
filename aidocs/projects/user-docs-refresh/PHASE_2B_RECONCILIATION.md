# Phase 2B Corpus Reconciliation

Status: approved by the owner on 2026-08-31. No broad content moves or
deletions have been made.

## Outcome

The imported corpus does not expose a missing top-level user journey. Keep the
approved Phase 2A navigation spine. The corpus mainly reveals content gaps,
stale assertions, duplicated wrapper pages, and reference material that needs
consolidation; those findings do not justify changing the north-star structure.

The current corpus contains 128 Markdown sources including <code>SUMMARY.md</code>:

- 70 content destinations are linked from the current navigation;
- 23 pages are already under `Outdated/`;
- 34 non-outdated content pages are not linked from the navigation; and
- <code>SUMMARY.md</code> is the remaining source file.

Every source is classified below. `Keep` means retain the page's principal
purpose through a focused accuracy pass, not preserve every current sentence.
`Rewrite` means retain a recognizable destination but rebuild it from current
source truth. `Merge` means extract useful material into the named destination
and retire the current page. `Remove` means the page has no active user-manual
destination after the pre-cleanup tag.

The resulting disposition is 16 keep, 35 rewrite, 42 merge, and 35 remove.

## Proposed final navigation and source tree

This is the approved north-star table of contents expressed as concrete target
paths. Titles may receive ordinary editorial refinements during bounded
concern work, but structural changes require a user need or operational concern
that this reconciliation missed.

### Front matter

- Welcome to Gopherbot — <code>welcome.md</code>
- Choose Your Path — <code>choose-your-path.md</code>
- Version, Feature, and Support Status — <code>status.md</code>

### Part I — Evaluate and understand

1. What Gopherbot Does—and What You Build — <code>evaluate/what-gopherbot-does.md</code>
2. When to Choose Gopherbot for DevOps Automation — <code>evaluate/when-to-choose.md</code>
3. Architecture and Responsibility Boundaries — <code>evaluate/architecture.md</code>
4. Core Concepts — <code>evaluate/core-concepts.md</code>
5. Security Model in Brief — <code>evaluate/security-brief.md</code>
6. Requirements and Support Matrix — <code>evaluate/requirements.md</code>

### Part II — Try Gopherbot over SSH

1. Start a Local Demonstration — <code>try/start-local-demo.md</code>
2. Connect with `bot-ssh` — <code>try/connect-bot-ssh.md</code>
3. Talk to the Robot — <code>try/commands-help-responses.md</code>
4. Follow a Command Through a Pipeline — <code>try/follow-a-pipeline.md</code>
5. Inspect, Stop, Reset, and Choose a Next Step — <code>try/inspect-stop-reset.md</code>

### Part III — Create your Robot

1. Plan the Robot — <code>create/plan.md</code>
2. Run Guided `new-robot` Setup — <code>create/new-robot.md</code>
3. Understand the Robot Repository — <code>create/repository-layout.md</code>
4. Configure the Development Environment — <code>create/development-environment.md</code>
5. Resume or Recover Setup — <code>create/resume-recover.md</code>
6. Commit and Prepare Production Handoff — <code>create/repository-handoff.md</code>
7. Recreate a Pre-v3 Robot — <code>create/recreate-pre-v3.md</code>

### Part IV — Build your first automation

1. How Extensions Make a Robot Useful — <code>first-automation/extensions.md</code>
2. Choose a Built-in Runtime — <code>first-automation/choose-runtime.md</code>
3. Create a Credential-Free Command Plugin — <code>first-automation/create-plugin.md</code>
4. Add Matching, Arguments, Help, and Safe Output — <code>first-automation/command-surface.md</code>
5. Check with `syntax` and `script` — <code>first-automation/check.md</code>
6. Enable, Exercise, Commit, and Push — <code>first-automation/ship.md</code>
7. Use the Extension-authoring Skill — <code>first-automation/skill.md</code>

### Part V — Configure behavior, identity, and state

1. Configuration Sources, Loading, and Precedence — <code>configure/sources-precedence.md</code>
2. Installed Defaults and Delta-only Custom Config — <code>configure/defaults-and-deltas.md</code>
3. Environments and Environment-specific Behavior — <code>configure/environments.md</code>
4. Variables, Parameters, Parameter Sets, and Templates — <code>configure/variables-parameters.md</code>
5. Encryption Keys, Secrets, and Secret Files — <code>configure/secrets.md</code>
6. Development File Brains and Production Cloud Brains — <code>configure/brains.md</code>
7. DynamoDB and Firestore — <code>configure/cloud-brain-providers.md</code>
8. OAuth Provider Setup and Per-user Credentials — <code>configure/oauth.md</code>
9. Enable and Customize Shipped Extensions — <code>configure/shipped-extensions.md</code>
10. Validate, Reload, and Restart — <code>configure/validate-reload.md</code>

### Part VI — Connect users and services

1. Connector Model and Canonical Identity — <code>connect/model-and-identity.md</code>
2. SSH Connector — <code>connect/ssh.md</code>
3. Slack — <code>connect/slack.md</code>
4. Google Chat — <code>connect/google-chat.md</code>
5. Run Multiple Connectors Safely — <code>connect/multiple-connectors.md</code>
6. Verify DMs, Hidden Commands, Threads, and Routing — <code>connect/verify-context.md</code>

### Part VII — Deploy and update a production Robot

1. Production Architecture and Readiness — <code>deploy/readiness.md</code>
2. Install on a Dedicated Linux Instance — <code>deploy/install-linux.md</code>
3. Run with systemd — <code>deploy/systemd.md</code>
4. Configure a Production Cloud Brain — <code>deploy/cloud-brain.md</code>
5. Runtime Environment, Instance Credentials, and Secrets — <code>deploy/runtime-secrets.md</code>
6. GitOps Development and Update Loop — <code>deploy/gitops.md</code>
7. Test Branches, Promote Main, and Roll Back — <code>deploy/promote-rollback.md</code>
8. Replace Failed Instances Safely — <code>deploy/replace-instance.md</code>
9. Containers and Kubernetes — <code>deploy/containers-kubernetes.md</code>
10. Verify Production Behavior — <code>deploy/verify.md</code>

### Part VIII — Secure the Robot

1. Trust Boundaries and Threat Model — <code>security/trust-boundaries.md</code>
2. Canonical Users, Validation, Administrators, and Groups — <code>security/identity-admins.md</code>
3. Private Commands and Message Confidentiality — <code>security/private-commands.md</code>
4. Authorization and Elevation — <code>security/authorization-elevation.md</code>
5. Extension Trust, Privilege, and Secret Scope — <code>security/extension-trust.md</code>
6. UID-only Privilege Separation — <code>security/privilege-separation.md</code>
7. Host, File, Network, Metadata, and Credential Hardening — <code>security/host-hardening.md</code>
8. Security Validation Checklist — <code>security/validation.md</code>

### Part IX — Operate and recover

1. Startup, Readiness, Reload, and Shutdown — <code>operations/lifecycle.md</code>
2. Status, Logs, Pipelines, and Failure Inspection — <code>operations/inspection.md</code>
3. Update, Switch Branches, Roll Back, and Recover — <code>operations/change-recovery.md</code>
4. Cloud-brain Persistence, Locking, Backup, and Restore — <code>operations/brain-recovery.md</code>
5. Synchronize a Cloud Brain for Local Development — <code>operations/brain-sync.md</code>
6. Schedules, Queues, and Automatic Work — <code>operations/automatic-work.md</code>
7. Connector and Provider Failure Behavior — <code>operations/provider-failures.md</code>
8. Incident and Disaster-recovery Playbooks — <code>operations/playbooks.md</code>

### Part X — Build advanced automation

1. Choose an Extension Form — <code>authoring/extension-forms.md</code>
2. Choose Go, Lua, JavaScript, or Gopherbot Shell — <code>authoring/built-in-runtimes.md</code>
3. Use Python When Its Ecosystem Is the Advantage — <code>authoring/python.md</code>
4. Matchers, Help, and Command Discovery — <code>authoring/matchers-help.md</code>
5. Compose Tasks, Jobs, and Pipelines — <code>authoring/pipelines.md</code>
6. Parameters, Secrets, Privilege, and Working Context — <code>authoring/context-secrets.md</code>
7. Robot API and Message-formatting Patterns — <code>authoring/robot-api-patterns.md</code>
8. Use Per-user OAuth Credentials Safely — <code>authoring/oauth.md</code>
9. Trigger a GitHub Workflow from JavaScript — <code>authoring/github-workflow.md</code>
10. Test with Local Checks and Integration Suites — <code>authoring/testing.md</code>
11. Package, Deploy, Version, and Roll Back — <code>authoring/lifecycle.md</code>
12. Use and Extend the Authoring Skill — <code>authoring/skill.md</code>

### Part XI — Reference

1. Command-line Reference — <code>reference/cli.md</code>
2. Environment Variable Reference — <code>reference/environment-variables.md</code>
3. Configuration Files and Precedence — <code>reference/configuration.md</code>
4. Robot and Extension Configuration Schemas — <code>reference/configuration-schemas.md</code>
5. Connector Capability Matrix and Settings — <code>reference/connectors.md</code>
6. Brain, History, Queue, and OAuth Provider Settings — <code>reference/providers.md</code>
7. Pipeline Semantics and API — <code>reference/pipelines.md</code>
8. Robot API by Language — <code>reference/robot-api.md</code>
9. BasicMarkdown and Message Formatting — <code>reference/basic-markdown.md</code>
10. Bundled Extensions and Providers — <code>reference/bundled.md</code>
11. Installation and Robot Filesystem Layouts — <code>reference/filesystem-layouts.md</code>
12. Terminology — <code>reference/terminology.md</code>
13. Troubleshooting and Diagnostic Index — <code>reference/troubleshooting.md</code>

## Move and removal policy

- Use clean moves without transition stubs. The refreshed structure should not
  carry duplicate pages solely to preserve old manual URLs.
- Contributor-only material preserved under `devdocs/` leaves the user manual
  rather than retaining a user-manual placeholder.
- Create the pre-cleanup tag immediately before the approved removal slice.
  Git history is the archive; do not retain an active `Outdated/` section.

## Concern-local readiness

Product-readiness and content gaps are resolved as part of the applicable
Phase 2.5, Phase 2.6, or concern-by-concern slice in <code>PLAN.md</code>, not
as a separate parallel checklist. Each slice still establishes source truth
and completes any required product work before its prose is accepted.

## Corpus disposition matrix

### Root and front-matter sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>Admin.md</code> | Merge | <code>operations/lifecycle.md</code>, <code>operations/inspection.md</code> |
| <code>BasicMarkdown.md</code> | Keep | <code>reference/basic-markdown.md</code> |
| <code>Basics.md</code> | Merge | <code>try/commands-help-responses.md</code> |
| <code>Configuration.md</code> | Merge | <code>reference/configuration.md</code> |
| <code>DevNotes.md</code> | Remove | Core-contributor history; Git retains it |
| <code>Environment-Variables.md</code> | Rewrite | <code>reference/environment-variables.md</code> |
| <code>Foreword.md</code> | Merge | <code>welcome.md</code> |
| <code>GopherDev.md</code> | Remove | Core-contributor material belongs outside the manual |
| <code>InstallOverview.md</code> | Remove | Superseded wrapper |
| <code>Installation.md</code> | Merge | <code>evaluate/requirements.md</code>, <code>reference/filesystem-layouts.md</code> |
| <code>Introduction.md</code> | Rewrite | <code>welcome.md</code>, <code>evaluate/what-gopherbot-does.md</code> |
| <code>Modules.md</code> | Remove | Obsolete stub |
| <code>PrivilegeSeparation.md</code> | Rewrite | <code>security/privilege-separation.md</code> |
| <code>RobotSetup.md</code> | Rewrite | <code>try/start-local-demo.md</code>, <code>create/new-robot.md</code> |
| <code>RunRobot.md</code> | Rewrite | <code>deploy/readiness.md</code> |
| <code>Security-Overview.md</code> | Rewrite | Part VIII security section |
| <code>Status.md</code> | Rewrite | <code>status.md</code> |
| <code>SUMMARY.md</code> | Rewrite | Approved final navigation |
| <code>Terminology.md</code> | Keep | <code>reference/terminology.md</code> |
| <code>Title.md</code> | Merge | <code>welcome.md</code> |
| <code>Upgrading.md</code> | Rewrite | <code>create/recreate-pre-v3.md</code> |

### Explicitly outdated sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>Outdated/Configuration.md</code> | Remove | Superseded by current configuration references |
| <code>Outdated/Linux-Install.md</code> | Remove | Superseded by current install/deploy material |
| <code>Outdated/Quick-Start.md</code> | Remove | Superseded by Parts II–IV |
| <code>Outdated/README.md</code> | Remove | Obsolete archive index |
| <code>Outdated/Windows-Install.md</code> | Remove | Unsupported legacy path |
| <code>Outdated/botsetup/ManualSetup.md</code> | Remove | Competing setup path is intentionally retired |
| <code>Outdated/botsetup/bothome.md</code> | Remove | Superseded by guided setup |
| <code>Outdated/botsetup/connect.md</code> | Remove | Superseded connector workflow |
| <code>Outdated/botsetup/copystd.md</code> | Remove | Copy-the-defaults workflow is obsolete |
| <code>Outdated/botsetup/finalenv.md</code> | Remove | Obsolete environment/bootstrap details |
| <code>Outdated/botsetup/finished.md</code> | Remove | Empty completion stub |
| <code>Outdated/botsetup/initcrypt.md</code> | Remove | Obsolete encryption bootstrap |
| <code>Outdated/botsetup/initenv.md</code> | Remove | Obsolete environment bootstrap |
| <code>Outdated/botsetup/saverobot.md</code> | Merge | <code>create/repository-handoff.md</code> after source verification |
| <code>Outdated/botsetup/sshkeys.md</code> | Remove | Obsolete self-managed key flow |
| <code>Outdated/containercli.md</code> | Remove | Obsolete container wrapper workflow |
| <code>Outdated/gitpodcli.md</code> | Remove | Unsupported legacy development path |
| <code>Outdated/upgrade/BotInfo.md</code> | Merge | <code>create/recreate-pre-v3.md</code> |
| <code>Outdated/upgrade/Custom-Dir.md</code> | Merge | <code>create/recreate-pre-v3.md</code> |
| <code>Outdated/upgrade/Encryption.md</code> | Merge | <code>create/recreate-pre-v3.md</code> |
| <code>Outdated/upgrade/External-Plugin.md</code> | Merge | <code>create/recreate-pre-v3.md</code> |
| <code>Outdated/upgrade/Memories.md</code> | Merge | <code>create/recreate-pre-v3.md</code> |
| <code>Outdated/upgrade/robot-yaml.md</code> | Merge | <code>create/recreate-pre-v3.md</code> |

### Robot API sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>api/API-Introduction.md</code> | Keep | <code>reference/robot-api.md</code> landing page |
| <code>api/Attribute-Retrieval-API.md</code> | Keep | Robot API attribute subpage |
| <code>api/Brain-API.md</code> | Keep | Robot API memory subpage |
| <code>api/Languages.md</code> | Rewrite | <code>first-automation/choose-runtime.md</code>, Part X runtime chapters |
| <code>api/Message-Sending-API.md</code> | Keep | Robot API messaging subpage |
| <code>api/Misc-Methods.md</code> | Rewrite | Split into the applicable Robot API subpages |
| <code>api/Pipeline-API.md</code> | Keep | <code>reference/pipelines.md</code>, Robot API pipeline subpage |
| <code>api/Response-Request-API.md</code> | Keep | Robot API prompting subpage |
| <code>api/Utility-API.md</code> | Keep | Robot API utility subpage |

### Appendix and connector stubs

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>appendices/Appendix.md</code> | Remove | Empty wrapper |
| <code>appendices/InstallArchive.md</code> | Remove | Historical install archive |
| <code>appendices/Protocols.md</code> | Merge | <code>reference/connectors.md</code> |
| <code>appendices/nullconn.md</code> | Remove | Internal connector stub |
| <code>appendices/rocket.md</code> | Remove | Unsupported connector stub |
| <code>appendices/slack.md</code> | Merge | <code>connect/slack.md</code> |
| <code>appendices/terminal.md</code> | Remove | Connector is pending retirement |
| <code>appendices/testproto.md</code> | Remove | Internal test connector |

### Daily-use sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>basics/channels.md</code> | Rewrite | <code>connect/verify-context.md</code>, <code>security/private-commands.md</code> |
| <code>basics/context.md</code> | Merge | <code>connect/model-and-identity.md</code>, <code>connect/verify-context.md</code> |
| <code>basics/help.md</code> | Rewrite | <code>try/commands-help-responses.md</code>, <code>authoring/matchers-help.md</code> |
| <code>basics/matching.md</code> | Merge | <code>first-automation/command-surface.md</code>, <code>authoring/matchers-help.md</code> |
| <code>basics/ping.md</code> | Merge | <code>try/commands-help-responses.md</code> |
| <code>basics/stdplugins.md</code> | Rewrite | <code>reference/bundled.md</code> |

### Core-contributor sources currently under the manual

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>botdev/DevelRobot.md</code> | Remove | Obsolete contributor stub |
| <code>botdev/IDE.md</code> | Remove | Obsolete contributor environment |
| <code>botdev/IntegrationTests.md</code> | Merge | Preserve current material under `devdocs/`; extract extension-facing checks into <code>authoring/testing.md</code> |
| <code>botdev/StructsInterfaces.md</code> | Remove | Core-contributor material belongs outside the manual |
| <code>botdev/protocols.md</code> | Remove | Empty contributor stub |
| <code>botdev/releases.md</code> | Remove | Stale contributor release procedure |

### Setup sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>botsetup/Plugin.md</code> | Rewrite | <code>try/start-local-demo.md</code>, <code>create/new-robot.md</code>, <code>create/resume-recover.md</code> |
| <code>botsetup/Requirements.md</code> | Merge | <code>evaluate/requirements.md</code>, <code>create/development-environment.md</code> |
| <code>botsetup/credentials.md</code> | Rewrite | <code>configure/secrets.md</code>, connector setup chapters |
| <code>botsetup/gopherhome.md</code> | Rewrite | <code>create/repository-layout.md</code>, <code>reference/filesystem-layouts.md</code> |
| <code>botsetup/slacksock.md</code> | Rewrite | <code>connect/slack.md</code> |

### Configuration sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>config/file.md</code> | Keep | <code>configure/sources-precedence.md</code>, <code>reference/configuration.md</code> |
| <code>config/job-plug.md</code> | Remove | Empty wrapper |
| <code>config/plugin-yaml.md</code> | Keep | <code>reference/configuration-schemas.md</code> extension schema |
| <code>config/robot-yaml.md</code> | Keep | <code>reference/configuration-schemas.md</code> Robot schema |
| <code>config/templates.md</code> | Keep | <code>configure/variables-parameters.md</code>, configuration reference subpage |
| <code>config/troubleshooting.md</code> | Rewrite | <code>reference/troubleshooting.md</code> |

### Introductory extension-authoring sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>customizing.md</code> | Rewrite | <code>first-automation/extensions.md</code>, <code>authoring/extension-forms.md</code> |
| <code>customizing/first-plugin.md</code> | Rewrite | <code>first-automation/create-plugin.md</code> through <code>first-automation/ship.md</code> |
| <code>customizing/style.md</code> | Merge | <code>configure/defaults-and-deltas.md</code>, relevant authoring chapters |
| <code>customizing/syntax-help.md</code> | Keep | <code>authoring/matchers-help.md</code> |

### Deployment sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>deploy/Container.md</code> | Merge | <code>deploy/containers-kubernetes.md</code> |
| <code>deploy/DockerDeploy.md</code> | Merge | <code>deploy/containers-kubernetes.md</code> |
| <code>deploy/Kubernetes.md</code> | Merge | <code>deploy/containers-kubernetes.md</code> |
| <code>deploy/deploy-environment.md</code> | Rewrite | <code>deploy/runtime-secrets.md</code> |
| <code>deploy/systemd.md</code> | Rewrite | <code>deploy/systemd.md</code> |

### Extension workflow sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>extensiondev/CLI.md</code> | Rewrite | <code>first-automation/check.md</code>, <code>authoring/testing.md</code>, <code>reference/cli.md</code> |
| <code>extensiondev/devenv.md</code> | Merge | <code>create/development-environment.md</code> as an optional path |
| <code>extensiondev/local.md</code> | Rewrite | <code>first-automation/ship.md</code>, <code>authoring/lifecycle.md</code> |
| <code>extensiondev/secrets.md</code> | Rewrite | <code>configure/secrets.md</code>, <code>authoring/context-secrets.md</code> |
| <code>extensiondev/terminal.md</code> | Remove | Obsolete connector stub |

### Installation sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>install/LinuxInstall.md</code> | Rewrite | <code>try/start-local-demo.md</code>, <code>reference/filesystem-layouts.md</code> |
| <code>install/ManualInstall.md</code> | Rewrite | <code>deploy/install-linux.md</code>, <code>reference/filesystem-layouts.md</code> |
| <code>install/Requirements.md</code> | Rewrite | <code>evaluate/requirements.md</code> |

### Pipeline and bundled-task sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>pipelines/TaskEnvironment.md</code> | Keep | <code>authoring/context-secrets.md</code>, pipeline reference subpage |
| <code>pipelines/fail.md</code> | Merge | <code>authoring/pipelines.md</code>, <code>reference/pipelines.md</code> |
| <code>pipelines/final.md</code> | Merge | <code>authoring/pipelines.md</code>, <code>reference/pipelines.md</code> |
| <code>pipelines/integrations.md</code> | Remove | Empty integration stub |
| <code>pipelines/jobspipes.md</code> | Rewrite | <code>authoring/extension-forms.md</code>, <code>authoring/pipelines.md</code> |
| <code>pipelines/primary.md</code> | Rewrite | <code>try/follow-a-pipeline.md</code>, <code>authoring/pipelines.md</code>, <code>reference/pipelines.md</code> |
| <code>pipelines/ssh.md</code> | Merge | <code>reference/bundled.md</code>, advanced task patterns |
| <code>pipelines/tasks.md</code> | Rewrite | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/email-log.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/git-command.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/pause-brain.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/pause.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/restart-robot.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/resume-brain.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/robot-quit.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/rotate-log.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/send-message.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/ssh-agent.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/ssh-git-helper.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/tail-log.md</code> | Merge | <code>reference/bundled.md</code> |
| <code>pipelines/tasks/template.md</code> | Merge | <code>reference/bundled.md</code> |

### Command-authoring, security, and operations sources

| Current source | Disposition | Proposed target |
| --- | --- | --- |
| <code>reference/README.md</code> | Remove | Empty wrapper replaced by focused reference entries |
| <code>reference/SimpleMatcher.md</code> | Keep | Matcher reference subpage under <code>reference/configuration-schemas.md</code> |
| <code>security/userapproval.md</code> | Merge | <code>security/authorization-elevation.md</code>, <code>reference/bundled.md</code> |
| <code>usage/admin.md</code> | Rewrite | <code>operations/lifecycle.md</code>, <code>operations/inspection.md</code> |
| <code>usage/logging.md</code> | Rewrite | <code>operations/inspection.md</code> |
| <code>usage/update.md</code> | Rewrite | <code>deploy/gitops.md</code>, <code>deploy/promote-rollback.md</code>, <code>operations/change-recovery.md</code> |

## Phase 2B approval

The owner approved the concrete target tree and all page dispositions on
2026-08-31. The owner selected clean moves without transition stubs and
confirmed the Phase 2.5/2.6 sequence. Readiness gaps are handled within their
applicable project slices rather than maintained as a separate checklist.
