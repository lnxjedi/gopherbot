output "instance_name" {
  value       = google_compute_instance.bot.name
  description = "Compute Engine instance name"
}

output "instance_zone" {
  value       = google_compute_instance.bot.zone
  description = "Compute Engine instance zone"
}

output "external_ip" {
  value       = google_compute_instance.bot.network_interface[0].access_config[0].nat_ip
  description = "Robot public IPv4 address"
}

output "robot_home" {
  value       = local.robot_home
  description = "Robot home directory on the VM"
}

output "ssh_command" {
  value       = "gcloud compute ssh --project ${var.project_id} --zone ${var.zone} ${google_compute_instance.bot.name}"
  description = "Convenience command to SSH to the robot VM"
}
