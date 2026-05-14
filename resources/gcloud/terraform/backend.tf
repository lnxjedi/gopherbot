terraform {
  # Configure the backend by passing values from backend.hcl, for example:
  # terraform init -backend-config=backend.hcl
  backend "gcs" {}
}
