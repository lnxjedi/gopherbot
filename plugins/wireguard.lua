local gopherbot = require("gopherbot_v1")
local bot = gopherbot.Robot:new()
local ret = gopherbot.ret
local task = gopherbot.task
local log = gopherbot.log

local command = arg[1]

if command == "_configure" then
  return "---\nCommands: []"
end

local function say(message)
  bot:Say(message)
end

local function log_msg(level, message)
  bot:Log(level, "wireguard: " .. tostring(message))
end

local function log_debug(message)
  log_msg(log.Debug, message)
end

local function log_info(message)
  log_msg(log.Debug, message)
end

local function log_warn(message)
  log_msg(log.Warn, message)
end

local function log_error(message)
  log_msg(log.Error, message)
end

local function trim(s)
  return tostring(s or ""):match("^%s*(.-)%s*$")
end

local environment = trim(bot:GetParameter("GOPHER_ENVIRONMENT"))
local dry_run = environment == "development"

local function summarize_output(out)
  out = tostring(out or ""):gsub("%s+$", "")
  if out == "" then
    return "<empty>"
  end
  if #out > 300 then
    return out:sub(1, 300) .. "...<truncated>"
  end
  return out
end

local function shell_quote(s)
  s = tostring(s or "")
  return "'" .. s:gsub("'", "'\"'\"'") .. "'"
end

local function shell_capture(cmd, label)
  label = label or cmd
  local marker = "__GBOT_STATUS__"
  local pipe, err = io.popen(cmd .. " 2>&1; _gb_status=$?; printf '\\n" .. marker .. "%s' \"$_gb_status\"", "r")
  if not pipe then
    log_error("shell popen failed: " .. label .. ": " .. tostring(err))
    return false, "", tostring(err)
  end
  local out = pipe:read("*a") or ""
  pipe:close()
  local body, status = out:match("^(.*)\n" .. marker .. "(%d+)$")
  if not status then
    log_error("shell missing status: " .. label .. ": " .. summarize_output(out))
    return false, out, "missing status"
  end
  local ok = tonumber(status) == 0
  if not ok then
    log_warn("shell failed: " .. label .. " status=" .. tostring(status) .. " output=" .. summarize_output(body))
  end
  return ok, body or "", status
end

local function write_temp_file(content)
  local ok, tmp_path, status = shell_capture("mktemp /tmp/gopherbot-wireguard.XXXXXX", "mktemp wireguard config")
  tmp_path = trim(tmp_path)
  if not ok or tmp_path == "" then
    log_error("temp write failed: mktemp status=" .. tostring(status) .. " output=" .. summarize_output(tmp_path))
    return nil, "mktemp failed"
  end

  local f, err = io.open(tmp_path, "w")
  if not f then
    log_error("temp write failed: open temp: " .. tostring(err))
    shell_capture("rm -f " .. shell_quote(tmp_path), "cleanup temp wireguard config")
    return nil, tostring(err)
  end

  f:write(content)
  f:close()

  shell_capture("chmod 0600 " .. shell_quote(tmp_path), "chmod temp wireguard config")
  return tmp_path, ""
end

