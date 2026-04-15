# Gopherbot Google Cloud Setup

This is the preferred path for setting up a new Gopherbot robot on Google
Cloud.

The goal is to get all of the project resources in place with `gcloud`, then use
the Google UI only for the Chat app, Marketplace SDK, and admin-install steps
that cannot realistically be automated away.

The intended sequence is:

1. set up the Chat API and Pub/Sub interaction path first
2. create the robot resources with `gcloud`
3. configure the Chat app
4. add the Marketplace/admin-install layer for ambient messages

This README is the canonical setup path. Older notes remain in this directory
for reference while the process continues to settle down.

## Best Google Documents

The single best document for initial Chat app setup:

- https://developers.google.com/workspace/chat/quickstart/pub-sub

The single most useful Google document for the Chat app auth/admin-approval
path appears to be:

- https://developers.google.com/workspace/chat/authenticate-authorize-chat-app

There is not currently a single Google document that cleanly covers:

- Pub/Sub interaction events
- Firestore brain credentials
- Marketplace-compatible OAuth client setup
- private Marketplace publication
- admin install
- Workspace Events ambient subscriptions

So this guide stitches together the closest official docs plus the sharp edges
we hit while bringing a real robot online.

If you get stuck on the interactive baseline, keep
[cloud_shell/README.md](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/cloud_shell/README.md)
around as older troubleshooting/reference material.

## Directory Layout

- [`gcloud.env.example`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/gcloud.env.example):
  editable environment template for your project and robot
- [`scripts/enable-project-services.sh`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/scripts/enable-project-services.sh):
  enable all required GCP APIs in one `gcloud services enable` command
- [`scripts/create-project-resources.sh`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/scripts/create-project-resources.sh):
  create Firestore, Pub/Sub, service-account, and IAM resources
- [`scripts/create-service-account-key.sh`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/scripts/create-service-account-key.sh):
  create the JSON key that you will then encrypt for the robot
- [`cloud_shell/`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/cloud_shell):
  older field notes and sample/control material, kept for reference
- [`terraform/`](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/terraform):
  older infrastructure reference, kept for comparison only; it is not the
  preferred path now

## Step 1: Open The Cloud Editor

In Google Cloud:

1. open your target GCP project
2. open **Cloud Shell**
3. switch to the **Editor** view

The Editor view matters because users can edit files there, not just paste
commands into a terminal.

## Step 2: Clone Gopherbot

In the Cloud Editor terminal:

```bash
git clone https://github.com/lnxjedi/gopherbot.git
cd gopherbot/resources/gcloud
```

## Step 3: Create And Edit `gcloud.env`

Copy the example file:

```bash
cp gcloud.env.example gcloud.env
```

Edit `gcloud.env` in the Cloud Editor.

Important values:

- `PROJECT_ID`
- `REGION`
- `TOPIC_ID`
- `SUBSCRIPTION_ID`
- `SERVICE_ACCOUNT_ID`
- `SERVICE_ACCOUNT_KEY_JSON`
- `WORKSPACE_TEST_USER`
- `ROBOT_NAME`
- `ROBOT_SLASH_COMMAND`

Then source it:

```bash
source ./gcloud.env
```

Re-run that `source` command any time you change the file or open a new shell
tab.

## Step 4: Create Project and Enable Project Services

> USER NOTES: split into two scripts; initialize project and create services; projects create and config set in first script

> USER NOTES: After creating the project, the user should be sure the project is selected in the upper-left, to ensure any new terminal windows opened use the new project

Run:

```bash
./scripts/enable-project-services.sh
```

This coalesces the required APIs into a single `gcloud services enable`
command:

- Cloud Resource Manager
- IAM
- Firestore
- Google Chat API
- Pub/Sub
- Workspace Events API
- Google Workspace Marketplace SDK support

Note:

- during Bishop setup, we explicitly used
  `gcloud services enable appsmarket-component.googleapis.com`
- that API enable is already included in the script

