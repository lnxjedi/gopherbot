# Google Chat Pub/Sub Quickstart Control

Reference note:

- this file is older field-note material and control-experiment guidance
- the preferred setup path is now `resources/gcloud/README.md`
- keep this file around for comparison and troubleshooting until the newer flow is fully settled

This file is a Cloud Shell companion to Google's official Pub/Sub quickstart for a Google Chat app:

- Official quickstart: https://developers.google.com/workspace/chat/quickstart/pub-sub

It is organized to track the quickstart's major sections so you can compare what you run in Cloud Shell with what the official guide says to do.

This is a control experiment. The goal is to run Google's published Python sample with the least possible extra machinery:

- no ambient Workspace Events
- no Marketplace SDK
- no admin install
- no Bishop-specific code path

If this control works, then Google Chat interaction events are fine and the remaining issue is in our app/config path. If this control fails in the same way, the issue is upstream of Bishop.

## Prerequisites

Before you start:

- use a Google Workspace account
- ensure billing is enabled for the new project
- use Cloud Shell

Set the base variables. The first command uses the exact project name you requested:

```bash
export PROJECT_ID="bishop-gopherbot"
export PROJECT_NAME="chatapi-bishop-gopherbot"
export REGION="us-central1"
export TOPIC_ID="bishop-chat-topic"
export SUBSCRIPTION_ID="bishop-chat-sub"
export SERVICE_ACCOUNT_ID="bishop-chat-sa"
export SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_ID}@${PROJECT_ID}.iam.gserviceaccount.com"
export WORKDIR="${HOME}/gopherbot-chatapi-bishop"
```

Create the working directory:

```bash
mkdir -p "${WORKDIR}"
cd "${WORKDIR}"
```

## Set Up The Environment

The quickstart says to enable the Google Chat API and Pub/Sub API. In Cloud Shell, do:

```bash
gcloud projects create "${PROJECT_ID}" --name="${PROJECT_NAME}"
gcloud config set project "${PROJECT_ID}"
```

If billing is not already attached, do that in the Console before continuing.

Then enable the base APIs:

> NOTE: Not used
```bash
gcloud services enable \
  chat.googleapis.com \
  pubsub.googleapis.com
```

## Set Up Pub/Sub

The quickstart says to:

1. create a Pub/Sub topic
2. grant Chat permission to publish
3. create a service account
4. create a pull subscription
5. assign the Pub/Sub Subscriber role on the subscription

### 1. Create A Pub/Sub Topic

Try the plain topic create first:

```bash
gcloud pubsub topics create "${TOPIC_ID}"
```

Do not add `--message-storage-policy-allowed-regions="${REGION}"` here if this
topic might ever be reused for Google Chat ambient Workspace Events.

Why this matters:

- a narrow Pub/Sub topic region policy can look fine during initial setup
- later, Workspace Events subscriptions can become `SUSPENDED` with reason
  `OTHER` when Google publishes from another region
- that failure is extremely difficult to trace back to the original topic
  create command

### 2. Grant Chat Permission To Publish

The official quickstart grants `roles/pubsub.publisher` to:

- `chat-api-push@system.gserviceaccount.com`

Run:

```bash
gcloud pubsub topics add-iam-policy-binding "${TOPIC_ID}" \
  --member="serviceAccount:chat-api-push@system.gserviceaccount.com" \
  --role="roles/pubsub.publisher"
```

### 3. Create A Service Account

```bash
gcloud iam service-accounts create "${SERVICE_ACCOUNT_ID}" \
  --display-name="Quickstart Google Chat Service Account"
```

Create the key file in the current directory:

```bash
gcloud iam service-accounts keys create "${WORKDIR}/gopherbot-key.json" \
  --iam-account="${SERVICE_ACCOUNT_EMAIL}"
```

### 4. Create A Pull Subscription

```bash
gcloud pubsub subscriptions create "${SUBSCRIPTION_ID}" \
  --topic="${TOPIC_ID}" \
  --expiration-period=never
```

### 5. Assign The Pub/Sub Subscriber Role On The Subscription

The quickstart scopes this to the subscription, so do the same here:

```bash
gcloud pubsub subscriptions add-iam-policy-binding "${SUBSCRIPTION_ID}" \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/pubsub.subscriber"
```

