module("luci.controller.router-ease", package.seeall)

local dispatcher = require "luci.dispatcher"

function index()
     -- Add authentication requirement to fix the visibility issue
     local page = entry({"admin", "router-ease"}, firstchild(), "Router-Ease", 60)
     page.dependent = false
     page.sysauth = "admin"  -- Fix for authentication issue

     entry({"admin", "router-ease", "network"}, firstchild(), "Network Tools", 10)

     -- Add both features as submenu items under Network Tools
     entry({"admin", "router-ease", "network", "speedtest"}, template("router-ease/speed-test"), "Speed Test", 10)
     entry({"admin", "router-ease", "network", "qrcode"}, template("router-ease/qr"), "WiFi QR Code", 20)
     entry({"admin", "router-ease", "dashboard"}, template("router-ease/dashboard"), "Connected Devices", 5)

     -- Keep existing QoS integration
-- Quality of Service (working correctly)
     entry({"admin", "router-ease", "qos"}, cbi("nft-qos/nft-qos"), _("Quality of Service"), 30)

-- Main bandwidth monitoring page
     entry({"admin", "router-ease", "bandwidth"}, view("nlbw/display"), _("Bandwidth Monitor"), 35)

     -- Sub-pages if desired (optional)
     entry({"admin", "router-ease", "bandwidth", "config"}, view("nlbw/config"), _("Configuration"))
     entry({"admin", "router-ease", "bandwidth", "backup"}, view("nlbw/backup"), _("Backup"))
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
    sys.call("speedtest-netperf.sh  -H 79.127.209.1 -t 20 > /tmp/speedtest_results.txt 2>&1 &")

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

function get_connected_devices()
    local sys = require "luci.sys"
    local util = require "luci.util"
    local uci = require "luci.model.uci".cursor()
    local json = require "luci.jsonc"
    local nixio = require "nixio"

    local result = {}
    local devices = {}
    local hostnames = {}

    -- Load manufacturer database as fallback
    local manufacturers = {}
    if nixio.fs.access("/usr/share/arp-scan/ieee-oui.txt") then
        for line in io.lines("/usr/share/arp-scan/ieee-oui.txt") do
            local mac_prefix, manufacturer = line:match("^([0-9A-F]+)%s+(.+)$")
            if mac_prefix then
                manufacturers[mac_prefix] = manufacturer
            end
        end
    end

    -- Function to get manufacturer from MAC
    local function get_manufacturer(mac)
        if not mac then return "Unknown" end
        local prefix = mac:gsub(":", ""):sub(1, 6):upper()
        return manufacturers[prefix] or "Unknown"
    end

    -- Get DHCP leases for hostname mapping
    uci:foreach("dhcp", "host", function(s)
        if s.mac and s.name then
            hostnames[string.upper(s.mac)] = s.name
        end
    end)

    -- Get DHCP leases for active clients
    local dhcp_leases = util.exec("cat /tmp/dhcp.leases") or ""
    for mac, ip, name in dhcp_leases:gmatch("(%S+) (%S+) (%S+)") do
        if name ~= "*" then
            hostnames[string.upper(mac)] = name
        end
    end

    -- Run arp-scan with fallback options to handle missing IP address
    local arp_scan_cmd = "arp-scan 192.168.16.0/24 -xg 2>/dev/null"
    local arp_scan_result = util.exec(arp_scan_cmd) or ""

    -- Process arp-scan results
    for line in arp_scan_result:gmatch("[^\r\n]+") do
        -- Skip header lines
        if line:match("^%d+%.%d+%.%d+%.%d+") then
            local ip, mac, vendor = line:match("(%d+%.%d+%.%d+%.%d+)%s+([0-9a-fA-F:]+)%s+(.*)")
            if ip and mac and mac:match("^[0-9a-fA-F:]+$") and mac ~= "00:00:00:00:00:00" then
                local upper_mac = string.upper(mac)

                -- Use vendor from arp-scan output or fallback to database lookup
                local manufacturer = vendor
                if not manufacturer or manufacturer:match("Unknown") then
                    manufacturer = get_manufacturer(upper_mac)
                end

                local dev_info = {
                    ip = ip,
                    mac = upper_mac,
                    interface = "unknown", -- Will be updated later if found in other sources
                    hostname = hostnames[upper_mac] or "Unknown",
                    manufacturer = manufacturer,
                    connection_type = "unknown", -- Will be updated later
                    signal = nil,
                    rx_bytes = 0,
                    tx_bytes = 0,
                    last_seen = os.time()
                }

                -- Fallback name when hostname is unknown
                if dev_info.hostname == "Unknown" then
                    dev_info.display_name = manufacturer .. " Device"
                else
                    dev_info.display_name = dev_info.hostname
                end

                devices[upper_mac] = dev_info
            end
        end
    end

    -- Get ARP table for additional connected devices and update existing ones
    local arp_table = nixio.fs.readfile("/proc/net/arp") or ""
    for ip, mac, device in arp_table:gmatch("(%d+%.%d+%.%d+%.%d+)%s+%S+%s+%S+%s+(%S+)%s+%S+%s+(%S+)") do
        if mac:match("^[0-9a-fA-F:]+$") and mac ~= "00:00:00:00:00:00" then
            local upper_mac = string.upper(mac)

            if devices[upper_mac] then
                -- Update existing device with additional info
                devices[upper_mac].interface = device
                devices[upper_mac].connection_type = "wired"
            else
                -- Create new device entry
                local manufacturer = get_manufacturer(upper_mac)

                local dev_info = {
                    ip = ip,
                    mac = upper_mac,
                    interface = device,
                    hostname = hostnames[upper_mac] or "Unknown",
                    manufacturer = manufacturer,
                    connection_type = "wired",
                    signal = nil,
                    rx_bytes = 0,
                    tx_bytes = 0,
                    last_seen = os.time()
                }

                -- Fallback name when hostname is unknown
                if dev_info.hostname == "Unknown" then
                    dev_info.display_name = manufacturer .. " Device"
                else
                    dev_info.display_name = dev_info.hostname
                end

                devices[upper_mac] = dev_info
            end
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
            if mac ~= "" then
                local upper_mac = string.upper(mac)
                if devices[upper_mac] then
                    -- Update existing device with wifi info
                    devices[upper_mac].connection_type = "wifi"
                    devices[upper_mac].essid = current_essid
                    devices[upper_mac].interface = current_iface

                    -- Get signal strength
                    local signal = line:match("Signal: ([%-0-9]+)") or "0"
                    devices[upper_mac].signal = tonumber(signal) or 0
                end
            end
        end
    end

    -- Convert to array
    for _, device in pairs(devices) do
        table.insert(result, device)
    end

    luci.http.prepare_content("application/json")
    luci.http.write(json.stringify(result))
end