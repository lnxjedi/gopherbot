import { useState } from "react";
import { 
  BookOpen, 
  Code2, 
  Terminal, 
  Copy, 
  Check, 
  ExternalLink, 
  Github, 
  Cloud, 
  Bot,
  Database,
  MessageSquare
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import Markdown from "react-markdown";
import { cn } from "@/src/lib/utils";

const README_CONTENT = `# Gopherbot Google Cloud Setup Guide

This guide will help you set up the necessary infrastructure on Google Cloud Platform (GCP) to run a Gopherbot instance using Firestore for its "brain" and Google Chat for communication.

## Prerequisites

1.  **Google Workspace Account**: You need a Google Workspace account (e.g., \`you@yourcompany.com\`). Personal \`@gmail.com\` accounts have limited support for Google Chat bots.
2.  **GCP Project Permissions**: You need permissions to create projects or at least \`Owner\` permissions on an existing project.
3.  **Google Cloud Shell**: We will use the Cloud Shell for a consistent environment with \`git\` and \`terraform\` pre-installed.

## Step 1: Create or Select a GCP Project

1.  Go to the [GCP Console](https://console.cloud.google.com/).
2.  If you don't have a project, click the project selector at the top and select **New Project**.
3.  Note your **Project ID**.

## Step 2: Open Google Cloud Shell

Click the **Activate Cloud Shell** icon ( \`>_\` ) in the top right of the GCP Console.

## Step 3: Clone the Gopherbot Repository

In the Cloud Shell, run:

\`\`\`bash
git clone https://github.com/lnxjedi/gopherbot.git
cd gopherbot/resources/gcloud
\`\`\`

*(Note: If the directory doesn't exist in the repo yet, you can create it and use the Terraform files provided below.)*

## Step 4: Configure and Run Terraform

1.  Create a \`terraform.tfvars\` file or be prepared to enter variables. Example \`terraform.tfvars\`:
    \`\`\`hcl
    project_id = "your-gcp-project-id"
    region     = "us-central1"
    \`\`\`
2.  Initialize Terraform:
    \`\`\`bash
    terraform init
    \`\`\`
3.  Apply the configuration:
    \`\`\`bash
    terraform apply
    \`\`\`
    You will be prompted for your \`project_id\` and \`region\`.

## Step 5: Save Credentials

Terraform will output a service account key. Save this as \`gopherbot-key.json\` in your robot's configuration directory. **Keep this file secure!**

## Step 6: Manual Google Chat Configuration

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
    *   **Topic ID**: Use the topic ID created by Terraform (e.g., \`projects/[PROJECT_ID]/topics/gopherbot-chat\`).
    *   **Visibility**: Select "Make this Chat app available to specific people and groups in your Workspace domain" (or everyone, depending on your preference).
5.  Click **Save**.

## Step 7: Start Your Robot

Configure your Gopherbot \`conf/gopherbot.yaml\` to use the \`googlechat\` connector and \`firestore\` brain, pointing to your credentials file and the Pub/Sub subscription ID (\`projects/[PROJECT_ID]/subscriptions/gopherbot-chat-sub\`).
`;

const TERRAFORM_MAIN = `terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Enable required APIs
resource "google_project_service" "firestore" {
  service            = "firestore.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "chat" {
  service            = "chat.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "pubsub" {
  service            = "pubsub.googleapis.com"
  disable_on_destroy = false
}

# Firestore Database (Native Mode)
resource "google_firestore_database" "database" {
  name                    = "(default)"
  location_id             = var.region
  type                    = "FIRESTORE_NATIVE"
  delete_protection_state = "DELETE_PROTECTION_DISABLED"

  depends_on = [google_project_service.firestore]
}

# Pub/Sub Topic for Google Chat
resource "google_pubsub_topic" "chat_topic" {
  name = "gopherbot-chat"

  depends_on = [google_project_service.pubsub]
}

# Pub/Sub Subscription for Gopherbot to pull messages
resource "google_pubsub_subscription" "chat_subscription" {
  name  = "gopherbot-chat-sub"
  topic = google_pubsub_topic.chat_topic.name

  # Never expire the subscription
  expiration_policy {
    ttl = ""
  }
}

# Service Account for the Robot
resource "google_service_account" "gopherbot" {
  account_id   = "gopherbot-robot"
  display_name = "Gopherbot Robot Service Account"
}

# IAM Roles for the Service Account
resource "google_project_iam_member" "firestore_owner" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:\${google_service_account.gopherbot.email}"
}

resource "google_project_iam_member" "pubsub_subscriber" {
  project = var.project_id
  role    = "roles/pubsub.subscriber"
  member  = "serviceAccount:\${google_service_account.gopherbot.email}"
}

# Allow Google Chat to publish to the topic
resource "google_pubsub_topic_iam_member" "chat_publisher" {
  topic  = google_pubsub_topic.chat_topic.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:chat-api-push@system.gserviceaccount.com"
}

# Service Account Key (for the credentials file)
resource "google_service_account_key" "gopherbot_key" {
  service_account_id = google_service_account.gopherbot.name
}
`;

const TERRAFORM_VARS = `variable "project_id" {
  description = "The GCP Project ID"
  type        = string
}

variable "region" {
  description = "The region for Firestore and other resources (e.g., us-central1)"
  type        = string
  default     = "us-central1"
}
`;

const TERRAFORM_OUTPUTS = `output "gopherbot_service_account_email" {
  value = google_service_account.gopherbot.email
}

output "chat_topic_id" {
  value = google_pubsub_topic.chat_topic.id
}

output "chat_subscription_id" {
  value = google_pubsub_subscription.chat_subscription.id
}

output "service_account_key" {
  value     = google_service_account_key.gopherbot_key.private_key
  sensitive = true
}

output "instructions" {
  value = "Save the 'service_account_key' output to a file named 'gopherbot-key.json'. You can use 'terraform output -raw service_account_key | base64 --decode > gopherbot-key.json'"
}
`;

export default function App() {
  const [activeTab, setActiveTab] = useState<"guide" | "terraform">("guide");
  const [copied, setCopied] = useState<string | null>(null);

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(null), 2000);
  };

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 font-sans selection:bg-blue-100">
      {/* Header */}
      <header className="bg-white border-b border-slate-200 sticky top-0 z-10">
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="bg-blue-600 p-2 rounded-lg">
              <Bot className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-xl font-bold tracking-tight">Gopherbot GCloud Assistant</h1>
              <p className="text-xs text-slate-500 font-medium">DevOps Chatbot Infrastructure Guide</p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <a 
              href="https://github.com/lnxjedi/gopherbot" 
              target="_blank" 
              rel="noopener noreferrer"
              className="text-slate-500 hover:text-slate-900 transition-colors"
            >
              <Github className="w-5 h-5" />
            </a>
          </div>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-12">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-12">
          {/* Sidebar Navigation */}
          <aside className="lg:col-span-3 space-y-6">
            <nav className="space-y-1">
              <button
                onClick={() => setActiveTab("guide")}
                className={cn(
                  "w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all",
                  activeTab === "guide" 
                    ? "bg-blue-50 text-blue-700 shadow-sm" 
                    : "text-slate-600 hover:bg-slate-100"
                )}
              >
                <BookOpen className="w-4 h-4" />
                Setup Guide
              </button>
              <button
                onClick={() => setActiveTab("terraform")}
                className={cn(
                  "w-full flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-medium transition-all",
                  activeTab === "terraform" 
                    ? "bg-blue-50 text-blue-700 shadow-sm" 
                    : "text-slate-600 hover:bg-slate-100"
                )}
              >
                <Code2 className="w-4 h-4" />
                Terraform Files
              </button>
            </nav>

            <div className="p-4 bg-amber-50 rounded-2xl border border-amber-100">
              <h3 className="text-xs font-bold text-amber-800 uppercase tracking-wider mb-2">Key Components</h3>
              <ul className="space-y-3">
                <li className="flex items-start gap-3">
                  <Database className="w-4 h-4 text-amber-600 mt-0.5" />
                  <span className="text-xs text-amber-900 leading-relaxed">
                    <strong>Firestore</strong>: Used as the robot's persistent "brain" for state and memory.
                  </span>
                </li>
                <li className="flex items-start gap-3">
                  <MessageSquare className="w-4 h-4 text-amber-600 mt-0.5" />
                  <span className="text-xs text-amber-900 leading-relaxed">
                    <strong>Google Chat</strong>: The primary interface for users to interact with the bot.
                  </span>
                </li>
                <li className="flex items-start gap-3">
                  <Cloud className="w-4 h-4 text-amber-600 mt-0.5" />
                  <span className="text-xs text-amber-900 leading-relaxed">
                    <strong>Pub/Sub</strong>: Enables real-time message delivery from Google Chat to the bot.
                  </span>
                </li>
              </ul>
            </div>
          </aside>

          {/* Main Content Area */}
          <div className="lg:col-span-9">
            <AnimatePresence mode="wait">
              {activeTab === "guide" ? (
                <motion.div
                  key="guide"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  className="bg-white rounded-3xl border border-slate-200 shadow-sm overflow-hidden"
                >
                  <div className="p-8 lg:p-12 prose prose-slate max-w-none prose-headings:font-bold prose-h1:text-3xl prose-h2:text-xl prose-h2:mt-10 prose-h2:border-b prose-h2:pb-2 prose-code:text-blue-600 prose-code:bg-blue-50 prose-code:px-1 prose-code:rounded prose-pre:bg-slate-900 prose-pre:text-slate-100">
                    <Markdown>{README_CONTENT}</Markdown>
                  </div>
                </motion.div>
              ) : (
                <motion.div
                  key="terraform"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  className="space-y-8"
                >
                  <FileBlock 
                    title="main.tf" 
                    content={TERRAFORM_MAIN} 
                    onCopy={() => copyToClipboard(TERRAFORM_MAIN, "main")}
                    isCopied={copied === "main"}
                  />
                  <FileBlock 
                    title="variables.tf" 
                    content={TERRAFORM_VARS} 
                    onCopy={() => copyToClipboard(TERRAFORM_VARS, "vars")}
                    isCopied={copied === "vars"}
                  />
                  <FileBlock 
                    title="outputs.tf" 
                    content={TERRAFORM_OUTPUTS} 
                    onCopy={() => copyToClipboard(TERRAFORM_OUTPUTS, "outputs")}
                    isCopied={copied === "outputs"}
                  />
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-200 mt-24 py-12">
        <div className="max-w-5xl mx-auto px-6 flex flex-col md:flex-row items-center justify-between gap-6">
          <p className="text-sm text-slate-500">
            &copy; {new Date().getFullYear()} Gopherbot Project. Built for Google Cloud Platform.
          </p>
          <div className="flex items-center gap-6">
            <a href="https://github.com/lnxjedi/gopherbot" className="text-sm font-medium text-slate-600 hover:text-blue-600 flex items-center gap-1.5">
              Documentation <ExternalLink className="w-3 h-3" />
            </a>
            <a href="https://github.com/lnxjedi/gopherbot/issues" className="text-sm font-medium text-slate-600 hover:text-blue-600 flex items-center gap-1.5">
              Support <ExternalLink className="w-3 h-3" />
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}

function FileBlock({ title, content, onCopy, isCopied }: { title: string, content: string, onCopy: () => void, isCopied: boolean }) {
  return (
    <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
      <div className="bg-slate-50 px-6 py-3 border-b border-slate-200 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Terminal className="w-4 h-4 text-slate-400" />
          <span className="text-sm font-mono font-medium text-slate-700">{title}</span>
        </div>
        <button
          onClick={onCopy}
          className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white border border-slate-200 text-xs font-medium text-slate-600 hover:bg-slate-50 hover:text-blue-600 transition-all active:scale-95"
        >
          {isCopied ? (
            <>
              <Check className="w-3.5 h-3.5 text-green-500" />
              Copied!
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              Copy Code
            </>
          )}
        </button>
      </div>
      <div className="p-0 overflow-x-auto">
        <pre className="p-6 text-sm font-mono text-slate-800 leading-relaxed">
          <code>{content}</code>
        </pre>
      </div>
    </div>
  );
}

