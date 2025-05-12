module("luci.controller.router-ease", package.seeall)

function index()
     entry({"admin", "router-ease"}, firstchild(), "Router-Ease", 60).dependent=false
     entry({"admin", "router-ease", "network"}, firstchild(), "Network Tools", 10)

     -- Add both features as submenu items under Network Tools
     entry({"admin", "router-ease", "network", "speedtest"}, template("router-ease/speed-test"), "Speed Test", 10)
     entry({"admin", "router-ease", "network", "qrcode"}, template("router-ease/qr"), "WiFi QR Code", 20)
     entry({"admin", "router-ease", "dashboard"}, template("router-ease/dashboard"), "Connected Devices", 5)

     -- QoS integration - direct method, not iframe
     entry({"admin", "router-ease", "qos"}, cbi("nft-qos/nft-qos"), _("Quality of Service"), 30)

     -- API endpoints for features
     entry({"admin", "network", "speedtest", "action_run_speedtest"}, call("action_run_speedtest"), nil).leaf = true
     entry({"admin", "network", "speedtest", "action_status"}, call("action_status"), nil).leaf = true
     entry({"admin", "router-ease", "get_wifi_info"}, call("get_wifi_info"), nil).leaf = true
     entry({"admin", "router-ease", "get_connected_devices"}, call("get_connected_devices"), nil).leaf = true
     entry({"admin", "router-ease", "kick_device"}, call("action_kick_device"), nil).leaf = true
     entry({"admin", "router-ease", "qos_status"}, call("qos_status"))
end

local function qos_status()
    local sys = require "luci.sys"
    local util = require "luci.util"

    -- Get QoS status
    local status = util.trim(util.exec("nft list ruleset | grep -q 'qos' && echo on || echo off"))
    return status
end

-- Rest of the functions remain the same...

function action_run_speedtest()
    local sys = require "luci.sys"
    local fs = require "nixio.fs"

    -- Clear old results if they exist
    if fs.access("/tmp/speedtest_results.txt") then
        fs.unlink("/tmp/speedtest_results.txt")
    end

    -- Run speedtest-netperf.sh and redirect output to a file
    sys.call("which speedtest-netperf.sh > /tmp/speedtest_path.txt")
    sys.call("speedtest-netperf.sh  -H 79.127.209.1 -t 30 > /tmp/speedtest_results.txt 2>&1 &")

    luci.http.prepare_content("application/json")
    luci.http.write_json({ status = "running" })
end

function parse_speedtest_results(results)
    local download = 0
    local upload = 0
    local ping = 0

    -- Log raw results for debugging
    local debug_log = io.open("/tmp/speedtest_parse_debug.log", "w")
    if debug_log then
        debug_log:write("Raw results:\n" .. results .. "\n\n")
    end

    -- Try multiple patterns for download speed
    local download_patterns = {
        "Download:.-\n%s*([%d%.]+)%s*Mbps",
        "Download[^\n]*\n%s*([%d%.]+)"  -- More generic pattern
    }

    for _, pattern in ipairs(download_patterns) do
        local match = results:match(pattern)
        if match and match ~= "0.00" then
            download = tonumber(match) or 0
            if debug_log then debug_log:write("Download matched: " .. match .. "\n") end
            break
        end
    end

    -- Try multiple patterns for upload speed
    local upload_patterns = {
        "Upload:.-\n%s*([%d%.]+)%s*Mbps",
        "Upload[^\n]*\n%s*([%d%.]+)"  -- More generic pattern
    }

    for _, pattern in ipairs(upload_patterns) do
        local match = results:match(pattern)
        if match and match ~= "0.00" then
            upload = tonumber(match) or 0
            if debug_log then debug_log:write("Upload matched: " .. match .. "\n") end
            break
        end
    end

    -- Extract ping values
    if results:match("Median:%s*([%d%.]+)") then
        ping = tonumber(results:match("Median:%s*([%d%.]+)")) or 0
    elseif results:match("Avg:%s*([%d%.]+)") then
        ping = tonumber(results:match("Avg:%s*([%d%.]+)")) or 0
    end

    if debug_log then
        debug_log:write("Final values: Download=" .. download .. ", Upload=" .. upload .. ", Ping=" .. ping .. "\n")
        debug_log:close()
    end

    -- If we have errors in the results, report them in the response
    local error_msg = nil
    if results:match("WARNING:") or results:match("invalid number") then
        error_msg = "Test completed with warnings. Results may be inaccurate."
    end

    return download, upload, ping, error_msg
end

function action_status()
    local fs = require "nixio.fs"
    local sys = require "luci.sys"

    luci.http.prepare_content("application/json")

    -- Check if speedtest is still running
    local running = (sys.exec("pgrep -f speedtest-netperf.sh") ~= "")

    if running then
        luci.http.write_json({ status = "running" })
        return
    end

    -- Check for results file
    if fs.access("/tmp/speedtest_results.txt") then
        local results = fs.readfile("/tmp/speedtest_results.txt")

        -- Parse the results
        local download, upload, ping, error_msg = parse_speedtest_results(results)

        local response = {
            status = "complete",
            data = {
                download = { bps_mean = download * 1000000 },
                upload = { bps_mean = upload * 1000000 },
                ping = { median = ping }
            }
        }

        if error_msg then
            response.warning = error_msg
        end

        luci.http.write_json(response)
    else
        luci.http.write_json({
            status = "error",
            message = "No test results found"
        })
    end
end

