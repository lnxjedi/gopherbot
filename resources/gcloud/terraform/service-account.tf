resource "google_service_account" "bot_vm" {
  project      = var.project_id
  account_id   = var.vm_service_account_id
  display_name = "${var.bot_name} VM runtime"
}

resource "google_project_iam_member" "bot_vm_logging" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.bot_vm.email}"
}

resource "google_project_iam_member" "bot_vm_monitoring" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.bot_vm.email}"
}

data "google_secret_manager_secret" "robot_env" {
  project   = var.project_id
  secret_id = var.robot_env_secret_name
}

resource "google_secret_manager_secret_iam_member" "robot_env_access" {
  project   = var.project_id
  secret_id = data.google_secret_manager_secret.robot_env.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.bot_vm.email}"
}

data "google_secret_manager_secret" "wireguard" {
  count     = var.enable_vpn && var.wireguard_private_key_secret_name != "" ? 1 : 0
  project   = var.project_id
  secret_id = var.wireguard_private_key_secret_name
}

resource "google_secret_manager_secret_iam_member" "wireguard_access" {
  count     = var.enable_vpn && var.wireguard_private_key_secret_name != "" ? 1 : 0
  project   = var.project_id
  secret_id = data.google_secret_manager_secret.wireguard[0].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.bot_vm.email}"
}
