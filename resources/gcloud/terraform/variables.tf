variable "project_id" {
  description = "The GCP Project ID"
  type        = string
}

variable "region" {
  description = "The region for Firestore and other resources (e.g., us-central1)"
  type        = string
  default     = "us-central1"
}
