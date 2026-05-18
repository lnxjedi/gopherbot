#!/bin/bash
set -euo pipefail

exec > >(tee -a /var/log/gopherbot-bootstrap.log) 2>&1

echo "Starting Gopherbot bootstrap"

BOT_NAME="${bot_name}"
BOT_HOME="${bot_home}"
PROJECT_ID="${project_id}"
ROBOT_ENV_SECRET_NAME="${robot_env_secret_name}"
WIREGUARD_SECRET_NAME="${wireguard_secret_name}"
ENABLE_VPN="${enable_vpn}"
WIREGUARD_PORT="${wireguard_port}"
VPN_CIDR="${vpn_cidr}"
ENABLE_FIREWALL="${enable_firewall}"

apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y \
  ca-certificates \
  curl \
  git \
  iptables \
  jq \
  python3-pip \
  ruby-full \
  wireguard

get_access_token() {
  curl -fsS -H "Metadata-Flavor: Google" \
    "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token" \
    | jq -r '.access_token'
}

read_secret_value() {
  local secret_name="$1"
  local token
  token="$(get_access_token)"

  curl -fsS \
    -H "Authorization: Bearer $${token}" \
    "https://secretmanager.googleapis.com/v1/projects/$${PROJECT_ID}/secrets/$${secret_name}/versions/latest:access" \
    | jq -r '.payload.data' \
    | tr '_-' '/+' \
    | base64 --decode
}

echo "Reading robot .env from Secret Manager secret: $${ROBOT_ENV_SECRET_NAME}"
ROBOT_ENV_CONTENT="$(read_secret_value "$${ROBOT_ENV_SECRET_NAME}")"

if [[ -z "$${ROBOT_ENV_CONTENT}" ]]; then
  echo "Secret $${ROBOT_ENV_SECRET_NAME} returned empty content" >&2
  exit 1
fi

echo "Installing Gopherbot"
GBDL="/root/gopherbot.tar.gz"
if [[ "${gopherbot_version}" == "latest" ]]; then
  GB_VERSION="$(curl -fsS https://api.github.com/repos/lnxjedi/gopherbot/releases/latest | jq -r .tag_name)"
else
  GB_VERSION="${gopherbot_version}"
fi

curl -fsSL -o "$${GBDL}" "https://github.com/lnxjedi/gopherbot/releases/download/$${GB_VERSION}/gopherbot-linux-amd64.tar.gz"
mkdir -p /opt
cd /opt
rm -rf gopherbot
tar xzf "$${GBDL}"
rm -f "$${GBDL}"

if [[ "${gopherbot_nobody}" == "true" ]]; then
  /opt/gopherbot/setuid-nobody.sh
  iptables -A OUTPUT -m owner --uid-owner nobody -d 169.254.169.254 -j DROP
fi

if [[ "$${ENABLE_VPN}" == "true" ]]; then
  if [[ -z "$${WIREGUARD_SECRET_NAME}" ]]; then
    echo "WireGuard is enabled but wireguard_private_key_secret_name is empty" >&2
    exit 1
  fi

  echo "Configuring WireGuard"
  WG_PRIVATE="$(read_secret_value "$${WIREGUARD_SECRET_NAME}")"

  cat > /etc/wireguard/wg0.conf <<EOF
[Interface]
Address = $${VPN_CIDR}
PrivateKey = $${WG_PRIVATE}
ListenPort = $${WIREGUARD_PORT}
PostUp = /etc/wireguard/start-nat.sh
PostDown = /etc/wireguard/stop-nat.sh
EOF

  cat > /etc/wireguard/start-nat.sh <<EOF
#!/bin/bash
set -euo pipefail
echo 1 > /proc/sys/net/ipv4/ip_forward
ETHERNET_INT=\$(ip route | awk '/default/ {print \$5; exit}')

iptables -N ALLOW_VPN 2>/dev/null || true
iptables -F ALLOW_VPN

