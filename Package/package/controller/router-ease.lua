module("luci.controller.router-ease", package.seeall)

local uci = require "luci.model.uci".cursor()
local util = require "luci.util"
local jsonc = require "luci.jsonc"
local sys = require "luci.sys"
local fs = require "nixio.fs"
local nixio = require "nixio"

function index()
    -- Main page entry
    local page = entry({"admin", "router-ease"}, firstchild(), "Router-Ease", 60)
    page.dependent = false
    page.sysauth = "admin"

    -- Visible menu items
    entry({"admin", "router-ease", "dashboard"}, template("router-ease/dashboard"), "Connected Devices", 5)
    entry({"admin", "router-ease", "network"}, firstchild(), "Network Tools", 10)
    entry({"admin", "router-ease", "network", "speedtest"}, template("router-ease/speed-test"), "Speed Test", 10)
    entry({"admin", "router-ease", "network", "qrcode"}, template("router-ease/qr"), "WiFi QR Code", 20)
    entry({"admin", "router-ease", "qos"}, cbi("nft-qos/nft-qos"), _("Quality of Service"), 30)
    entry({"admin", "router-ease", "bandwidth"}, view("nlbw/display"), _("Bandwidth Monitor"), 35)
    entry({"admin", "router-ease", "settings"}, template("router-ease/settings"), "Settings", 40)

    -- Sub-pages for bandwidth monitor (hidden from main menu)
    entry({"admin", "router-ease", "bandwidth", "config"}, view("nlbw/config"), nil)
    entry({"admin", "router-ease", "bandwidth", "backup"}, view("nlbw/backup"), nil)
-- In the index() function:
    entry({"admin", "network", "speedtest", "run"}, call("action_run_speedtest")).leaf = true
    entry({"admin", "network", "speedtest", "status"}, call("action_speedtest_status")).leaf = true
    -- Action endpoints (API calls, no UI)
    entry({"admin", "router-ease", "get_wifi_info"}, call("get_wifi_info")).leaf = true
    entry({"admin", "router-ease", "configure_doh"}, call("configure_doh")).leaf = true
    entry({"admin", "router-ease", "get_connected_devices"}, call("get_connected_devices")).leaf = true
    entry({"admin", "router-ease", "qos_status"}, call("qos_status")).leaf = true
    entry({"admin", "router-ease", "speedtest_run"}, call("action_run_speedtest")).leaf = true
    entry({"admin", "router-ease", "speedtest_status"}, call("action_speedtest_status")).leaf = true
end

function qos_status()
    local status = util.trim(sys.exec("nft list ruleset | grep -q 'qos' && echo on || echo off"))
    luci.http.prepare_content("text/plain")
    luci.http.write(status)
end

function get_wifi_info()
    local result = {}
    uci:foreach("wireless", "wifi-iface", function(s)
        if s.mode == "ap" and s.network == "lan" then
            result.ssid = s.ssid or ""
            result.key = s.key or ""
            result.encryption = s.encryption or "none"
            return false -- Stop after finding the first AP
        end
    end)
    luci.http.prepare_content("application/json")
    luci.http.write(jsonc.stringify(result))
end

