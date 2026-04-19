# Gopherbot Google Cloud New Robot Setup

This is the canonical setup path for a brand-new Google Chat-connected
Gopherbot robot.

The goal is to finish with:

- a new GCP project
- Firestore, Pub/Sub, IAM, and service-account resources in place
- a working Chat app for DM, `@mention`, and slash-command use
- full access to all messages in invited spaces enabled through Marketplace publish + Workspace admin install
- a `gopherbot-key.json.enc` file stored with the robot

## Best Google Docs

The two most useful Google docs for this setup are:

- initial Chat app setup:
  https://developers.google.com/workspace/chat/quickstart/pub-sub
- Chat app auth and admin approval:
  https://developers.google.com/workspace/chat/authenticate-authorize-chat-app

## What You Need Ready

Before you start, have these ready:

- Google Cloud Shell access authenticated as yourself in the correct Workspace
  organization
- enough Google Cloud permissions to create a project and manage APIs/IAM in it
- a Google Workspace admin who can do the final **Admin install** if that is
  not you
- a square robot avatar image available locally
- a public HTTPS URL for that avatar image
- a public HTTPS README or similar documentation URL
- Marketplace listing assets Google requires:
  - icon images
  - banner image
  - at least one screenshot

Practical note:

- a single `512x512` master avatar image is a good source asset, and for a
  simple internal setup it is fine to reuse the same image for the required
  Marketplace uploads

## Files In This Directory

- [`gcloud.env.example`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/gcloud.env.example)
  is the local env template
- [`scripts/create-project.sh`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/scripts/create-project.sh)
  creates the GCP project or re-selects it if it already exists
- [`scripts/enable-project-services.sh`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/scripts/enable-project-services.sh)
  enables the required APIs
- [`scripts/create-project-resources.sh`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/scripts/create-project-resources.sh)
  creates Firestore, Pub/Sub, service-account, and IAM resources
- [`scripts/create-service-account-key.sh`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/scripts/create-service-account-key.sh)
  creates the plaintext JSON key so you can immediately encrypt it

## Step 1: Open Cloud Shell Editor

In Google Cloud:

1. open Cloud Shell
2. switch to the **Editor** view
3. make sure you are authenticated as yourself in the correct Workspace org

## Step 2: Shallow-Clone Gopherbot

In the Cloud Shell terminal:

```bash
git clone --depth 1 https://github.com/lnxjedi/gopherbot.git
cd gopherbot/resources/gcloud
```

This setup flow does not need repository history, so a shallow clone is faster
and keeps the Cloud Shell workspace lighter.

## Step 3: Create `gcloud.env`

Copy the example file:

```bash
cp gcloud.env.example gcloud.env
```

Edit `gcloud.env` in the Cloud Shell Editor.

Important values:

- `PROJECT_ID`
- `REGION`
- `TOPIC_ID`
- `SUBSCRIPTION_ID`
- `SERVICE_ACCOUNT_ID`
- `SERVICE_ACCOUNT_KEY_JSON`
- `ROBOT_NAME`

Then source it:

```bash
source ./gcloud.env
```

Re-run that command any time you change the file or open a new shell tab.

## Step 4: Create The GCP Project

Create the project from the terminal:

```bash
./scripts/create-project.sh
```

If the project already exists and you intend to reuse it, the script just
selects it and continues cleanly.

After project creation, make sure the new project is selected in the Cloud
Console project picker as well. That keeps new shell tabs and UI actions aimed
at the right project.

## Step 5: Verify Billing In The Web UI

Do this in the Google Cloud web UI before you enable APIs or create resources.

Open **Billing** for the new project and confirm both of these are true:

- the project is linked to a billing account
- the linked billing account is active and in good standing

Do not continue until billing is correct.

## Step 6: Enable Project APIs

Run:

```bash
./scripts/enable-project-services.sh
```

This enables the APIs the helper scripts and connector need for the project
setup path:

- Firestore
- Google Chat API
- Pub/Sub
- Workspace Events API