## Step 4a: Set up billing for the project
Follow [these directions](https://docs.cloud.google.com/billing/docs/how-to/verify-billing-enabled) to ensure your project is linked to a billing account.

## Step 5: Create The Project Resources

Run:

```bash
./scripts/create-project-resources.sh
```

This script creates or verifies:

- Firestore database
- Pub/Sub topic
- Pub/Sub pull subscription
- service account
- Pub/Sub publisher permission for `chat-api-push@system.gserviceaccount.com`
- robot project roles:
  - `roles/datastore.user`
  - `roles/pubsub.subscriber`
  - `roles/pubsub.viewer`

> USER NOTES: pause after creating service account and check for it's existence - the script errored out the first time when trying to update the service account

Expected resource names are controlled by `gcloud.env`.

If your organization policy blocks Pub/Sub topic creation unless a storage
region is explicit, set `PUBSUB_ALLOWED_REGIONS` in `gcloud.env`. Using the
same region as Firestore is fine for a simple setup.

## Step 6: Create And Encrypt The Service Account Key

Run:

```bash
./scripts/create-service-account-key.sh
```

That writes the plaintext JSON key to `SERVICE_ACCOUNT_KEY_JSON`.

Then encrypt it from your robot's `custom/` directory:

```bash
gopherbot encrypt -f /path/to/gopherbot-key.json > gopherbot-key.json.enc
```

If you use the default filename `gopherbot-key.json.enc`, both the Firestore
brain and the Google Chat connector can use it without extra filename
overrides.

After encrypting it, remove the plaintext JSON.

## Step 7: Configure The Firestore Brain

If you are using the Firestore brain, the usual minimal config is:

```yaml
BrainConfig:
  ProjectID: "your-gcp-project-id"
  DatabaseID: "(default)"
  Collection: "gopherbot-brain"
  CredentialsEncryptedFile: "gopherbot-key.json.enc"
```

Add this content to your robot's `conf/brains/firestore.yaml`. If the credential file keeps the default name, usually the only thing you need to override is `ProjectID`.

## Step 8: Configure The Google Chat API

Open:

- **APIs & Services > Enabled APIs & services > Google Chat API > Configuration**

Clear **Build this Chat app as a Google Workspace add-on**. A dialog opens asking you to confirm. In the dialog, click **Disable**.

Recommended values:

- **App name**: your robot name, for example `Bishop` - this is important for your robot's `@mention` to match the robot's name
- **Avatar**: add a square image to your robot's repository, provide a URL for it
- **Description**: something short and internal
- **Interactive Features**: turned on
- **Functionality**:
  - enable **Join spaces and group conversations**
- **Connection settings**: **Cloud Pub/Sub**
- **Topic ID**:
  `projects/${PROJECT_ID}/topics/${TOPIC_ID}` - or use `gcloud pubsub topics list`
- **Visibility**: start with only specific users/groups while testing
- **Logs**: enable **Log errors to Logging**

Slash commands:

- add the slash command you want, for example `/bishop`

## Step 9: Verify The Interactive Baseline

Add `conf/protocols/googlechat.yaml`, example:
```yaml
ProtocolConfig:
  ProjectID: <your-project-id>
  SubscriptionID: "gopherbot-chat-sub"
  CredentialsEncryptedFile: gopherbot-key.json.enc
  # When true, the connector creates and maintains per-space Google Workspace
  # Events subscriptions so it can receive ambient message traffic in spaces
  # where the app has been added. Requires admin-approved chat.app.* scopes.
  AmbientMessages: false
  ThreadResponses: false
  # Set this to the slash command name configured in the Google Chat API,
  # without or with the leading slash. It is used for hidden-command help and
  # fallback rendering, and enables hidden command support in the connector.
  SlashCommand: bishop
```

Before touching ambient message setup, verify all of these work:

- DM the bot: `ping`
- mention the bot: `@Bishop help`
- slash command: `/bishop ping`

Do not move on until those are working.

## Step 10: Ambient Message Setup

Ambient message capture is the part Google makes awkward.

The best article for this:

- https://developers.google.com/workspace/chat/authenticate-authorize-chat-app

### 10a. Prepare The Service Account For Chat App Scopes

Open:

- **IAM & Admin > Service Accounts**
- open the robot service account created by the script
- open **Advanced settings**

Before creating the Marketplace-compatible OAuth client, Google may require an
OAuth consent screen / branding config to exist, even though it won't be used.

If Google blocks you there:

1. open **Google Auth Platform > Branding** or the older consent-screen UI
2. fill in the minimum required fields
3. choose **Internal** / organization-only visibility
4. save

This is confusing but expected. It is a prerequisite for the
Marketplace-compatible OAuth client, not the runtime auth model for the bot.

Then:

1. create a **Google Workspace Marketplace-compatible OAuth client** for the
   service account

### 10b. Open The Marketplace SDK

Open:

- **APIs & Services > Enabled APIs & services**
- open **Google Workspace Marketplace SDK**
- open **App Configuration**

### 10c. Configure App Auth Scopes

For Bishop's current ambient implementation, add:

- `https://www.googleapis.com/auth/chat.app.messages.readonly`

Notes:

- do **not** add `chat.bot` there; Google Chat already includes it
- `chat.app.spaces` and `chat.app.memberships` are not needed for the current
  message-created ambient flow

### 10d. Save Draft, Fill Store Listing, Publish

In practice, the flow we observed is:

1. configure the app
2. click **Save draft**
3. switch tab to **Store Listing** and fill in the blanks
    * Upload the same image for all the required images, it doesn't seem picky; this was tested with a 512px square png
    * Similarly for required URLs, you can use a link to the README for your robot
4. click **Save draft** again if needed
5. once the listing is complete, **Publish** becomes available
6. click **Publish**

Practical notes:

- Google requires app listing images
- prepare a square avatar image ahead of time
- for a simple internal setup, it is fine to upload the same avatar image for
  all required image fields
- for required URLs such as privacy/support, it is fine to use a robot README
  or similar internal documentation page while you are still iterating
- for **Distribution**, `All regions` is fine and the least surprising choice
  for an internal app unless you have a reason to restrict it

After publishing, the app should appear under **Internal Apps** in the
Workspace admin area.

### 10e. Admin Install

This step is easy to get wrong.

Do **not** rely on the generic `Install` button from the listing if it is
offering an individual/user install.

Instead:

1. open the app card/details page
2. choose **Admin Install**
3. not **Individual Install**

This is where the one-time administrator approval for `chat.app.*` scopes
effectively happens.

The Google admin help page that appeared relevant during Bishop setup is:

- https://knowledge.workspace.google.com/admin/chat/set-up-app-authorization-for-chat

### 10f. Turn On Ambient Messages In The Robot

In the robot's Google Chat protocol config:

```yaml
ProtocolConfig:
  AmbientMessages: true
```

Keep the working interactive values the same:

- `ProjectID`
- `SubscriptionID`
- `CredentialsEncryptedFile`
- `UserMap`
- `SlashCommand`

## Step 11: Restart And Test Ambient Behavior

Restart the robot and watch the logs.

You want to see:

- ambient subscription creation for joined spaces
- no new permission errors
- interactive behavior still working

Then test in a joined space:

1. plain message not addressed to the bot
2. `Bishop, ping`
3. `@Bishop ping`
4. `Did you see what @Bishop did?`
5. `/bishop ping`

Expected behavior:

- ambient messages are seen, but not all treated as bot-addressed commands
- `Bishop, ping` works
- `@Bishop ping` works
- mid-sentence mentions do not trigger commands
- slash commands remain private

## Troubleshooting Notes

- If the interactive baseline does not work, go back to
  [cloud_shell/README.md](/home/david/git/gopherbot-work/gopherbot/resources/gcloud/cloud_shell/README.md)
  and use the older sample/control notes to prove the basic Chat path first.
- If ambient subscription creation fails with a 403 about missing admin scope
  approval, the app was probably not truly **Admin Installed** yet.
- If the app appears installed only for your user, that is not enough for the
  `chat.app.*` ambient flow.
- If Google makes you configure a consent screen first, choose **Internal**.
- If topic creation is blocked by org policy, use an explicit message storage
  region.

## Status

This setup is still being refined, but the intended shape is now:

- `resources/gcloud/README.md` is the canonical setup path
- `resources/gcloud/scripts/` is the runnable helper layer
- `resources/gcloud/cloud_shell/` remains reference-only until we fully replace
  it
