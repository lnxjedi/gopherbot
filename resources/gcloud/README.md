# Gopherbot Google Cloud Setup Guide

This guide will help you set up the necessary infrastructure on Google Cloud Platform (GCP) to run a Gopherbot instance using Firestore for its "brain" and Google Chat for communication.

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

Terraform will output a service account key. Save this as `gopherbot-key.json` in your robot's configuration directory.

From your robot's `custom/` directory, encrypt it into the default filename the Firestore brain expects, `gopherbot-key.json.enc`:

```bash
cd custom
gopherbot encrypt -f path/to/gopherbot-key.json > gopherbot-key.json.enc
rm path/to/gopherbot-key.json
```

If you keep that default filename in `custom/`, the Firestore brain can use it without any filename override. The Firestore brain and the future Google Chat connector can read `gopherbot-key.json.enc` directly through Gopherbot's encrypted-file support, so the plaintext key does not need to live on disk at runtime.

# Step 7: Manual Google Chat Configuration

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
    *   **Topic ID**: Use the topic ID created by Terraform (e.g., `projects/[PROJECT_ID]/topics/gopherbot-chat`).
    *   **Visibility**: Select "Make this Chat app available to specific people and groups in your Workspace domain" (or everyone, depending on your preference).
5.  Click **Save**.

## Step 8: Configure Domain-Wide Delegation (Workspace Admin)

To allow Gopherbot to read *all* messages in spaces (ambient traffic), it needs to use the Workspace Events API with the `chat.app.messages.readonly` scope. This requires Domain-Wide Delegation and Admin approval.

1.  Go to the [Google Workspace Admin Console](https://admin.google.com/).
2.  Navigate to **Security > Access and data control > API controls > Manage Domain Wide Delegation**.
3.  Click **Add new**.
4.  **Client ID**: Enter the `gopherbot_service_account_unique_id` from the Terraform output.
5.  **OAuth scopes**: Enter `https://www.googleapis.com/auth/chat.app.messages.readonly`
6.  Click **Authorize**.

*(Note: Your Gopherbot code must also be updated to use the Workspace Events API to create subscriptions for the spaces it joins.)*

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

For the future Google Chat connector, keep the Pub/Sub subscription ID created by Terraform (`projects/[PROJECT_ID]/subscriptions/gopherbot-chat-sub`).

---

*Generated by Gopherbot GCloud Setup Assistant*
