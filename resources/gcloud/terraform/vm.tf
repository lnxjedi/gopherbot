data "google_compute_image" "debian" {
  project = "debian-cloud"
  family  = "debian-12"
}

locals {
  robot_home = "/var/lib/robots/${var.bot_name}"

  wireguard_secret_name = var.wireguard_private_key_secret_name != "" ? var.wireguard_private_key_secret_name : ""

  startup_script = templatefile("${path.module}/bootstrap.tpl", {
    bot_name                 = var.bot_name
    bot_home                 = local.robot_home
    project_id               = var.project_id
    robot_repository         = var.robot_repository
    protocol                 = var.protocol
    gopherbot_version        = var.gopherbot_version
    gopherbot_nobody         = var.gopherbot_nobody
    robot_env_secret_name    = var.robot_env_secret_name
    wireguard_secret_name    = local.wireguard_secret_name
    enable_vpn               = var.enable_vpn
    wireguard_port           = var.wireguard_port
    vpn_cidr                 = var.vpn_cidr
    enable_firewall          = var.enable_firewall
    systemd_timeout_stop_sec = var.systemd_timeout_stop_sec
  })
}

resource "google_compute_instance" "bot" {
  project        = var.project_id
  zone           = var.zone
  name           = "${var.bot_name}-robot"
  machine_type   = var.machine_type
  can_ip_forward = var.enable_vpn

  tags = compact([
    "gopherbot",
    var.enable_vpn ? "wireguard" : ""
  ])

  labels = merge(
    {
      robot = var.bot_name
    },
    var.labels
  )

  boot_disk {
    initialize_params {
      image = data.google_compute_image.debian.self_link
      size  = var.boot_disk_size_gb
      type  = "pd-balanced"
    }
  }

  network_interface {
    network    = google_compute_network.bot.id
    subnetwork = google_compute_subnetwork.bot.id

    access_config {
      nat_ip = var.create_static_ip ? google_compute_address.bot[0].address : null
    }
  }

  service_account {
    email  = google_service_account.bot_vm.email
    scopes = ["https://www.googleapis.com/auth/cloud-platform"]
  }

  metadata_startup_script = local.startup_script
}
