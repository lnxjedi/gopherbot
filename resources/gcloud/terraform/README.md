# Gopherbot Compute Engine Terraform

This directory is a copy-friendly Terraform deployment scaffold for running a single Gopherbot robot on Google Compute Engine.

It is designed for running in Google Cloud Shell after you already created project-level Chat and Firestore resources with scripts under resources/gcloud/scripts.

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

1. Existing GCP project and enabled APIs.
2. Existing robot integration resources created from resources/gcloud/scripts.
3. A GCS bucket for Terraform state.
4. A Secret Manager secret containing the robot .env content.
5. If VPN is enabled: a Secret Manager secret containing WireGuard private key text.

## Prepare Terraform backend

Create the state bucket once:

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

Create the robot environment secret from your local .env file:

```bash
gcloud secrets create bishop-env --replication-policy=automatic
gcloud secrets versions add bishop-env --data-file=/path/to/.env
```

If using WireGuard:

```bash
gcloud secrets create bishop-wireguard-private-key --replication-policy=automatic
printf '%s' 'YOUR_WIREGUARD_PRIVATE_KEY' | gcloud secrets versions add bishop-wireguard-private-key --data-file=-
```

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
GOPHER_DEPLOY_KEY=-----BEGIN OPENSSH PRIVATE KEY-----...-----END OPENSSH PRIVATE KEY-----
GOPHER_ENCRYPTION_KEY=...encrypted key material...
GOPHER_PROTOCOL=googlechat
GOPHER_BOTNAME=bishop
```
