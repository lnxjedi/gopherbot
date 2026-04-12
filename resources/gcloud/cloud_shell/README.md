# Google Chat Clean-Room Test From Cloud Shell

This guide creates a fresh Google Cloud project for a minimal Google Chat experiment.

The goal is to prove the simple interactive path first:

- direct messages
- `@mention` commands
- `/bishop` slash commands

This guide intentionally does **not** set up ambient message access, Marketplace publication, or Workspace Events subscriptions. We add those only after the basic interactive path works.

If the minimal setup works but the full setup breaks later, we have a strong signal that the Marketplace/admin-install/ambient layer is the culprit.

## Phase 1 Goal

At the end of this phase, Bishop should be able to:

- respond to a DM like `hello`
- respond to `@Bishop Gopherbot ping`
- respond to `/bishop ping`

Do **not** continue to ambient-message setup until all three work.

## Prerequisites

- A Google Workspace account that can use Google Chat.
- Permission to create a new Google Cloud project.
- Billing enabled for the new project before enabling APIs.
- A local checkout of the Gopherbot repo somewhere you can later copy the service-account key from.

## 1. Open Cloud Shell

In Google Cloud Console, open **Cloud Shell**.

## 2. Set Variables

Pick a fresh project ID and keep the rest of the names simple:

```bash
export PROJECT_ID="bishop-chat-test-1"
export PROJECT_NAME="Bishop Chat Test"
export REGION="us-central1"
export TOPIC_ID="gopherbot-chat"
export SUBSCRIPTION_ID="gopherbot-chat-sub"
export DEBUG_SUBSCRIPTION_ID="gopherbot-chat-debug-sub"
export SERVICE_ACCOUNT_ID="gopherbot-robot"
export SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_ID}@${PROJECT_ID}.iam.gserviceaccount.com"
```

If your first project ID is already taken, change `PROJECT_ID` and rerun the exports.

## 3. Create the Project

```bash
gcloud projects create "${PROJECT_ID}" --name="${PROJECT_NAME}"
gcloud config set project "${PROJECT_ID}"
```

If the new project does not already have billing enabled, link billing in the Console before continuing.

## 4. Enable Only the Base APIs

For this control experiment, enable only the APIs needed for the interactive Chat app path:

```bash
gcloud services enable \
  chat.googleapis.com \
  pubsub.googleapis.com \
  iam.googleapis.com \
  cloudresourcemanager.googleapis.com
```

Do **not** enable `workspaceevents.googleapis.com` yet.

## 5. Create Pub/Sub Resources

Create the topic, the robot's pull subscription, and a temporary debug subscription:

```bash
gcloud pubsub topics create "${TOPIC_ID}"

gcloud pubsub subscriptions create "${SUBSCRIPTION_ID}" \
  --topic="${TOPIC_ID}" \
  --expiration-period=never

gcloud pubsub subscriptions create "${DEBUG_SUBSCRIPTION_ID}" \
  --topic="${TOPIC_ID}" \
  --expiration-period=never
```

Grant Google Chat permission to publish to the topic:

```bash
gcloud pubsub topics add-iam-policy-binding "${TOPIC_ID}" \
  --member="serviceAccount:chat-api-push@system.gserviceaccount.com" \
  --role="roles/pubsub.publisher"
```

## 6. Create the Robot Service Account

```bash
gcloud iam service-accounts create "${SERVICE_ACCOUNT_ID}" \
  --display-name="Gopherbot Robot Service Account"
```

Grant the robot the Pub/Sub permissions the connector expects:

```bash
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/pubsub.subscriber"

gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${SERVICE_ACCOUNT_EMAIL}" \
  --role="roles/pubsub.viewer"
```

## 7. Create the Service Account Key

Save the key into the current Cloud Shell directory:

```bash
gcloud iam service-accounts keys create gopherbot-key.json \
  --iam-account="${SERVICE_ACCOUNT_EMAIL}"
```

Later, encrypt this into `gopherbot-key.json.enc` in Bishop's `custom/` directory using:

```bash
gopherbot encrypt -f path/to/gopherbot-key.json > gopherbot-key.json.enc
```

## 8. Record the Values Bishop Will Need

These are the values to carry into Bishop's Google Chat config:

```bash
echo "PROJECT_ID=${PROJECT_ID}"
echo "TOPIC=projects/${PROJECT_ID}/topics/${TOPIC_ID}"
echo "SUBSCRIPTION=projects/${PROJECT_ID}/subscriptions/${SUBSCRIPTION_ID}"
echo "SERVICE_ACCOUNT=${SERVICE_ACCOUNT_EMAIL}"
```

## 9. Configure the Google Chat App In The UI

Open:

- **APIs & Services > Enabled APIs & services > Google Chat API > Configuration**

Configure the Chat app like this:

- **App name**: `Bishop Gopherbot` or another clear test name
- **Description**: anything simple
- **Functionality**:
  - enable **Receive 1:1 messages**
  - enable **Join spaces and group conversations**
- **Interactive features**: enabled
- **Connection settings**: **Cloud Pub/Sub**
- **Topic ID**: `projects/${PROJECT_ID}/topics/${TOPIC_ID}`
- **Commands**:
  - add `/bishop`
  - command ID `1`
- **Visibility**:
  - make the app available only to your own Workspace email for this test
- **Logs**:
  - enable **Log errors to Logging**

Important:

- If the UI shows a control like **Build this Chat app as a Google Workspace add-on**, turn it off for this phase. The official Pub/Sub quickstart expects a Pub/Sub Chat app, not the add-on trigger flow.
- Do **not** configure Marketplace SDK, admin install, or `chat.app.*` scopes yet.
- Do **not** enable ambient-message setup yet.

After saving, wait a few minutes for the config to propagate.

## 10. Point Bishop At The New Project

Update Bishop's Google Chat config to use only the new interactive-test project.

Suggested config shape:

```yaml
ProtocolConfig:
  ProjectID: "YOUR_NEW_PROJECT_ID"
  SubscriptionID: "projects/YOUR_NEW_PROJECT_ID/subscriptions/gopherbot-chat-sub"
  CredentialsEncryptedFile: "gopherbot-key.json.enc"
  AmbientMessages: false
  ThreadResponses: true
  SlashCommand: bishop
  UserMap:
    yourusername: users/YOUR_USER_ID
```

Notes:

- Keep `AmbientMessages: false` for this phase.
- Do not add Bishop's bot ID yet unless you discover you need it.

## 11. Test The Interactive Path

Restart Bishop, then test in this order:

1. Open a DM with the test app and send `hello`
2. In a space, send `@Bishop Gopherbot ping`
3. In a space, send `/bishop ping`

With the current connector logging, Bishop should log something for every Pub/Sub delivery it receives.

## 12. Use The Debug Subscription To Inspect Raw Deliveries

If something fails, pull from the debug subscription immediately after the test action:

```bash
gcloud pubsub subscriptions pull \
  "projects/${PROJECT_ID}/subscriptions/${DEBUG_SUBSCRIPTION_ID}" \
  --limit=10 \
  --auto-ack \
  --format=json
```

Interpretation:

- If you see plain Chat interaction JSON, the interactive path is publishing correctly.
- If you see only `ce-type=google.workspace.chat.message.v1.created`, you are only seeing ambient-style Workspace Events payloads.
- If you see nothing for a DM or `/bishop ping`, Google did not publish that interaction event to the topic.

## 13. Stop Here If Interactive Events Fail

Do **not** continue to Marketplace/admin-install/ambient setup if DMs, mentions, and slash commands are not all working in this clean project.

This phase is the control experiment.

## 14. Only After Phase 1 Passes

Once the minimal interactive path works, move on to the fuller setup in:

- [../README.md](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/README.md)

At that point, add:

- `workspaceevents.googleapis.com`
- Marketplace SDK configuration
- admin install for `chat.app.*` scopes
- ambient Workspace Events subscriptions

Then re-test the same three interactive behaviors again before trusting the ambient setup.