--- QR Code Functionality
function get_wifi_info()
    local uci = require "luci.model.uci".cursor()
    local json = require "luci.jsonc"
    local result = {}

    -- Get primary WiFi network info
    uci:foreach("wireless", "wifi-iface", function(s)
        if s.mode == "ap" and s.network == "lan" then
            result.ssid = s.ssid or ""
            result.key = s.key or ""
            result.encryption = s.encryption or "none"
            return false  -- Stop after finding the first AP
        end
    end)

    luci.http.prepare_content("application/json")
    luci.http.write(json.stringify(result))
end


--- Connected devices


--- Connected Devices

function get_connected_devices()
    local sys = require "luci.sys"
    local util = require "luci.util"
    local uci = require "luci.model.uci".cursor()
    local json = require "luci.jsonc"
    local nixio = require "nixio"

    local result = {}
    local devices = {}
    local hostnames = {}

    -- Get DHCP leases for hostname mapping
    uci:foreach("dhcp", "host", function(s)
        if s.mac and s.name then
            hostnames[string.upper(s.mac)] = s.name
        end
    end)

    -- Get DHCP leases for active client
    local dhcp_leases = util.exec("cat /tmp/dhcp.leases") or ""
    for mac, ip, name in dhcp_leases:gmatch("(%S+) (%S+) (%S+)") do
        if name ~= "*" then
            hostnames[string.upper(mac)] = name
        end
    end

    -- Get ARP table for all connected devices
    local arp_table = nixio.fs.readfile("/proc/net/arp") or ""
    for ip, mac, device in arp_table:gmatch("(%d+%.%d+%.%d+%.%d+)%s+%S+%s+%S+%s+(%S+)%s+%S+%s+(%S+)") do
        if mac:match("^[0-9a-fA-F:]+$") and mac ~= "00:00:00:00:00:00" then
            local dev_info = {
                ip = ip,
                mac = string.upper(mac),
                interface = device,
                hostname = hostnames[string.upper(mac)] or "Unknown",
                connection_type = "wired",
                signal = nil,
                rx_bytes = 0,
                tx_bytes = 0,
                last_seen = os.time()
            }
            devices[string.upper(mac)] = dev_info
        end
    end

    -- Get wireless clients
    local iw_output = util.exec("iwinfo | grep -A 5 'ESSID\\|Associated'") or ""
    local current_iface = nil
    local current_essid = nil

    for line in iw_output:gmatch("[^\r\n]+") do
        if line:match("ESSID:") then
            current_iface = line:match("^(.-) ") or ""
            current_essid = line:match("ESSID: \"(.-)\"") or "Unknown"
        elseif line:match("Associated") then
            local mac = line:match("([0-9A-F:]+)") or ""
            if mac ~= "" and devices[string.upper(mac)] then
                devices[string.upper(mac)].connection_type = "wifi"
                devices[string.upper(mac)].essid = current_essid
                devices[string.upper(mac)].interface = current_iface

                -- Get signal strength
                local signal = line:match("Signal: ([%-0-9]+)") or "0"
                devices[string.upper(mac)].signal = tonumber(signal) or 0
            end
        end
    end

    -- Get traffic statistics from bandwidth monitoring
--[[     local bw_stats = util.exec("cat /tmp/nlbwmon.db 2>/dev/null") or ""
    for mac, rx, tx in bw_stats:gmatch("mac=([0-9A-F:]+).-rx_bytes=(%d+).-tx_bytes=(%d+)") do
        if devices[string.upper(mac)] then
            devices[string.upper(mac)].rx_bytes = tonumber(rx) or 0
            devices[string.upper(mac)].tx_bytes = tonumber(tx) or 0
        end
    end ]]

    -- Convert to array
    for _, device in pairs(devices) do
        table.insert(result, device)
    end

    luci.http.prepare_content("application/json")
    luci.http.write(json.stringify(result))
end

--- Kick devices from the LAN
function action_kick_device()
    local http = require "luci.http"
    local util = require "luci.util"
    local json = require "luci.jsonc"

    -- Parse input parameters
    local params = http.content()
    local input = json.parse(params)

    if not input or not input.mac then
        http.status(400, "Bad Request")
        http.prepare_content("application/json")
        http.write_json({success = false, message = "Missing MAC address"})
        return
    end

    local mac = input.mac:upper()
    local connection_type = input.connection_type or "unknown"

    -- Response to return
    local result = {success = false, message = "Failed to kick device"}

    -- Handle wireless devices
    if connection_type == "wifi" then
        -- Get all wireless interfaces
        local ifaces = util.exec("iwinfo | grep -E '^[a-z0-9]+'"):gsub("\n$", ""):split("\n")

        -- Try to kick from each interface until successful
        for _, iface in ipairs(ifaces) do
            local cmd = "hostapd_cli -i " .. iface .. " deauth " .. mac .. " 2>&1"
            local res = util.exec(cmd)

            if res:match("OK") then
                result.success = true
                result.message = "Device kicked from wireless network"
                break
            end
        end
    else
        -- For wired devices, block using firewall
        local blocktime = 60  -- Block for 60 seconds

        -- Create temporary firewall rule to block the MAC
        util.exec("iptables -I FORWARD -m mac --mac-source " .. mac .. " -j DROP")

        -- Schedule removal of the rule after blocktime
        util.exec("(sleep " .. blocktime .. " && iptables -D FORWARD -m mac --mac-source " .. mac .. " -j DROP) &")

        result.success = true
        result.message = "Device blocked for " .. blocktime .. " seconds"
    end

    http.prepare_content("application/json")
    http.write_json(result)
end

