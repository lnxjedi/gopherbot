resource "google_compute_network" "bot" {
  project                 = var.project_id
  name                    = var.network_name
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "bot" {
  project       = var.project_id
  name          = var.subnetwork_name
  region        = var.region
  network       = google_compute_network.bot.id
  ip_cidr_range = var.subnetwork_cidr
}

resource "google_compute_firewall" "ssh" {
  count   = var.enable_ssh_ingress ? 1 : 0
  project = var.project_id
  name    = "${var.bot_name}-allow-ssh"
  network = google_compute_network.bot.name

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = var.allow_ssh_cidrs
  target_tags   = ["gopherbot"]
}

resource "google_compute_firewall" "wireguard" {
  count   = var.enable_vpn ? 1 : 0
  project = var.project_id
  name    = "${var.bot_name}-allow-wireguard"
  network = google_compute_network.bot.name

  allow {
    protocol = "udp"
    ports    = [tostring(var.wireguard_port)]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["wireguard"]
}

resource "google_compute_address" "bot" {
  count   = var.create_static_ip ? 1 : 0
  project = var.project_id
  name    = "${var.bot_name}-ip"
  region  = var.region
}