function get_connected_devices()
    local result = {}
    local devices = {}
    local hostnames = {}

    -- MAC address manufacturer lookup cache to avoid redundant requests
    local mac_manufacturer_cache = {}

    local function get_manufacturer(mac)
        if not mac then return nil end

        local upper_mac = string.upper(mac)

        -- Check cache first to avoid redundant requests
        if mac_manufacturer_cache[upper_mac] then
            return mac_manufacturer_cache[upper_mac]
        end

        -- The URL for the manufacturer lookup service
        local url = "https://augustomancuso.com/routerease/mac-address/?mac=" .. upper_mac

        -- Execute the curl command with proper quoting and error handling
        local curl_cmd = 'curl -s -m 5 "' .. url .. '"'
        local resp = util.exec(curl_cmd)

        -- Debug output
        nixio.syslog("info", "MAC lookup for " .. upper_mac .. ": " .. (resp or "no response"))

        -- Process the response
        if resp and #resp > 0 then
            local ok, data = pcall(jsonc.parse, resp)
            if ok and data and data.manufacturer then
                mac_manufacturer_cache[upper_mac] = data.manufacturer
                return data.manufacturer
            else
                nixio.syslog("err", "Failed to parse JSON: " .. tostring(resp))
            end
        end

        mac_manufacturer_cache[upper_mac] = nil
        return nil
    end

    uci:foreach("dhcp", "host", function(s)
        if s.mac and s.name then
            hostnames[string.upper(s.mac)] = s.name
        end
    end)

    local dhcp_leases = util.exec("cat /tmp/dhcp.leases") or ""
    for mac, ip, name in dhcp_leases:gmatch("(%S+) (%S+) (%S+)") do
        if name ~= "*" then
            hostnames[string.upper(mac)] = name
        end
    end

    local arp_scan_result = util.exec("arp-scan 192.168.16.0/24 -xg 2>/dev/null") or ""
    for line in arp_scan_result:gmatch("[^\r\n]+") do
        if line:match("^%d+%.%d+%.%d+%.%d+") then
            local ip, mac, vendor = line:match("(%d+%.%d+%.%d+%.%d+)%s+([0-9a-fA-F:]+)%s+(.*)")
            if ip and mac and mac:match("^[0-9a-fA-F:]+$") and mac ~= "00:00:00:00:00:00" then
                local upper_mac = string.upper(mac)
                local hostname = hostnames[upper_mac] or "Unknown"
                local manufacturer = get_manufacturer(upper_mac)
                local dev_info = {
                    ip = ip, mac = upper_mac, interface = "unknown", hostname = hostname,
                    manufacturer = manufacturer or "Unknown", connection_type = "unknown",
                    signal = nil, rx_bytes = 0, tx_bytes = 0, last_seen = os.time()
                }
                dev_info.display_name = (hostname == "Unknown") and (manufacturer or "Unknown") or hostname
                devices[upper_mac] = dev_info
            end
        end
    end

    -- Process ARP table entries
    local arp_table = fs.readfile("/proc/net/arp") or ""
    for ip, mac, device in arp_table:gmatch("(%d+%.%d+%.%d+%.%d+)%s+%S+%s+%S+%s+(%S+)%s+%S+%s+(%S+)") do
        if mac:match("^[0-9a-fA-F:]+$") and mac ~= "00:00:00:00:00:00" then
            local upper_mac = string.upper(mac)
            if devices[upper_mac] then
                devices[upper_mac].interface = device
                devices[upper_mac].connection_type = "wired"
            else
                local hostname = hostnames[upper_mac] or "Unknown"
                local manufacturer = get_manufacturer(upper_mac)
                local dev_info = {
                    ip = ip, mac = upper_mac, interface = device, hostname = hostname,
                    manufacturer = manufacturer or "Unknown", connection_type = "wired",
                    signal = nil, rx_bytes = 0, tx_bytes = 0, last_seen = os.time()
                }
                dev_info.display_name = (hostname == "Unknown") and (manufacturer or "Unknown") or hostname
                devices[upper_mac] = dev_info
            end
        end
    end

    -- Get wireless client information
    local iw_output = util.exec("iwinfo | grep -A 5 'ESSID\\|Associated'") or ""
    local current_iface, current_essid
    for line in iw_output:gmatch("[^\r\n]+") do
        if line:match("ESSID:") then
            current_iface = line:match("^(.-) ")
            current_essid = line:match('ESSID: "(.-)"')
        elseif line:match("Associated") then
            local mac = line:match("([0-9A-F:]+)")
            if mac and devices[string.upper(mac)] then
                local dev = devices[string.upper(mac)]
                dev.connection_type = "wifi"
                dev.essid = current_essid
                dev.interface = current_iface
                dev.signal = tonumber(line:match("Signal: ([%-0-9]+)")) or 0
            end
        end
    end

    -- Build final result array
    for _, device in pairs(devices) do
        table.insert(result, device)
    end

    luci.http.prepare_content("application/json")
    luci.http.write(jsonc.stringify(result))
