# Ideal North-Star Table of Contents

Status: revised Phase 2A proposal for owner approval. This is a greenfield
target, not a rearrangement of the imported manual.

## Design premise

Gopherbot is an automation framework and driver, not a useful appliance by
itself. A functional Robot is expected to contain custom extensions that
connect team chat, automation events, SaaS APIs, and infrastructure. The
primary reader is therefore both a DevOps operator and an automation author.

The manual must help that reader understand what a Robot can do, reach a safe
and observable success over SSH, create useful custom automation, deploy it,
and operate it confidently. Navigation follows user work rather than repository
layout, legacy pages, or API shape.

Design rules:

- Lead with outcomes, supported paths, and automation possibilities.
- Provide one short SSH path from evaluation to a locally working Robot.
- Treat guided `new-robot` onboarding and creation of a first custom extension
  as consecutive steps in the primary journey.
- Explain security and recovery where choices create risk, then consolidate the
  complete models in dedicated sections.
- Teach installed defaults as authoritative and custom Robot configuration as
  intentional deltas.
- Require cloud brains for production Robots expected to remember; teach file
  brains as local-development and cloud-to-local synchronization tools.
- Keep task-oriented guidance in the main journey and dense schemas/API
  material in reference.
- Give pre-v3 owners a concise rebuild-and-port path rather than mixing legacy
  configurations into the clean v3 story.
- Delete obsolete documentation after a pre-cleanup tag instead of maintaining
  a browsable historical archive.

## Primary journey

Evaluate → try over SSH → create a Robot with `new-robot` → create a first
extension → commit and push → deploy to a dedicated compute instance → operate
and extend through the GitOps loop.

## Proposed navigation spine

### Front matter

- **Welcome to Gopherbot** — explain the automation framework, its intended
  outcomes, and the expectation that useful Robots contain custom extensions.
- **Choose Your Path** — direct evaluators, new Robot owners, automation
  authors, day-two operators, and pre-v3 owners to their starting points.
- **Version, Feature, and Support Status** — identify the documented version,
  supported connectors and deployment paths, current limitations, and material
  platform caveats.

### Part I — Evaluate and understand

Help a prospective owner decide whether Gopherbot fits an automation program
that might otherwise use generic Jenkins automation or bespoke services around
GitHub/GitLab builds and team chat.

1. **What Gopherbot Does—and What You Build**
2. **When to Choose Gopherbot for DevOps Automation**
3. **Architecture and Responsibility Boundaries**
4. **Core Concepts: Robots, Connectors, Extensions, Pipelines, and Brains**
5. **Security Model in Brief**
6. **Requirements and Support Matrix**

### Part II — Try Gopherbot over SSH

Provide the shortest disposable path to a successful interaction using the SSH
connector and no production credentials. The Terminal connector is not part of
the refreshed manual.

1. **Start a Local Demonstration**
2. **Connect with `bot-ssh`**
3. **Talk to the Robot: Commands, Help, and Responses**
4. **Follow a Command Through a Pipeline**
5. **Inspect, Stop, Reset, and Choose a Next Step**

### Part III — Create your Robot

Turn the demonstration into an owned, versioned Robot configuration through
the single supported onboarding journey.

1. **Plan the Robot: Repository, Environment, Chat, Brain, and Host**
2. **Run the Guided `new-robot` Setup**
3. **Understand the Robot Repository and `custom/` Tree**
4. **Configure and Validate the Development Environment**
5. **Resume or Recover an Interrupted Setup**
6. **Commit the Robot and Prepare the Production Handoff**
7. **Recreate a Pre-v3 Robot on the v3 Skeleton** — create a clean Robot, then
   deliberately port current secrets, persistent data, and extensions.

No competing manual-assembly tutorial belongs in this refresh.

### Part IV — Build your first automation

Make extension creation part of initial success, not an advanced optional
persona. Start without external credentials so every reader can complete the
local edit/test/reload loop.

1. **How Extensions Make a Robot Useful**
2. **Choose a Built-in Runtime for the First Command**
3. **Create a Credential-Free Command Plugin**
4. **Add Matching, Arguments, Help, and Safe Output**
5. **Check with `syntax` and `script`**
6. **Enable, Exercise, Commit, and Push the Extension**
7. **Create Extensions with the `write-robot-extension` AI Skill**

