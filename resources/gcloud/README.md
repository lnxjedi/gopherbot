# Gopherbot Google Cloud Setup Guide

This guide sets up the Google Cloud resources Gopherbot needs for a Firestore brain and a Google Chat connector.

The Google Cloud part is only half of the Google Chat setup. Terraform can create the project-level resources, service account, and Pub/Sub plumbing, but ambient message access in Chat also requires manual Google Workspace administrator approval for Chat app scopes. Once that approval is in place, the connector can manage per-space Workspace Events subscriptions itself after the app is invited to spaces.

## Prerequisites

1.  **Google Workspace Account**: You need a Google Workspace account (e.g., `you@yourcompany.com`). Personal `@gmail.com` accounts have limited support for Google Chat bots.
2.  **GCP Project Permissions**: You need permissions to create projects or at least `Owner` permissions on an existing project.
3.  **Google Cloud Shell**: We will use the Cloud Shell for a consistent environment with `git` and `terraform` pre-installed.

## Step 1: Create or Select a GCP Project

1.  Go to the [GCP Console](https://console.cloud.google.com/).
2.  If you don't have a project, click the project selector at the top and select **New Project**.
3.  Note your **Project ID**.

## Step 2: Open Google Cloud Shell

Click the **Activate Cloud Shell** icon ( `>_` ) in the top right of the GCP Console.

## Step 3: Clone the Gopherbot Repository

In the Cloud Shell, run:

```bash
git clone https://github.com/lnxjedi/gopherbot.git
cd gopherbot/resources/gcloud/terraform
```

*(Note: If the directory doesn't exist in the repo yet, you can create it and use the Terraform files provided below.)*

## Step 4: Enable Base APIs

Before Terraform can manage your project, it needs the Resource Manager and IAM APIs enabled so it can enable other APIs and create service accounts. Run this in your Cloud Shell (replace `YOUR_PROJECT_ID` with your actual project ID):

```bash
gcloud config set project YOUR_PROJECT_ID
gcloud services enable cloudresourcemanager.googleapis.com iam.googleapis.com
```

## Step 5: Configure and Run Terraform

1.  Create a `terraform.tfvars` file or be prepared to enter variables. Example `terraform.tfvars`:
    ```hcl
    project_id = "your-gcp-project-id"
    region     = "us-central1"
    ```
2.  Initialize Terraform:
    ```bash
    terraform init
    ```
3.  Apply the configuration:
    ```bash
    terraform apply
    ```
    You will be prompted for your `project_id` and `region`.

## Step 6: Save Credentials

Terraform outputs a service account key. Save it as `gopherbot-key.json`, then encrypt it from your robot's `custom/` directory into the default runtime filename `gopherbot-key.json.enc`:

```bash
cd custom
gopherbot encrypt -f path/to/gopherbot-key.json > gopherbot-key.json.enc
rm path/to/gopherbot-key.json
```

If you keep that default filename in `custom/`, both the Firestore brain and the Google Chat connector can use it without any filename override. Gopherbot reads `gopherbot-key.json.enc` directly through its encrypted-file support, so the plaintext key does not need to live on disk at runtime.

## Step 7: Manual Google Chat Configuration

Terraform enables the APIs, but some steps must be done manually in the UI:

1.  Go to **APIs & Services > Enabled APIs & services**.
2.  Search for **Google Chat API** and click it.
3.  Click the **Configuration** tab.
4.  Set the following:
    *   **App name**: Your Robot's Name (e.g., "Gopherbot")
    *   **Avatar URL**: (Optional) A link to an image.
    *   **Description**: "DevOps Chatbot"
    *   **Functionality**: Enable "Receive 1:1 messages" and "Join spaces and group conversations".
    *   **Connection settings**: Select **Cloud Pub/Sub**.
    *   **Topic ID**: Use the `chat_topic_id` output from Terraform (for example `projects/[PROJECT_ID]/topics/gopherbot-chat`).
    *   **Visibility**: Select "Make this Chat app available to specific people and groups in your Workspace domain" (or everyone, depending on your preference).
5.  Click **Save**.

## Step 8: Enable Chat App Admin Approval For Ambient Messages

To allow Gopherbot to read ambient traffic in spaces after it is invited, Google requires app authentication plus one-time administrator approval for `chat.app.*` scopes. This is not Domain-Wide Delegation.

Use the service account created by Terraform, identified by the `gopherbot_service_account_email` output.

1.  In Google Cloud Console, go to **IAM & Admin > Service Accounts** and open the Gopherbot service account.
2.  Under **Advanced settings**, create a **Google Workspace Marketplace-compatible OAuth client** for that service account.
3.  Enable the **Google Workspace Marketplace SDK** in the project.
4.  Open **Google Workspace Marketplace SDK > App Configuration**.
5.  Configure the app as a private Workspace app with Chat enabled.
6.  Add the Chat app scopes your connector needs. For ambient message capture, include:
    *   `https://www.googleapis.com/auth/chat.app.messages.readonly`
7.  Click **Save draft**.

For this flow, **Save draft** is expected, but it is not enough by itself to make the app show up for installation.

8.  In **Google Workspace Marketplace SDK > Store Listing**, fill in the required store-listing fields for the app.
9.  Publish the app as a **private** Marketplace app for your organization.

For a private app, this publish step makes the app available internally right away. You do not need a public Marketplace listing or public review.

10. Make sure the Chat app itself is enabled and saved in **Google Chat API > Configuration**.
11. In the Google Workspace Admin console, go to **Apps > Google Workspace Marketplace apps > Apps list**.
12. Click **Install app** and select your app.
13. Choose **Admin install**.
14. Review the app's data access requirements and complete the install.

For `chat.app.*` scopes, this **Admin install** step is where the one-time administrator approval happens. There is not a separate Domain-Wide Delegation approval step for `chat.app.messages.readonly`.

After the admin install is complete, Gopherbot can use the same encrypted service account key to create Workspace Events subscriptions for the spaces the app joins.

Important notes:

*   No Terraform-per-space setup is required.
*   No Domain-Wide Delegation step is required for `chat.app.messages.readonly`.
*   The connector should create and maintain Workspace Events subscriptions per space at runtime.

## Step 9: Configure Your Robot

Configure your robot to use the Firestore brain and point it at the encrypted service-account file:

```yaml
# conf/environments/production.yaml
PrimaryProtocol: ssh
DefaultProtocol: ssh
Brain: firestore
LogDest: stdout
```

```yaml
# conf/brains/firestore.yaml
BrainConfig:
  ProjectID: "your-gcp-project-id"
  DatabaseID: "(default)"
  Collection: "gopherbot-brain"
  CredentialsEncryptedFile: "gopherbot-key.json.enc"
```

With the default filename above, the only required override is usually `ProjectID`.

For the Google Chat connector, keep the Terraform outputs handy:

*   `gopherbot_service_account_email`: used during the Marketplace/admin-approval setup.
*   `chat_topic_id`: use this in the Google Chat API configuration page.
*   `chat_subscription_id`: this is the pull subscription Gopherbot reads from.

## Step 10: What Terraform Does And Does Not Do

Terraform creates:

*   the Firestore database
*   the Pub/Sub topic and pull subscription
*   the Gopherbot service account and key
*   the Pub/Sub publisher permission Chat needs to deliver events to your topic

Terraform does not create:

*   Google Workspace Marketplace SDK app configuration
*   one-time Workspace administrator approval for `chat.app.*` scopes
*   per-space Workspace Events subscriptions

Once the app has administrator-approved Chat app scopes, the connector can handle those per-space subscriptions automatically when the app joins spaces.

---

*Generated by Gopherbot GCloud Setup Assistant*