end

function action_run_speedtest()


    -- Run speedtest-cli directly and get the output
    local raw_output = util.exec("speedtest-cli --json")

    -- Check if we got valid output
    if not raw_output or #raw_output == 0 then
        luci.http.prepare_content("application/json")
        luci.http.write_json({
            status = "error",
            message = "Failed to run speedtest. No output returned."
        })
        return
    end

    -- Parse the JSON output
    local ok, data = pcall(jsonc.parse, raw_output)
    if not ok or not data then
        luci.http.prepare_content("application/json")
        luci.http.write_json({
            status = "error",
            message = "Failed to parse speedtest results"
        })
        return
    end

    -- Format and return the results
    local result = {
        status = "complete",
        data = {
            ping = {median = data.ping or 0},
            download = {bps_mean = data.download or 0},
            upload = {bps_mean = data.upload or 0}
        }
    }
    luci.http.prepare_content("application/json")
    luci.http.write_json(result)
end



-- function configure_doh()
--     local body = nixio.stdin:read(-1)
--     local response = { success = false }
--
--     if not body or #util.trim(body) == 0 then
--         luci.http.status(400, "Bad Request")
--         response.message = "Request body is empty. Please provide a token."
--         luci.http.prepare_content("application/json")
--         luci.http.write(jsonc.stringify(response))
--         return
--     end
--
--     local token = util.trim(body)
--     local doh_url = "https://augustomancuso.com/routerease/dns/dns-query"
--     local proxy_listen_addr = "127.0.0.1"
--     local proxy_listen_port = "5055"
--     local conf_path = "/tmp/dnsmasq.doh.conf"
--
--     -- Kill previous instances to avoid conflicts
--     util.exec("pkill -f 'https-dns-proxy.*" .. proxy_listen_port .. "'")
--     util.exec("pkill -f 'dnsmasq.*" .. conf_path .. "'")
--
--     -- Start https-dns-proxy with the authentication token
--     local https_dns_cmd = string.format(
--         "https-dns-proxy -a %s -p %s -r '%s' -H 'Authorization: Bearer %s' --no-daemon &",
--         proxy_listen_addr, proxy_listen_port, doh_url, token
--     )
--     util.exec(https_dns_cmd)
--
--     -- Write dnsmasq config to use https-dns-proxy as upstream
--     local conf = string.format(
--         "listen-address=127.0.0.1\nport=53\nno-resolv\nserver=%s#%s\n",
--         proxy_listen_addr, proxy_listen_port
--     )
--
--     local ok = nixio.fs.writefile(conf_path, conf)
--     if not ok then
--         response.message = "Failed to write dnsmasq config."
--         luci.http.prepare_content("application/json")
--         luci.http.write(jsonc.stringify(response))
--         return
--     end
--
--     -- Start dnsmasq with the new configuration
--     local start_cmd = "dnsmasq --conf-file=" .. conf_path .. " --no-daemon &"
--     util.exec(start_cmd)
--
--     response.success = true
--     response.message = "dnsmasq and https-dns-proxy started. DNS is now served over HTTPS."
--
--     luci.http.prepare_content("application/json")
--     luci.http.write(jsonc.stringify(response))
-- end
--
-- function configure_doh()
--     local body = nixio.stdin:read(-1)
--     local response = { success = false }
--
--     if not body or #util.trim(body) == 0 then
--         luci.http.status(400, "Bad Request")
--         response.message = "Request body is empty. Please provide a token."
--         luci.http.prepare_content("application/json")
--         luci.http.write(jsonc.stringify(response))
--         return
--     end
--
--     -- Write dnsmasq config to /tmp/dnsmasq.conf
--     local conf = "listen-address=127.0.0.1\nport=53\n"
--     local conf_path = "/tmp/dnsmasq.conf"
--     local ok = nixio.fs.writefile(conf_path, conf)
--
--     if not ok then
--         response.message = "Failed to write dnsmasq config."
--         luci.http.prepare_content("application/json")
--         luci.http.write(jsonc.stringify(response))
--         return
--     end
--
--     util.exec("pkill -f 'dnsmasq.*" .. conf_path .. "'")
--     local start_cmd = "dnsmasq --conf-file=" .. conf_path .. " --no-daemon &"
--     util.exec(start_cmd)
--
--     response.success = true
--     response.message = "dnsmasq started on 127.0.0.1:53"
--
--     luci.http.prepare_content("application/json")
--     luci.http.write(jsonc.stringify(response))
-- end