### Part V — Configure behavior, identity, and state

Teach governing models before exhaustive schemas and before the first external
SaaS integration.

1. **Configuration Sources, Loading, and Precedence**
2. **Installed Defaults and Delta-Only Custom Configuration**
3. **Environments and Environment-Specific Behavior**
4. **Variables, Parameters, Parameter Sets, and Templates**
5. **Encryption Keys, Secrets, and Secret Files**
6. **Development File Brains and Production Cloud Brains**
7. **DynamoDB for AWS and Firestore for Other Cloud Environments**
8. **OAuth Provider Setup and Per-user Identity Credentials** — distinguish
   provider-specific setup/linking from provider-neutral storage, refresh, and
   short-lived credential use.
9. **Enable and Customize Shipped Extensions**
10. **Validate, Reload, and Restart After Changes**

### Part VI — Connect users and services

Configure transports while preserving the distinction between transport
identity, canonical security identity, and message privacy.

1. **Connector Model and Canonical User Identity**
2. **SSH Connector for Development and Administration**
3. **Slack**
4. **Google Chat**
5. **Run Multiple Connectors Safely**
6. **Verify DMs, Hidden Commands, Threads, and Message Routing**

### Part VII — Deploy and update a production Robot

Lead with a dedicated Linux compute instance managed by systemd and updated
through the Robot's git repository. Containers are supported but not the
recommended operating model.

1. **Production Architecture and Readiness Checklist**
2. **Install the Release on a Dedicated Linux Instance**
3. **Run Gopherbot with systemd**
4. **Configure a Production Cloud Brain**
5. **Provide Runtime Environment, Instance Credentials, and Secrets**
6. **Use the GitOps Development and Update Loop**
7. **Test Branches, Promote Main, and Roll Back**
8. **Replace Failed Instances Safely** — use cloud-brain locking and one active
   Robot; an auto-scaling group of size one is resilience, not active/active HA.
9. **Containers and Kubernetes: Supported but Not Recommended** — document a
   single Robot replica, durable state, replacement behavior, and limitations.
10. **Verify Startup, Connectivity, Persistence, Update, and Shutdown**

### Part VIII — Secure the Robot

Present the complete security model after the reader has concrete connector,
configuration, extension, and deployment context.

1. **Trust Boundaries and Threat Model**
2. **Canonical Users, Validation, Administrators, and Groups**
3. **Private Commands and Message Confidentiality**
4. **Authorization and Elevation**
5. **Extension Trust, Privilege, and Secret Scope**
6. **UID-Only Privilege Separation**
7. **Host, File, Network, Metadata, and Credential Hardening**
8. **Security Validation Checklist**

### Part IX — Operate and recover

Organize day-two work around observation, controlled change, durable cloud
state, and failure recovery.

1. **Startup, Readiness, Reload, and Shutdown**
2. **Status, Logs, Pipelines, and Failure Inspection**
3. **Update, Switch Branches, Roll Back, and Recover**
4. **Cloud-brain Persistence, Locking, Backup, and Restore**
5. **Synchronize a Cloud Brain for Local Development**
6. **Schedules, Queues, and Automatic Work**
7. **Connector and Provider Failure Behavior**
8. **Incident and Disaster-Recovery Playbooks**

### Part X — Build advanced automation

Cover all extension forms and the runtimes that account for practical Robot
automation. Lead with built-in interpreters; make Python the primary external
language. Ruby and Bash remain compatibility surfaces, not preferred authoring
paths.

1. **Choose Plugins, Jobs, Tasks, Pipelines, or Shared Libraries**
2. **Choose Go, Lua, JavaScript, or Gopherbot Shell**
3. **Use Python When Its Library Ecosystem Is the Advantage**
4. **Matchers, Help, and Command Discovery**
5. **Compose Tasks, Jobs, and Pipelines**
6. **Parameters, Secrets, Privilege, and Working Context**
7. **Robot API and Message-formatting Patterns**
8. **Use Per-user OAuth Credentials Safely**
9. **Tutorial: Trigger a GitHub Workflow from JavaScript**
10. **Test with Local Checks and Focused Integration Suites**
11. **Package, Deploy, Version, and Roll Back Extensions**
12. **Use and Extend the `write-robot-extension` Skill**

