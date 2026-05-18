variable "project_id" {
  description = "Google Cloud project ID that already contains robot resources"
  type        = string
}

variable "region" {
  description = "Google Cloud region for regional resources"
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "Google Cloud zone for the VM"
  type        = string
  default     = "us-central1-a"
}

variable "bot_name" {
  description = "Robot name and Linux user name"
  type        = string
}



variable "gopherbot_version" {
  description = "Gopherbot release tag, or latest"
  type        = string
  default     = "latest"
}

variable "gopherbot_nobody" {
  description = "Whether to configure setuid/setgid nobody mode and clear robot supplementary groups before startup"
  type        = bool
  default     = false
}

variable "machine_type" {
  description = "Compute Engine machine type; e2-micro is free-tier eligible in supported regions"
  type        = string
  default     = "e2-micro"
}

variable "boot_disk_size_gb" {
  description = "Boot disk size in GB"
  type        = number
  default     = 20
}

variable "network_name" {
  description = "VPC network name"
  type        = string
  default     = "gopherbot-net"
}

variable "subnetwork_name" {
  description = "Subnetwork name"
  type        = string
  default     = "gopherbot-subnet"
}

variable "subnetwork_cidr" {
  description = "Subnetwork CIDR"
  type        = string
  default     = "10.42.0.0/24"
}

variable "enable_ssh_ingress" {
  description = "Whether to create a GCP firewall rule allowing inbound SSH"
  type        = bool
  default     = false
}

variable "allow_ssh_cidrs" {
  description = "CIDRs allowed to SSH to the instance when enable_ssh_ingress is true"
  type        = list(string)
  default     = ["35.235.240.0/20"]
}

variable "create_static_ip" {
  description = "Whether to reserve and attach a static external IP"
  type        = bool
  default     = true
}

variable "enable_vpn" {
  description = "Whether to configure WireGuard VPN on the robot VM"
  type        = bool
  default     = true
}

variable "wireguard_port" {
  description = "WireGuard UDP listen port"
  type        = number
  default     = 51820
}

variable "vpn_cidr" {
  description = "WireGuard interface CIDR for the server, such as 10.77.0.1/24"
  type        = string
  default     = "10.77.0.1/24"
}

variable "enable_firewall" {
  description = "Whether to block WireGuard connections by default and require ALLOW_VPN host rules"
  type        = bool
  default     = true
}

variable "vm_service_account_id" {
  description = "Service account ID for the VM runtime identity"
  type        = string
  default     = "gopherbot-vm"
}

variable "robot_env_secret_name" {
  description = "Secret Manager secret name containing the full robot .env file"
  type        = string
}

variable "wireguard_private_key_secret_name" {
  description = "Optional secret name containing WireGuard private key"
  type        = string
  default     = ""
}

variable "systemd_timeout_stop_sec" {
  description = "systemd TimeoutStopSec for the robot service"
  type        = number
  default     = 600
}

variable "labels" {
  description = "Resource labels"
  type        = map(string)
  default     = {}
}
