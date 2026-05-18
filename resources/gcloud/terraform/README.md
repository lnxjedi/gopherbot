# Gopherbot Compute Engine Terraform

This directory is a copy-friendly Terraform deployment scaffold for running a single Gopherbot robot on Google Compute Engine.

It is designed for running in Google Cloud Shell after you already created project-level Chat and Firestore resources with scripts under resources/gcloud/scripts.

## Overview

This module codifies the infrastructure and runtime setup for a single Gopherbot instance on GCE. The workflow is:

1. **Prepare project-level resources** — Chat integration, Firestore brain, service accounts, etc. (via `resources/gcloud/scripts`)
2. **Create deployment infrastructure** — VPC, firewall rules, static IP, service account for the robot VM
3. **Prepare secrets** — Store robot .env and optional WireGuard key in Google Secret Manager
4. **Configure and deploy** — Use Terraform variables, then apply to instantiate the VM with bootstrap

The VM startup script installs Gopherbot from a release tarball, retrieves secrets, configures optional VPN, and starts the robot service. No manual SSH or post-deployment configuration is needed.

## What this creates

- a dedicated VPC and subnet
- firewall rule for WireGuard VPN UDP only by default
- optional SSH firewall rule (disabled by default)
- an optional reserved static external IP
- a VM service account with secret access
- a Debian 12 Compute Engine instance with startup bootstrap
- robot runtime at /var/lib/robots/${bot_name}
- systemd service ${bot_name}.service

## Prerequisites

1. Existing GCP project.
2. Existing robot integration resources created from resources/gcloud/scripts.

## Enable Required APIs

Enable the APIs required by this module:

```bash
gcloud services enable \
  compute.googleapis.com \
  secretmanager.googleapis.com \
  storage.googleapis.com
```

## Prepare Terraform backend

Create the state bucket once using the `gcloud storage` CLI (recommended by Google over `gsutil`):

```bash
gcloud storage buckets create gs://$PROJECT_ID-terraform-state --location=$REGION
gcloud storage buckets update gs://$PROJECT_ID-terraform-state --versioning
```

Alternatively, using the older `gsutil` CLI:

```bash
gsutil mb -p "$PROJECT_ID" -l "$REGION" "gs://$PROJECT_ID-terraform-state"
gsutil versioning set on "gs://$PROJECT_ID-terraform-state"
```

Create backend config:

```bash
cp backend.hcl.example backend.hcl
```

Edit backend.hcl with your bucket name and a per-robot prefix.

## Prepare robot secrets

First, install WireGuard tools in Cloud Shell if you plan to use VPN:

```bash
sudo apt-get update && sudo apt-get install -y wireguard-tools
```

Create the robot environment secret from your local .env file:

```bash
gcloud secrets create bishop-env --replication-policy=automatic
gcloud secrets versions add bishop-env --data-file=/path/to/.env
```

If using WireGuard, generate a key pair locally:

```bash
wg genkey | tee wg-private.txt | wg pubkey > wg-public.txt
```

Then store the private key in Secret Manager:

```bash
gcloud secrets create bishop-wireguard-private-key --replication-policy=automatic
gcloud secrets versions add bishop-wireguard-private-key --data-file=wg-private.txt
```

Save the public key (`wg-public.txt`) for use in peer configuration elsewhere.

## Configure Terraform variables

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit terraform.tfvars values.

## Deploy

```bash
terraform init -backend-config=backend.hcl
terraform plan
terraform apply
```

## Notes

- Free-tier eligibility depends on region, machine type, and account limits.
- The startup script installs Gopherbot from GitHub release tarballs.
- Set gopherbot_version to a release tag (for example v2.9.0) to pin version.
- robot_env_secret_name should contain the full .env content expected by your robot.
- enable_firewall = true configures host iptables to default-deny WireGuard and require explicit ALLOW_VPN entries.
- enable_ssh_ingress = false means no inbound tcp/22 rule is created in GCP.

Example minimum .env content:

```dotenv
GOPHER_CUSTOM_REPOSITORY=git@github.com:your-org/your-robot-repo.git
GOPHER_DEPLOY_KEY=-----BEGIN_OPENSSH_PRIVATE_KEY-----...-----END_OPENSSH_PRIVATE_KEY-----
GOPHER_ENCRYPTION_KEY=<encryption_key>
```