The GitHub tutorial uses the generic engine-owned OAuth storage/refresh model
and declarative provider configuration. A maintained GitHub plugin may perform
the provider-specific application setup, consent, or linking work, however
complex, but its output is generic configuration plus securely stored
long-lived credential material. The tutorial extension obtains a short-lived
credential through the standard Robot API for each GitHub operation; it does
not know how the provider was bootstrapped. Provider-specific code is limited
to setup/authorization UX or genuine behavior that the generic model cannot
express. The workflow trigger is tutorial code, not a shipped extension.

### Part XI — Reference

Collect material readers look up nonlinearly after learning the governing
concepts in task-oriented chapters.

1. **Command-Line Reference**
2. **Environment Variable Reference**
3. **Configuration File and Precedence Reference**
4. **Robot and Extension Configuration Schemas**
5. **Connector Capability Matrix and Settings**
6. **Brain, History, Queue, and OAuth Provider Settings**
7. **Pipeline Semantics and API Reference**
8. **Robot API by Language**
9. **BasicMarkdown and Message Formatting**
10. **Bundled Plugins, Jobs, Tasks, and Providers**
11. **Installation and Robot Filesystem Layouts**
12. **Terminology**
13. **Troubleshooting and Diagnostic Index**

Core-engine contributor documentation remains outside the operator manual.
After a pre-cleanup tag provides historical access, obsolete imported pages are
deleted rather than retained in active documentation or an archive section.

## Accepted product and documentation decisions

1. **Audience:** Robot owners/operators and extension authors are co-primary
   concerns, usually embodied by the same DevOps engineer. Core-engine
   contributors are out of scope for this manual.
2. **Recommended journey:** local SSH demonstration → guided `new-robot` setup
   → first custom extension → git commit/push → dedicated-instance deployment →
   day-two GitOps operation and continued automation development.
3. **Deployment priority:** a dedicated Linux compute instance with systemd is
   the lead path. Containers and Kubernetes are supported but nonrecommended,
   single-replica alternatives. Gopherbot does not provide active/active or
   active/passive Robot HA.
4. **Connector boundary:** SSH, Slack, and Google Chat are the active documented
   connector set. Terminal is omitted and tracked for later removal.
5. **Onboarding contract:** guided `new-robot` is the sole documented creation
   path in this epic.
6. **Security posture:** short orientation precedes trial, risk guidance appears
   in task context, and the full model has a production section.
7. **Pre-v3 boundary:** recreate a clean v3 Robot and port what is still needed;
   do not preserve legacy configurations or build a large migration manual.
8. **Reference boundary:** schemas and method inventories live in reference;
   task chapters link to them instead of duplicating them.
9. **Cleanup boundary:** create a pre-cleanup tag, then delete obsolete pages
   rather than maintaining an archive.
10. **Brain boundary:** file brains are for local development; production
    Robots expected to remember require cloud brains. DynamoDB is preferred on
    AWS and Firestore is the general alternative.
11. **Extension skill:** ship a portable Agent Skills package named
    `write-robot-extension` under `resources/skills/`. It covers built-in Go,
    Lua, JavaScript, and Gopherbot shell plus Python, and all principal extension
    forms. Ruby and Bash are not preferred targets.
12. **OAuth boundary:** provider-specific setup may be arbitrarily complex and
    may live in a maintained plugin, but it must converge on provider-neutral
    configuration and secure long-lived credential storage. The engine owns
    generic token storage, refresh, and standards-based retrieval of
    short-lived credentials for individual API operations. Provider
    configuration owns standard endpoints, parameters, headers, scopes, and
    client authentication; provider adapters are an escape hatch for setup and
    authorization UX or genuine nonstandard behavior. Ordinary extensions use
    the generic Robot API and remain unaware of the setup mechanism. Live
    provider validation is required before the model is taught as
    production-ready.

## Phase 2A approval question

Does this revised navigation spine capture the approved product story strongly
enough to begin the pre-v3 policy update and corpus reconciliation? Phase 2B
may propose an extension only when a genuine user journey or operational
concern is missing; legacy page existence alone is not justification.