#### Also ran:
```bash
gcloud pubsub subscriptions add-iam-policy-binding "${SUBSCRIPTION_ID}" \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/pubsub.viewer"
```

## Write The Script

The quickstart's Python section says to:

1. provide service account credentials
2. provide the project ID
3. provide the subscription ID
4. create `requirements.txt`
5. create `app.py`

### 1. Provide Service Account Credentials

```bash
export GOOGLE_APPLICATION_CREDENTIALS="${WORKDIR}/gopherbot-key.json"
```

### 2. Provide The Google Cloud Project ID

```bash
export PROJECT_ID="${PROJECT_ID}"
```

### 3. Provide The Pub/Sub Subscription ID

```bash
export SUBSCRIPTION_ID="${SUBSCRIPTION_ID}"
```

### 4. Create `requirements.txt`

Fetch the official sample file directly:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/googleworkspace/google-chat-samples/main/python/pub-sub-app/requirements.txt \
  -o requirements.txt
```

### 5. Create `app.py`

Fetch the official sample file directly:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/googleworkspace/google-chat-samples/main/python/pub-sub-app/app.py \
  -o app.py
```

If you want to verify the sample contents match the quickstart:

```bash
sed -n '1,220p' requirements.txt
sed -n '1,260p' app.py
```

## Configure The Chat App

This section is still manual in the Console.

Open:

- **APIs & Services > Enabled APIs & services > Google Chat API > Configuration**

Then follow the quickstart's Chat app settings as closely as possible:

1. **App name**: `Quickstart App`
2. **Avatar URL**: `https://developers.google.com/chat/images/quickstart-app-avatar.png`
3. **Description**: `Quickstart app`
4. Under **Functionality**, enable **Join spaces and group conversations**
5. Under **Functionality**, also enable **Receive 1:1 messages**
6. Under **Connection settings**, select **Cloud Pub/Sub**
7. Under **Topic ID**, paste:

```text
projects/PROJECT_ID/topics/quickstart-chat-topic
```

Replace `PROJECT_ID` with your real project ID.

8. Under **Visibility**, select **Make this Google Chat app available to specific people and groups in your domain**
9. Enter your own Workspace email address only
10. Under **Logs**, select **Log errors to Logging**
11. Click **Save**

Wait a few minutes after saving.

## Run The Script

The quickstart says:

```bash
python -m venv env
source env/bin/activate
pip install -r requirements.txt -U
python app.py
```

Run exactly that:

```bash
cd "${WORKDIR}"
python -m venv env
source env/bin/activate
pip install -r requirements.txt -U
python app.py
```

If it starts correctly, you should see it listening on the Pub/Sub subscription.

## Test Your Chat App

The quickstart says to open a DM with the Chat app and send `Hello`.

Do exactly that:

1. Open Google Chat with the same Workspace account you entered in **Visibility**
2. Click **New chat**
3. Search for `Quickstart App`
4. Open the DM
5. Send:

```text
Hello
```

Expected control result:

- the Python sample logs the inbound event
- the app replies with an echo-style response

## Troubleshoot

If the Chat app still says `not responding`, check these in order:

### 1. Confirm The Sample Process Is Still Running

Your Cloud Shell terminal should still be running `python app.py`.

### 2. Confirm The Topic And Subscription Exist

```bash
gcloud pubsub topics describe "${TOPIC_ID}"
gcloud pubsub subscriptions describe "${SUBSCRIPTION_ID}"
```

### 3. Confirm Chat Can Publish To The Topic

```bash
gcloud pubsub topics get-iam-policy "${TOPIC_ID}"
```

Look for:

```text
serviceAccount:chat-api-push@system.gserviceaccount.com
roles/pubsub.publisher
```

### 4. Confirm The Service Account Can Pull From The Subscription

```bash
gcloud pubsub subscriptions get-iam-policy "${SUBSCRIPTION_ID}"
```

Look for:

```text
serviceAccount:QUICKSTART_SERVICE_ACCOUNT_EMAIL
roles/pubsub.subscriber
```

### 5. Confirm The Chat App Config Matches The Quickstart

Re-check:

- `Join spaces and group conversations`
- `Receive 1:1 messages`
- `Cloud Pub/Sub`
- correct topic name
- your email in `Visibility`
- `Log errors to Logging`

### 6. Check Chat App Error Logs

In Logs Explorer, query:

```text
resource.type="chat.googleapis.com/Project"
severity=ERROR
```

If you see errors, save them.

### 7. Optional: Inspect Raw Pub/Sub Messages

Create a temporary debug subscription:

```bash
gcloud pubsub subscriptions create quickstart-chat-debug-sub \
  --topic="${TOPIC_ID}" \
  --expiration-period=never
```

Then, after a failed DM test, pull from it:

```bash
gcloud pubsub subscriptions pull quickstart-chat-debug-sub \
  --limit=10 \
  --auto-ack \
  --format=json
```

If even the official quickstart does not receive DM events in this fresh project, the issue is upstream of Bishop and likely in Google-side app delivery or org policy behavior.

## Phase 2: Add Ambient Message Support

There does not appear to be a single Google quickstart for "Chat app with Pub/Sub interaction events plus ambient Workspace Events subscriptions".

The closest official documents are:

- Chat app authentication and one-time administrator approval for `chat.app.*` scopes:
  https://developers.google.com/workspace/chat/authenticate-authorize-chat-app
- General Chat app auth overview:
  https://developers.google.com/workspace/chat/authenticate-authorize
- Google Chat events via the Workspace Events API:
  https://developers.google.com/workspace/events/guides/events-chat
- Create a Google Workspace subscription:
  https://developers.google.com/workspace/events/guides/create-subscription
- Publish a private Marketplace app:
  https://developers.google.com/workspace/marketplace/how-to-publish

This section turns those into the closest thing we currently have to a reproducible ambient-message setup flow for Gopherbot.

### Goal

Keep the known-good interactive baseline working:

- DM works
- `@mention` works
- slash command works

Then add:

- administrator-approved `chat.app.*` scopes
- private Marketplace publication
- admin install
- connector-managed per-space Workspace Events subscriptions
- `AmbientMessages: true`

### Before You Start

Do this only after Phase 1 is working.

Before adding any ambient setup, confirm again:

- DM `hello` works
- `@Bishop help` works
- `/bishop ping` works

If those are not all working, stop and fix that first.

### 1. Enable The Workspace Events API

In Cloud Shell:

```bash
gcloud services enable workspaceevents.googleapis.com
```

### 2. Prepare The Service Account For `chat.app.*` Scopes

This part is manual in the Google Cloud console.

Before the Marketplace-compatible OAuth client step, enable the Marketplace SDK/API in Cloud Shell:

```bash
gcloud services enable appsmarket-component.googleapis.com
```

Open:

- **IAM & Admin > Service Accounts**
- click the service account used by the Chat app
- click **Advanced settings**

If Google refuses to let you create the Marketplace-compatible OAuth client yet, it may first require an OAuth consent screen to exist for the project.

If that happens:

1. open **Google Auth Platform > Branding** (or the older **OAuth consent screen** UI if that is what Google shows you)
2. configure the minimum required fields
3. choose **Internal** / user-organization-only visibility for a Workspace-internal bot
4. save the consent-screen configuration

Important note:

- this consent-screen setup is confusing but expected
- it is a platform prerequisite for creating the Marketplace-compatible OAuth client
- it does **not** mean the Chat app will use a normal end-user OAuth consent flow at runtime

Then:

1. click **Create Google Workspace Marketplace-compatible OAuth client**
2. wait for that to complete

This is the Google-required prerequisite for app-auth scopes like:

- `https://www.googleapis.com/auth/chat.app.messages.readonly`

Important note:

- this is not the normal end-user OAuth consent flow
- this is preparation for one-time administrator approval of `chat.app.*` scopes

### 3. Configure The Marketplace SDK

Open:

- **Google Workspace Marketplace SDK**

Then:

1. configure the app metadata as a private/internal app
2. add the app-auth scopes needed for ambient Chat subscriptions
3. `Save draft`
4. continue with the publish/install flow documented in `resources/gcloud/README.md`

Notes:

- Google requires app listing images for the Marketplace store listing
- prepare a square avatar image ahead of time
- for a simple internal setup, you can upload the same avatar image for all required image fields
- in the current Google UI, `Save draft` may be the only explicit action available at this stage
- after `Save draft`, follow the repo-level setup guide in `resources/gcloud/README.md` for the next publish/install steps
- the current Google admin help page for that flow is:
  https://knowledge.workspace.google.com/admin/chat/set-up-app-authorization-for-chat
- for Bishop's current ambient implementation, add `https://www.googleapis.com/auth/chat.app.messages.readonly`
- do **not** add `chat.bot` here; Google Chat already includes it for the app
- `chat.app.spaces` and `chat.app.memberships` are not needed for Bishop's current message-created ambient subscription flow

### 4. Admin-Install The App

Treat `resources/gcloud/README.md` as the authoritative guide from this point onward for the publish/admin-install flow.

The current Google admin help page appears to be:

- https://knowledge.workspace.google.com/admin/chat/set-up-app-authorization-for-chat

Depending on the UI/version, Google may route you there directly after the Marketplace/OAuth-client setup, or you may need to follow the publish/install sequence documented in `resources/gcloud/README.md`.

If you are taken through the Admin console manually, the flow is still roughly:

1. locate the Chat app authorization/install step
2. choose the app
3. complete admin authorization/install
4. review the data access requirements
5. finish the setup

This is the point where the `chat.app.*` scopes are effectively approved for the app.

### 5. Re-Check The Chat API Configuration

Open:

- **APIs & Services > Enabled APIs & services > Google Chat API > Configuration**

Confirm all of these are still correct:

- **Join spaces and group conversations** is enabled
- **Receive 1:1 messages** is enabled
- **Cloud Pub/Sub** is still the connection setting
- **Topic ID** still points at the same Pub/Sub topic
- **Commands** still includes your slash command
- **Visibility** is set the way you intend for current testing
- **Log errors to Logging** is enabled

### 6. Turn On Ambient Messages In Bishop

In Bishop's Google Chat protocol config, enable the ambient path:

```yaml
ProtocolConfig:
  AmbientMessages: true
```

Keep the rest of the known-good interactive settings the same:

- `ProjectID`
- `SubscriptionID`
- `CredentialsEncryptedFile`
- `UserMap`
- `SlashCommand`

If you created a new service-account key during this process and it differs from the old one, re-encrypt it before restarting Bishop:

```bash
gopherbot encrypt -f path/to/gopherbot-key.json > gopherbot-key.json.enc
```

If the key file did not change, the existing encrypted JSON is still fine.

### 7. Restart Bishop And Watch The Logs

After restarting, look for messages like:

- ambient subscription creation for joined spaces
- ambient subscription renewal
- any Workspace Events API warnings or failures

Expected healthy behavior:

- the connector lists joined spaces
- it creates or renews per-space subscriptions
- DM, mention, and slash behavior still work

### 8. Test Ambient Behavior Carefully

Use a space where Bishop is already present.

Test these one at a time:

1. plain message that is not addressed to the bot
2. `Bishop, ping`
3. `@Bishop ping`
4. `Did you see what @Bishop did?`
5. `/bishop ping`

Expected behavior:

- plain ambient messages are seen by the bot but not treated as addressed-to-bot commands
- `Bishop, ping` works through normal engine name matching
- `@Bishop ping` works
- `Did you see what @Bishop did?` does not trigger a command
- `/bishop ping` still stays hidden/private

### 9. If Ambient Subscription Creation Fails

Check:

- Chat app error logs:

```text
resource.type="chat.googleapis.com/Project"
severity=ERROR
```

- Workspace Events/API errors:

```text
resource.type="audited_resource"
protoPayload.serviceName="workspaceevents.googleapis.com"
severity>=WARNING
```

- Bishop's own `robot.log` warnings about:
  - subscription creation
  - subscription renewal
  - permission or scope failures

### 10. Only After Ambient Works, Revisit Visibility

Once all of the above works together:

- interactive events still work
- ambient events work
- no obvious regressions

Then decide whether to:

- add more test users to **Visibility**
- or move to the broader private-app rollout path for your Workspace

Do not widen visibility before the combined interactive + ambient path is proven stable.