iptables -t nat -I POSTROUTING 1 -s $${VPN_CIDR} -o \$${ETHERNET_INT} -j MASQUERADE
iptables -I INPUT 1 -i wg0 -j ACCEPT
iptables -I FORWARD 1 -i \$${ETHERNET_INT} -o wg0 -j ACCEPT
iptables -I FORWARD 1 -i wg0 -o \$${ETHERNET_INT} -j ACCEPT
EOF

  cat > /etc/wireguard/stop-nat.sh <<EOF
#!/bin/bash
set -euo pipefail
echo 0 > /proc/sys/net/ipv4/ip_forward
ETHERNET_INT=\$(ip route | awk '/default/ {print \$5; exit}')

iptables -t nat -D POSTROUTING -s $${VPN_CIDR} -o \$${ETHERNET_INT} -j MASQUERADE
iptables -D INPUT -i wg0 -j ACCEPT
iptables -D FORWARD -i \$${ETHERNET_INT} -o wg0 -j ACCEPT
iptables -D FORWARD -i wg0 -o \$${ETHERNET_INT} -j ACCEPT
EOF

  if [[ "$${ENABLE_FIREWALL}" == "true" ]]; then
    cat >> /etc/wireguard/start-nat.sh <<EOF
iptables -I INPUT 1 -i \$${ETHERNET_INT} -p udp --dport $${WIREGUARD_PORT} -j DROP
iptables -I INPUT 1 -i \$${ETHERNET_INT} -p udp --dport $${WIREGUARD_PORT} -j ALLOW_VPN
EOF

    cat >> /etc/wireguard/stop-nat.sh <<EOF
iptables -D INPUT -i \$${ETHERNET_INT} -p udp --dport $${WIREGUARD_PORT} -j ALLOW_VPN || true
iptables -D INPUT -i \$${ETHERNET_INT} -p udp --dport $${WIREGUARD_PORT} -j DROP || true
iptables -F ALLOW_VPN || true
iptables -X ALLOW_VPN || true
EOF
  fi

  chmod +x /etc/wireguard/start-nat.sh /etc/wireguard/stop-nat.sh
  systemctl enable wg-quick@wg0
  systemctl start wg-quick@wg0
fi

mkdir -p /var/lib/robots
if ! id -u "$${BOT_NAME}" >/dev/null 2>&1; then
  useradd -d "$${BOT_HOME}" -r -m -c "$${BOT_NAME} gopherbot" "$${BOT_NAME}"
fi

mkdir -p "$${BOT_HOME}"
printf '%s\n' "$${ROBOT_ENV_CONTENT}" > "$${BOT_HOME}/.env"

chown -R "$${BOT_NAME}:$${BOT_NAME}" "$${BOT_HOME}"
chmod 0600 "$${BOT_HOME}/.env"

cat > "/etc/sudoers.d/$${BOT_NAME}-user" <<EOF
# User rules for robot
$${BOT_NAME} ALL=(ALL) NOPASSWD:ALL
EOF
chmod 0440 "/etc/sudoers.d/$${BOT_NAME}-user"

cat > "/etc/systemd/system/$${BOT_NAME}.service" <<EOF
[Unit]
Description=$${BOT_NAME} - Gopherbot DevOps Chatbot
Documentation=https://lnxjedi.github.io/gopherbot
After=syslog.target
After=network.target

[Service]
Type=simple
User=$${BOT_NAME}
Group=$${BOT_NAME}
WorkingDirectory=$${BOT_HOME}
ExecStart=/opt/gopherbot/gopherbot -plainlog
Restart=on-failure
Environment=HOSTNAME=%H
KillMode=process
TimeoutStopSec=${systemd_timeout_stop_sec}

[Install]
WantedBy=default.target
EOF

systemctl daemon-reload
systemctl enable "$${BOT_NAME}"
systemctl start "$${BOT_NAME}"

echo "Bootstrap complete"