local function sudo_write_file(path, content)
  log_info("sudo write start: path=" .. tostring(path) .. " bytes=" .. tostring(#(content or "")))

  local tmp_path, err = write_temp_file(content)
  if not tmp_path then
    return false, err
  end

  local install_cmd = "timeout 20s sudo -n install -m 0600 -o root -g root " ..
    shell_quote(tmp_path) .. " " .. shell_quote(path)
  local install_ok, install_out, install_status = shell_capture(install_cmd, "install wireguard config")
  shell_capture("rm -f " .. shell_quote(tmp_path), "cleanup temp wireguard config")

  if not install_ok then
    log_error("sudo write failed: install status=" .. tostring(install_status) .. " output=" .. summarize_output(install_out))
    return false, "install status=" .. tostring(install_status)
  end

  log_info("sudo write done: path=" .. tostring(path))
  return true, ""
end

local function md5_file(path, use_sudo)
  local prefix = use_sudo and "timeout 10s sudo -n " or "timeout 10s "
  local ok, out, status = shell_capture(prefix .. "md5sum " .. shell_quote(path), "md5sum wireguard config")
  if not ok then
    return nil, status
  end
  return tostring(out or ""):match("^(%x+)")
end

local function config_unchanged(path, content)
  local tmp_path, err = write_temp_file(content)
  if not tmp_path then
    log_warn("config compare skipped: temp write failed: " .. tostring(err))
    return false
  end

  local desired_md5 = md5_file(tmp_path, false)
  shell_capture("rm -f " .. shell_quote(tmp_path), "cleanup temp wireguard config")
  if not desired_md5 then
    log_warn("config compare skipped: unable to hash rendered config")
    return false
  end

  local current_md5 = md5_file(path, true)
  if not current_md5 then
    log_info("config compare skipped: live config unavailable")
    return false
  end
  return desired_md5 == current_md5
end

local function parse_ipv4(ip)
  local a, b, c, d = tostring(ip):match("^(%d+)%.(%d+)%.(%d+)%.(%d+)$")
  a, b, c, d = tonumber(a), tonumber(b), tonumber(c), tonumber(d)
  if not a or not b or not c or not d then
    return nil
  end
  if a > 255 or b > 255 or c > 255 or d > 255 then
    return nil
  end
  return a * 16777216 + b * 65536 + c * 256 + d
end

local function format_ipv4(n)
  local a = math.floor(n / 16777216) % 256
  local b = math.floor(n / 65536) % 256
  local c = math.floor(n / 256) % 256
  local d = n % 256
  return string.format("%d.%d.%d.%d", a, b, c, d)
end

local function split_cidr(cidr)
  local ip, prefix = tostring(cidr or ""):match("^([^/]+)/(%d+)$")
  if not ip then
    return nil, nil
  end
  prefix = tonumber(prefix)
  if not parse_ipv4(ip) or not prefix or prefix < 0 or prefix > 32 then
    return nil, nil
  end
  return ip, prefix
end

local function ipv4_number_from_cidr(cidr_or_host)
  local ip = tostring(cidr_or_host or ""):match("^([^/]+)") or ""
  return parse_ipv4(ip)
end

local function is_global_ipv4(address)
  local ip = tostring(address or ""):match("^([^/]+)") or ""
  local n = parse_ipv4(ip)
  if not n then
    return false
  end
  local first = math.floor(n / 16777216) % 256
  local second = math.floor(n / 65536) % 256
  if first == 0 or first == 10 or first == 127 or first >= 224 then
    return false
  end
  if first == 100 and second >= 64 and second <= 127 then
    return false
  end
  if first == 169 and second == 254 then
    return false
  end
  if first == 172 and second >= 16 and second <= 31 then
    return false
  end
  if first == 192 and second == 168 then
    return false
  end
  return true
end

local function sorted_keys(tbl)
  local keys = {}
  for key in pairs(tbl or {}) do
    table.insert(keys, key)
  end
  table.sort(keys)
  return keys
end

local function state_counts(state)
  local user_count = 0
  local device_count = 0
  local users = state and state.datum and state.datum.Users or {}
  for _, devices in pairs(users) do
    user_count = user_count + 1
    for _ in pairs(devices or {}) do
      device_count = device_count + 1
    end
  end
  return user_count, device_count
end

-- WireGuard keys are 32 bytes encoded as canonical, padded base64. This
-- mirrors wireguard-tools key_from_base64(): 44 characters total, the
-- standard base64 alphabet, one trailing '=', and zero padding bits in the
-- final data character.
local canonical_key_tail = {
  A = true, E = true, I = true, M = true,
  Q = true, U = true, Y = true, c = true,
  g = true, k = true, o = true, s = true,
  w = true, ["0"] = true, ["4"] = true, ["8"] = true,
}

local function valid_wireguard_key(value)
  if type(value) ~= "string" or #value ~= 44 then
    return false
  end
  if value:sub(44, 44) ~= "=" then
    return false
  end
  if not value:sub(1, 43):match("^[A-Za-z0-9+/]+$") then
    return false
  end
  return canonical_key_tail[value:sub(43, 43)] == true
end

local function validate_wireguard_state(state)
  local users = state and state.datum and state.datum.Users
  if type(users) ~= "table" then
    return false, "<state>", "<state>", "Users"
  end
  for _, username in ipairs(sorted_keys(users)) do
    local devices = users[username]
    if type(devices) ~= "table" then
      return false, username, "<devices>", "device map"
    end
    for _, device in ipairs(sorted_keys(devices)) do
      local data = devices[device]
      if type(data) ~= "table" then
        return false, username, device, "peer record"
      end
      if not valid_wireguard_key(data.PublicKey) then
        return false, username, device, "public key"
      end
      if not valid_wireguard_key(data.PreSharedKey) then
        return false, username, device, "pre-shared key"
      end
    end
  end
  return true
end

local function ensure_valid_wireguard_state(state, notify_user)
  local ok, username, device, field = validate_wireguard_state(state)
  if ok then
    return true
  end
  log_error("stored state validation failed: user=" .. tostring(username) ..
    " device=" .. tostring(device) .. " field=" .. tostring(field))
  if notify_user then
    say("WireGuard state contains invalid key data; ask an administrator to remove and re-add the affected VPN device.")
  end
  return false
end

local function allocate_ip(cfg, state)
  log_info("allocate_ip start: interface=" .. tostring(cfg.InterfaceAddress))
  local base_num = ipv4_number_from_cidr(cfg.InterfaceAddress)
  if not base_num then
    log_error("allocate_ip failed: invalid interface address")
    return nil
  end

  local used = {}
  local users = state and state.datum and state.datum.Users or {}
  for username, devices in pairs(users) do
    for device, data in pairs(devices or {}) do
      local ip_num = ipv4_number_from_cidr(data and data.AllowedIPs)
      if ip_num then
        used[ip_num] = true
      else
        log_warn("allocate_ip ignoring invalid AllowedIPs: user=" .. tostring(username) ..
          " device=" .. tostring(device) ..
          " allowed_ips=" .. tostring(data and data.AllowedIPs))
      end
    end
  end

  local candidate = base_num + 1
  local max_candidate = base_num + 65534
  while candidate <= max_candidate do
    if not used[candidate] then
      local allocated = format_ipv4(candidate) .. "/32"
      log_info("allocate_ip done: allocated=" .. allocated)
      return allocated
    end
    candidate = candidate + 1
  end

  log_error("allocate_ip failed: no free addresses")
  return nil
end

local function load_config()
  local cfg, rv = bot:GetTaskConfig()
  if rv ~= ret.Ok then
    log_error("load_config failed: ret=" .. tostring(rv))
    say("WireGuard plugin configuration is unavailable")
    return nil
  end
  cfg.Environment = environment
  cfg.DryRun = dry_run
  cfg.WireGuardConfigPath = cfg.WireGuardConfigPath or "/etc/wireguard/wg0.conf"
  cfg.InterfaceAddress = cfg.InterfaceAddress or "10.77.0.1/24"
  cfg.ListenPort = tonumber(cfg.ListenPort or 51820)
  cfg.PostUp = cfg.PostUp or "/etc/wireguard/start-nat.sh"
  cfg.PostDown = cfg.PostDown or "/etc/wireguard/stop-nat.sh"
  if not cfg.DryRun and (not cfg.PrivateKey or cfg.PrivateKey == "") then
    log_error("load_config failed: missing private key")
    say("WireGuard private key is not configured")
    return nil
  end
  if cfg.PrivateKey and cfg.PrivateKey ~= "" and not valid_wireguard_key(cfg.PrivateKey) then
    log_error("load_config failed: invalid private key")
    say("WireGuard private key is invalid")
    return nil
  end
  if cfg.PublicKey and cfg.PublicKey ~= "" and not valid_wireguard_key(cfg.PublicKey) then
    log_error("load_config failed: invalid public key")
    say("WireGuard public key is invalid")
    return nil
  end
  if not split_cidr(cfg.InterfaceAddress) then
    log_error("load_config failed: invalid interface address " .. tostring(cfg.InterfaceAddress))
    say("WireGuard interface address is invalid")
    return nil
  end
  log_info("load_config done: environment=" .. tostring(cfg.Environment) ..
    " dry_run=" .. tostring(cfg.DryRun) ..
    " path=" .. tostring(cfg.WireGuardConfigPath) ..
    " interface=" .. tostring(cfg.InterfaceAddress) ..
    " port=" .. tostring(cfg.ListenPort) ..
    " post_up=" .. tostring(cfg.PostUp) ..
    " post_down=" .. tostring(cfg.PostDown) ..
    " private_key_set=" .. tostring(cfg.PrivateKey ~= nil and cfg.PrivateKey ~= "") ..
    " public_key_set=" .. tostring(cfg.PublicKey ~= nil and cfg.PublicKey ~= ""))
  return cfg
end

local function checkout_state(rw)
  local state, rv = bot:CheckoutDatum("wg", rw)
  if rv ~= ret.Ok then
    log_error("checkout_state failed: ret=" .. tostring(rv))
    say("Unable to load WireGuard state")
    return nil
  end
  state.datum = state.datum or {}
  state.datum.Users = state.datum.Users or {}
  local user_count, device_count = state_counts(state)
  log_info("checkout_state done: exists=" .. tostring(state.exists) ..
    " users=" .. tostring(user_count) ..
    " devices=" .. tostring(device_count))
  return state
end

local function update_state(state)
  local user_count, device_count = state_counts(state)
  log_info("update_state start: users=" .. tostring(user_count) ..
    " devices=" .. tostring(device_count))
  local rv = bot:UpdateDatum(state)
  if rv ~= ret.Ok then
    log_error("update_state failed: ret=" .. tostring(rv))
    say("Unable to update WireGuard state")
    return false
  end
  return true
end

local function checkin_state(state)
  if state and state.token and state.token ~= "" then
    bot:CheckinDatum(state)
  end
end

local function gen_psk()
  if dry_run then
    log_info("gen_psk skipped: development dry-run")
    return "<dry-run-psk>"
  end
  local ok, out = shell_capture("wg genpsk", "wg genpsk")
  if not ok then
    log_error("gen_psk failed")
    return nil
  end
  return trim(out)
end

local function external_ip()
  local ok, http = pcall(require, "http")
  if not ok then
    log_error("external_ip failed: require(http): " .. tostring(http))
    return nil
  end
  local response, err = http.request("GET", "https://cloudflare.com/cdn-cgi/trace", { timeout = "5s" })
  if err or response.status_code ~= 200 then
    log_error("external_ip failed: status=" .. tostring(response and response.status_code) .. " err=" .. tostring(err))
    return nil
  end
  local ip = (response.body or ""):match("\nip=([^\n]+)") or (response.body or ""):match("^ip=([^\n]+)")
  return ip
end

local function endpoint_ip(cfg)
  if cfg and cfg.DryRun then
    return "<robot-public-ip>"
  end
  return external_ip() or "<robot-public-ip>"
end

local function render_config(cfg, state)
  local lines = {
    "[Interface]",
    "Address = " .. tostring(cfg.InterfaceAddress),
    "PrivateKey = " .. tostring(cfg.PrivateKey),
    "ListenPort = " .. tostring(cfg.ListenPort),
    "PostUp = " .. tostring(cfg.PostUp),
    "PostDown = " .. tostring(cfg.PostDown),
    "",
  }

  for _, user in ipairs(sorted_keys(state.datum.Users)) do
    local devices = state.datum.Users[user]
    for _, device in ipairs(sorted_keys(devices)) do
      local data = devices[device]
      table.insert(lines, "[Peer]")
      table.insert(lines, "# " .. user .. " | " .. device)
      table.insert(lines, "PublicKey = " .. tostring(data.PublicKey))
      table.insert(lines, "PreSharedKey = " .. tostring(data.PreSharedKey))
      table.insert(lines, "AllowedIPs = " .. tostring(data.AllowedIPs))
      table.insert(lines, "")
    end
  end

  local rendered = table.concat(lines, "\n")
  return rendered
end

local function apply_wireguard(cfg, state)
  log_info("apply_wireguard start: dry_run=" .. tostring(cfg.DryRun))
  if not ensure_valid_wireguard_state(state, command ~= "_init") then
    log_error("apply_wireguard aborted: invalid stored state")
    return false
  end
  if cfg.DryRun then
    log_info("apply_wireguard skipped: development dry-run")
    return true
  end

  local rendered = render_config(cfg, state)
  if config_unchanged(cfg.WireGuardConfigPath, rendered) then
    log_info("apply_wireguard skipped: config unchanged")
    return true
  end

  local ok, detail = sudo_write_file(cfg.WireGuardConfigPath, rendered)
  if not ok then
    log_error("apply_wireguard write failed: " .. tostring(detail))
    say("Unable to write WireGuard configuration: " .. tostring(detail))
    return false
  end

  local enable_ok, enable_out, enable_status = shell_capture("timeout 20s sudo -n systemctl enable wg-quick@wg0", "systemctl enable wg-quick@wg0")
  if not enable_ok then
    log_warn("apply_wireguard enable service failed: status=" .. tostring(enable_status) .. " output=" .. summarize_output(enable_out))
  end

  local restart_out, restart_status
  ok, restart_out, restart_status = shell_capture("timeout 45s sudo -n systemctl restart wg-quick@wg0", "systemctl restart wg-quick@wg0")
  if not ok then
    log_error("apply_wireguard restart failed: status=" .. tostring(restart_status) .. " output=" .. summarize_output(restart_out))
    say("Unable to restart WireGuard: status=" .. tostring(restart_status))
    return false
  end
  log_info("apply_wireguard done")
  return true
end

local function add_device(cfg, state, device, public_key)
  log_info("add_device start: user=" .. tostring(bot.user) ..
    " device=" .. tostring(device) ..
    " public_key_len=" .. tostring(#(public_key or "")))
  local username = bot.user
  if not username or username == "" then
    log_error("add_device failed: missing bot.user")
    say("Unable to determine the requesting user")
    return false
  end
  device = string.lower(device or "")
  if device == "" or not device:match("^[%.%w%-]+$") then
    log_error("add_device failed: invalid device=" .. tostring(device))
    say("Invalid device name")
    return false
  end
  if not valid_wireguard_key(public_key) then
    log_error("add_device failed: invalid public key format")
    say("Invalid WireGuard public key; expected the 44-character base64 value produced by 'wg pubkey'.")
    return false
  end
  if not ensure_valid_wireguard_state(state, true) then
    log_error("add_device failed: invalid stored state")
    return false
  end

  state.datum.Users[username] = state.datum.Users[username] or {}
  if state.datum.Users[username][device] then
    log_warn("add_device duplicate: user=" .. tostring(username) .. " device=" .. tostring(device))
    checkin_state(state)
    say("Error: Device Already Added.")
    return false
  end

  local user_ip = allocate_ip(cfg, state)
  if not user_ip then
    log_error("add_device failed: unable to allocate address")
    say("Unable to allocate a VPN address")
    return false
  end

  local psk = gen_psk()
  if not psk or psk == "" then
    log_error("add_device failed: psk generation failed")
    say("Unable to generate a WireGuard pre-shared key")
    return false
  end
  if not cfg.DryRun and not valid_wireguard_key(psk) then
    log_error("add_device failed: generated invalid pre-shared key")
    say("Unable to generate a valid WireGuard pre-shared key")
    return false
  end

  state.datum.Users[username][device] = {
    PublicKey = public_key,
    PreSharedKey = psk,
    AllowedIPs = user_ip,
  }

  if not cfg.DryRun and not ensure_valid_wireguard_state(state, true) then
    log_error("add_device failed: candidate state validation failed")
    return false
  end

  if cfg.DryRun then
    local ip = endpoint_ip(cfg)
    local robot_public_key = cfg.PublicKey or "<robot-public-key>"
    say("Dry-run: VPN device '" .. device .. "' would be added for user '" .. username .. "'. No brain state, WireGuard configuration, or host service changes were made.")
    say("VPN config data: Robot_IP = " .. ip .. ":" .. tostring(cfg.ListenPort) ..
      " | Robot_Public_Key = " .. tostring(robot_public_key) ..
      " | USER_IP = " .. user_ip ..
      " | PSK = " .. psk)
    log_info("add_device dry-run done: user=" .. tostring(username) .. " device=" .. tostring(device) .. " ip=" .. tostring(user_ip))
    return true
  end

  if not update_state(state) then
    log_error("add_device failed: state update failed")
    return false
  end
  if not apply_wireguard(cfg, state) then
    log_error("add_device failed: apply_wireguard failed")
    return false
  end

  local ip = endpoint_ip(cfg)
  local robot_public_key = cfg.PublicKey or "<robot-public-key>"
  say("VPN config data: Robot_IP = " .. ip .. ":" .. tostring(cfg.ListenPort) ..
    " | Robot_Public_Key = " .. tostring(robot_public_key) ..
    " | USER_IP = " .. user_ip ..
    " | PSK = " .. psk)
  log_info("add_device done: user=" .. tostring(username) .. " device=" .. tostring(device) .. " ip=" .. tostring(user_ip))
  return true
end

local function delete_user(cfg, state, username)
  log_info("delete_user start: username=" .. tostring(username))
  username = string.lower(username or "")
  if state.datum.Users[username] then
    if cfg.DryRun then
      say("Dry-run: user '" .. username .. "' and all VPN devices would be deleted. No brain state, WireGuard configuration, or host service changes were made.")
      log_info("delete_user dry-run done: username=" .. tostring(username))
      return
    end
    state.datum.Users[username] = nil
    if ensure_valid_wireguard_state(state, true) and update_state(state) and apply_wireguard(cfg, state) then
      say("User '" .. username .. "' deleted successfully.")
      log_info("delete_user done: username=" .. tostring(username))
    end
  else
    log_warn("delete_user not found: username=" .. tostring(username))
    say("User '" .. username .. "' not found.")
  end
end

local function delete_device(cfg, state, device)
  local username = bot.user
  log_info("delete_device start: user=" .. tostring(username) .. " device=" .. tostring(device))
  device = string.lower(device or "")
  if state.datum.Users[username] and state.datum.Users[username][device] then
    if cfg.DryRun then
      say("Dry-run: device '" .. device .. "' for user '" .. tostring(username) .. "' would be deleted. No brain state, WireGuard configuration, or host service changes were made.")
      log_info("delete_device dry-run done: user=" .. tostring(username) .. " device=" .. tostring(device))
      return
    end
    state.datum.Users[username][device] = nil
    if ensure_valid_wireguard_state(state, true) and update_state(state) and apply_wireguard(cfg, state) then
      say("Device '" .. device .. "' deleted successfully.")
      log_info("delete_device done: user=" .. tostring(username) .. " device=" .. tostring(device))
    end
  else
    log_warn("delete_device not found: user=" .. tostring(username) .. " device=" .. tostring(device))
    say("Device '" .. device .. "' not found for user '" .. tostring(username) .. "'.")
  end
end

local function list_users(state)
  local rows = {}
  for _, username in ipairs(sorted_keys(state.datum.Users)) do
    table.insert(rows, username .. ": " .. table.concat(sorted_keys(state.datum.Users[username]), ", "))
  end
  if #rows == 0 then
    say("No Users Found.")
  else
    say("\n" .. table.concat(rows, "\n"))
  end
end

local function list_devices(state)
  local username = bot.user
  if state.datum.Users[username] then
    local devices = sorted_keys(state.datum.Users[username])
    say("Device(s) for user '" .. username .. "': " .. table.concat(devices, ", "))
  else
    say("No devices found for user '" .. tostring(username) .. "'")
  end
end

local function get_vpn(cfg, state, device)
  local username = bot.user
  log_info("get_vpn start: user=" .. tostring(username) .. " device=" .. tostring(device))
  device = string.lower(device or "")
  if not state.datum.Users[username] or not state.datum.Users[username][device] then
    log_warn("get_vpn not found: user=" .. tostring(username) .. " device=" .. tostring(device))
    say("Device '" .. device .. "' not found for user '" .. tostring(username) .. "'")
    return
  end
  local data = state.datum.Users[username][device]
  local ip = endpoint_ip(cfg)
  say("VPN config data: Robot_IP = " .. ip .. ":" .. tostring(cfg.ListenPort) .. " | USER_IP = " .. data.AllowedIPs .. " | PSK = " .. data.PreSharedKey)
  log_info("get_vpn done: user=" .. tostring(username) .. " device=" .. tostring(device) .. " ip=" .. tostring(data.AllowedIPs))
end

local function get_vpn_info(cfg)
  if not cfg.PublicKey or cfg.PublicKey == "" then
    log_error("get_vpn_info failed: missing public key")
    say("WireGuard public key is not configured")
    return
  end
  local ip = endpoint_ip(cfg)
  say("WireGuard VPN info:\nPublic key: " .. tostring(cfg.PublicKey) .. "\nEndpoint: " .. ip .. ":" .. tostring(cfg.ListenPort))
end

local function allow_ip(cfg, address)
  if not is_global_ipv4(address) then
    log_warn("allow_ip rejected: address=" .. tostring(address))
    say("Invalid, unparseable, or non-public IP address")
    return
  end
  if cfg.DryRun then
    log_info("allow_ip skipped: development dry-run address=" .. tostring(address))
    say("Dry-run: IP address " .. tostring(address) .. " would be added to ALLOW_VPN. No iptables changes were made.")
    return
  end
  local ok, out, status = shell_capture("sudo -n iptables -L ALLOW_VPN -n", "iptables list ALLOW_VPN")
  if not ok then
    log_error("allow_ip failed: iptables list status=" .. tostring(status) .. " output=" .. summarize_output(out))
    say("Unable to inspect ALLOW_VPN firewall chain")
    return
  end
  for line in out:gmatch("[^\n]+") do
    if line:match("^ACCEPT") and line:find(address, 1, true) then
      log_info("allow_ip already allowed: address=" .. tostring(address))
      say("IP already allowed")
      return
    end
  end
  ok, out, status = shell_capture("sudo -n iptables -A ALLOW_VPN -s " .. shell_quote(address) .. " -j ACCEPT", "iptables append ALLOW_VPN")
  if ok then
    log_info("allow_ip done: address=" .. tostring(address))
    say("IP address added")
  else
    log_error("allow_ip failed: iptables append status=" .. tostring(status) .. " output=" .. summarize_output(out))
    say("Unable to add IP address")
  end
end

log_info("command start: command=" .. tostring(command) ..
  " environment=" .. tostring(environment) ..
  " dry_run=" .. tostring(dry_run) ..
  " user=" .. tostring(bot.user) ..
  " channel=" .. tostring(bot.channel) ..
  " arg_count=" .. tostring(#arg))

local cfg = load_config()
if not cfg then
  log_error("command abort: config unavailable")
  return task.Fail
end

if command == "allow-ip" then
  allow_ip(cfg, arg[2])
  return task.Normal
end

if command == "get-vpn-info" then
  get_vpn_info(cfg)
  return task.Normal
end

local write_commands = {
  ["add-device"] = true,
  ["admin-delete-vpn-user"] = true,
  ["delete-device"] = true,
  ["clear-vpn"] = true,
}

local state = checkout_state(write_commands[command] == true and not dry_run)
if not state then
  log_error("command abort: state unavailable")
  return task.Fail
end

if command == "_init" then
  apply_wireguard(cfg, state)
elseif command == "add-device" then
  add_device(cfg, state, arg[2], arg[3])
elseif command == "admin-list-vpn-users" then
  list_users(state)
elseif command == "admin-delete-vpn-user" then
  delete_user(cfg, state, arg[2])
elseif command == "list-vpn-devices" then
  list_devices(state)
elseif command == "delete-device" then
  delete_device(cfg, state, arg[2])
elseif command == "get-vpn" then
  get_vpn(cfg, state, arg[2])
elseif command == "clear-vpn" then
  if cfg.DryRun then
    say("Dry-run: all VPN users and devices would be cleared. No brain state, WireGuard configuration, host service, or iptables changes were made.")
    log_info("clear-vpn dry-run done")
  else
    state.datum.Users = {}
    if ensure_valid_wireguard_state(state, true) and update_state(state) and apply_wireguard(cfg, state) then
      shell_capture("sudo -n iptables -F ALLOW_VPN", "iptables flush ALLOW_VPN")
      say("Cleared all VPN users and devices, and emptied the ALLOW_VPN chain")
    end
  end
end

log_info("command done: command=" .. tostring(command))
return task.Normal