function start_https_dns_proxy(token)
    -- Configuration values
    local custom_doh_url = "https://augustomancuso.com/routerease/dns/dns-query?dns="
    local proxy_port = "5053"
    local proxy_address = "127.0.0.1#" .. proxy_port
    local ok, err = pcall(function()
        local uci = luci.model.uci.cursor()

        -- 1. Clean up all existing https-dns-proxy configurations
            while uci:delete("https-dns-proxy", "@https-dns-proxy[0]") do
                -- This loop removes all anonymous sections until none are left
            end
        sys.exec("/etc/init.d/https-dns-proxy restart >/dev/null 2>&1")

        -- 2. Create and configure a single new https-dns-proxy instance for GET requests
        uci:section("https-dns-proxy", "https-dns-proxy", nil, {
            resolver_url = custom_doh_url,
            extra_headers = "Authorization: Bearer " .. token,
            bootstrap_dns = "8.8.8.8", -- Use a public DNS for bootstrapping
            listen_addr = "127.0.0.1",
            listen_port = proxy_port,
        })
        uci:commit("https-dns-proxy")
        sys.exec("/etc/init.d/https-dns-proxy restart >/dev/null 2>&1")
    end)

    -- Verify the service started (with a short delay to allow startup)
    util.exec("sleep 5")
    local is_running = (util.exec("ps | grep https-dns-proxy | grep -v grep") ~= "")
    return is_running
end


function configure_doh()
    local body = nixio.stdin:read(-1)
    local response = { success = false }

    if not body or #util.trim(body) == 0 then
        luci.http.status(400, "Bad Request")
        response.message = "Request body is empty. Please provide a token."
        luci.http.prepare_content("application/json")
        luci.http.write(jsonc.stringify(response))
        return
    end

    local token = util.trim(body)
    local success = start_https_dns_proxy(token)
    if not success then
        response.message = "Failed to start https-dns-proxy"
        luci.http.prepare_content("application/json")
        luci.http.write(jsonc.stringify(response))
        return
    end

    -- Configure dnsmasq to use ONLY the https-dns-proxy
    local dnsmasq_conf = [[
    listen-address=127.0.0.1
    port=53
    no-resolv
    no-poll
    server=127.0.0.1#5053
    domain-needed
    bogus-priv
    cache-size=1000
    neg-ttl=60
    ]]

    local conf_path = "/tmp/dnsmasq.doh.conf"
    local ok = nixio.fs.writefile(conf_path, dnsmasq_conf)
    util.exec("killall dnsmasq 2>/dev/null")
    util.exec("sleep 1")
    if not ok then
        response.message = "Failed to write dnsmasq config."
        luci.http.prepare_content("application/json")
        luci.http.write(jsonc.stringify(response))
        return
    end

    local start_cmd = "dnsmasq --conf-file=" .. conf_path .. " --no-daemon &"
    util.exec(start_cmd)

    response.success = true
    response.message = "dnsmasq and https-dns-proxy started. DNS is now served over HTTPS."

    luci.http.prepare_content("application/json")
    luci.http.write(jsonc.stringify(response))
end