## Step 7: Create Firestore, Pub/Sub, Service Account, And IAM Resources

Run:

```bash
./scripts/create-project-resources.sh
```

This creates or verifies:

- Firestore database
- Pub/Sub topic
- Pub/Sub pull subscription
- service account
- Pub/Sub publisher permission for `chat-api-push@system.gserviceaccount.com`
- IAM for the robot service account:
  - project-level `roles/datastore.user`
  - subscription-level `roles/pubsub.subscriber`
  - subscription-level `roles/pubsub.viewer`

Important note:

- very new projects can be slightly flaky right after creation or right after
  billing/API changes
- the script retries the create/bind steps most likely to fail during that
  propagation window, including the first service-account/IAM updates
- if it still fails, wait a minute and rerun it

## Step 8: Create The Service-Account Key And Encrypt It

Create the plaintext key:

```bash
./scripts/create-service-account-key.sh
```

That writes the JSON key to `SERVICE_ACCOUNT_KEY_JSON`.

Immediately encrypt it into your robot's `custom/` directory:

```bash
gopherbot encrypt -f "${SERVICE_ACCOUNT_KEY_JSON}" \
  > /path/to/robot/custom/gopherbot-key.json.enc
```

Then remove the plaintext JSON file.

The intended end state is:

- keep `gopherbot-key.json.enc`
- do not keep the plaintext `gopherbot-key.json`

## Step 9: Point The Robot At The Encrypted Credentials

For a Firestore brain, the usual minimal config is:

```yaml
BrainConfig:
  ProjectID: "your-gcp-project-id"
  DatabaseID: "(default)"
  Collection: "gopherbot-brain"
  CredentialsEncryptedFile: "gopherbot-key.json.enc"
```

For the Google Chat protocol, keep the connector pointed at the same project,
subscription, and encrypted credential file.

The sample config remains:

- [`conf/protocols/googlechat.yaml.sample`](/home/david/git/gopherbot-work/gopherbot/conf/protocols/googlechat.yaml.sample)

## Step 10: Configure The Google Chat API

Open:

- **APIs & Services > Enabled APIs & services > Google Chat API > Configuration**

If the UI shows **Build this Chat app as a Google Workspace add-on**, disable
that first.

Use values like these:

- **App name**: your robot name
  Keeping the app name close to the robot name makes `@mention` behavior more
  intuitive, but it does not have to be an exact match. If you have had false
  starts with an earlier app name, choosing a fresh app name can be the safer
  choice.
- **Avatar URL**: the public HTTPS URL for the avatar image
- **Description**: short and internal
- **Functionality**:
  - enable **Join spaces and group conversations**
- **Connection settings**: **Cloud Pub/Sub**
- **Topic ID**: `projects/${PROJECT_ID}/topics/${TOPIC_ID}` (this value was output during resource creation)
- **Commands**: add your slash command, for example `/bishop`
- **Visibility**: clear the checkbox for making the app visible to specific users - this app will be shared with all users via admin install
- **Logs**: enable **Log errors to Logging**

Now save the configuration.

## Step 11: Prepare The Service Account For Chat App Authorization

Ambient message support requires app-auth scopes, which means Google wants a
Marketplace-compatible OAuth client tied to the service account.

Open:

- **IAM & Admin > Service Accounts**
- open the robot service account
- open **Advanced settings**

Then create:

- **Google Workspace Marketplace-compatible OAuth client**

If Google blocks that step and demands branding or consent-screen setup first:

1. open **Google Auth Platform > Branding**
2. fill in the minimum required fields
3. choose **Internal**
4. save
5. go back and create the Marketplace-compatible OAuth client

This is a Google platform prerequisite for `chat.app.*` approval. It is not the
normal runtime auth model for the bot.

## Step 12: Configure Marketplace SDK For Ambient Messages

Open:

- **APIs & Services > Enabled APIs & services > Google Workspace Marketplace SDK**

If the SDK is not enabled yet, enable it first, then open **App Configuration**.

Use this shape:

- **App visibility**: `Private`
- **Installation settings**: `Individual + Admin Install`
- **App integrations**: `Chat app`
- Add to **OAuth scopes**:
  - `https://www.googleapis.com/auth/chat.app.messages.readonly`

Important notes:

- do **not** add `https://www.googleapis.com/auth/chat.bot` in Marketplace SDK
- for Gopherbot's current ambient message-created flow, `chat.app.spaces` and
  `chat.app.memberships` are not needed

Save the draft when App Configuration is complete.

## Step 13: Fill The Store Listing And Publish Privately

Open the **Store Listing** tab in Marketplace SDK.

Fill the required sections:

- App Details
- Graphic Assets
- Screenshots
- Support Links

Google currently requires:

- icon images
- banner image
- at least one screenshot
- Terms of service URL
- Privacy policy URL
- Support URL

Practical note:

- it is fine to reuse the same local image for the required uploads during an
  internal setup
- it is also fine to use a public README URL for the required listing links
  while you are still iterating

Publish the app as a private Marketplace app for your organization.

After private publish, the app should appear under **Internal Apps** for the
organization.

## Step 14: Have A Workspace Admin Do The Admin Install

This step matters. `Individual install` is not enough for the app-auth scope
approval path.

In the Google Admin console:

1. go to **Apps > Google Workspace Marketplace apps > Apps list**
2. click **Install app**
3. choose the newly published private app
4. choose **Admin install**
5. review the data access requirements
6. install for the intended users

This is the point where the one-time administrator approval for
`chat.app.messages.readonly` effectively happens.

## Step 16: Configure Google Chat and Start Your Robot

In the robot's Google Chat protocol config:

```yaml
ProtocolConfig:
  ProjectID: <your project ID>
  SlashCommand: <your robot's slash command> (e.g. bishop)
```

Start the robot and watch the logs.

You want to see:

- ambient subscription creation for joined spaces
- no new permission errors
- DM, `@mention`, and slash-command behavior working

## Step 17: Test Ambient Behavior

In a joined space, test all of these:

1. a plain message not addressed to the bot
2. `Bishop, ping`
3. `@Bishop ping`
4. `Did you see what @Bishop did?`
5. `/bishop ping`

Expected behavior:

- ambient messages are received
- `Bishop, ping` works
- `@Bishop ping` works
- mid-sentence mentions do not become commands
- slash-command replies remain private

## Troubleshooting

- If resource creation fails right after project creation, billing setup, or
  API enablement, wait a minute and rerun the script.
- If ambient subscription creation fails with a 403 about missing approval, the
  app probably was not truly **Admin installed** yet.
- If the app appears installed only for your user, that is not enough for the
  ambient `chat.app.*` flow.
- If Google makes you configure Branding first, choose **Internal**.
- If Pub/Sub topic creation is blocked by org policy, set an explicit
  `PUBSUB_ALLOWED_REGIONS` value in `gcloud.env`.

## Reference Docs

- Chat app auth and admin approval:
  https://developers.google.com/workspace/chat/authenticate-authorize-chat-app
- Chat Pub/Sub quickstart:
  https://developers.google.com/workspace/chat/quickstart/pub-sub
- Configure the Chat API:
  https://developers.google.com/workspace/chat/configure-chat-api
- Create Workspace Events subscriptions:
  https://developers.google.com/workspace/events/guides/create-subscription
- Chat events overview:
  https://developers.google.com/workspace/events/guides/events-chat
- Configure Marketplace SDK:
  https://developers.google.com/workspace/marketplace/enable-configure-sdk
- Marketplace store listing requirements:
  https://developers.google.com/workspace/marketplace/create-listing
- Publish a private Marketplace app:
  https://developers.google.com/workspace/marketplace/how-to-publish
- Admin install Marketplace apps:
  https://knowledge.workspace.google.com/admin/apps/install-marketplace-apps-for-your-organization
- Set up Chat app authorization:
  https://knowledge.workspace.google.com/admin/chat/set-up-app-authorization-for-chat
