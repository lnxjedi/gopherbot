# Google Chat Pub/Sub Quickstart Control

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
export PROJECT_ID="gopherbot-chatapi-quickstart"
export PROJECT_NAME="gopherbot-chatapi-quickstart"
export REGION="us-central1"
export TOPIC_ID="quickstart-chat-topic"
export SUBSCRIPTION_ID="quickstart-chat-sub"
export SERVICE_ACCOUNT_ID="quickstart-chat-sa"
export SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_ID}@${PROJECT_ID}.iam.gserviceaccount.com"
export WORKDIR="${HOME}/gopherbot-chatapi-quickstart"
```

If `gopherbot-chatapi-quickstart` is already taken as a project ID, add a suffix and rerun the exports, for example:

```bash
export PROJECT_ID="gopherbot-chatapi-quickstart-01"
export SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_ID}@${PROJECT_ID}.iam.gserviceaccount.com"
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
  pubsub.googleapis.com \
  iam.googleapis.com \
  cloudresourcemanager.googleapis.com
```

### Actual command (from quickstart)
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

> NOTE: Not used
```bash
gcloud pubsub topics add-iam-policy-binding "${TOPIC_ID}" \
  --member="serviceAccount:chat-api-push@system.gserviceaccount.com" \
  --role="roles/pubsub.publisher"
```

#### Actual command (from quickstart)
```bash
gcloud projects add-iam-policy-binding $PROJECT_ID --member=serviceAccount:chat-api-push@system.gserviceaccount.com --role=roles/pubsub.publisher
```

### 3. Create A Service Account

```bash
gcloud iam service-accounts create "${SERVICE_ACCOUNT_ID}" \
  --display-name="Quickstart Google Chat Service Account"
```

Create the key file in the current directory:

```bash
gcloud iam service-accounts keys create "${WORKDIR}/quickstart-key.json" \
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

## Write The Script

The quickstart's Python section says to:

1. provide service account credentials
2. provide the project ID
3. provide the subscription ID
4. create `requirements.txt`
5. create `app.py`

### 1. Provide Service Account Credentials

```bash
export GOOGLE_APPLICATION_CREDENTIALS="${WORKDIR}/quickstart-key.json"
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

## Clean Up

When you are done, delete the whole project:

```bash
gcloud projects delete "${PROJECT_ID}"
```
