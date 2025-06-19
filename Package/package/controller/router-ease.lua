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

-- Main bandwidth monitoring page|
     entry({"admin", "router-ease", "bandwidth"}, view("nlbw/display"), _("Bandwidth Monitor"), 35)
     entry({"admin", "router-ease", "settings"}, template("router-ease/settings"), "Settings", 10)
     -- Sub-pages if desired (optional)
     entry({"admin", "router-ease", "bandwidth", "config"}, view("nlbw/config"), _("Configuration"))
     entry({"admin", "router-ease", "bandwidth", "backup"}, view("nlbw/backup"), _("Backup"))
     entry({"admin", "network", "speedtest", "action_run_speedtest"}, call("action_run_speedtest"), nil).leaf = true
     entry({"admin", "network", "speedtest", "action_status"}, call("action_status"), nil).leaf = true
     entry({"admin", "router-ease", "get_wifi_info"}, call("get_wifi_info"), nil).leaf = true
     entry({"admin", "router-ease", "get_connected_devices"}, call("get_connected_devices"), nil).leaf = true
     entry({"admin", "router-ease", "kick_device"}, call("action_kick_device"), nil).leaf = true
     entry({"admin", "router-ease", "qos_status"}, call("qos_status"))

     -- Add these inside your index() function
     entry({"admin", "network", "speedtest"}, template("router-ease/speed-test"), _("Speed Test"), 90)
     entry({"admin", "network", "speedtest", "run"}, call("action_run_speedtest"))
     entry({"admin", "network", "speedtest", "status"}, call("action_speedtest_status"))
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

function parse_speedtest_results(json_str)
    local json = require "luci.jsonc"
    local result = {
        download = 0,
        upload = 0,
        ping = 0,
        error_msg = nil
    }

    -- Log raw results for debugging
    local debug_log = io.open("/tmp/speedtest_parse_debug.log", "w")
    if debug_log then
        debug_log:write("Raw results:\n" .. json_str .. "\n\n")
    end

    -- Try to parse the JSON
    local success, data = pcall(function() return json.parse(json_str) end)

    if success and data then
        -- Extract values from the JSON structure
        result.download = tonumber(data.download) or 0
        result.upload = tonumber(data.upload) or 0
        result.ping = tonumber(data.ping) or 0

        if debug_log then
            debug_log:write("Successfully parsed JSON data\n")
            debug_log:write(string.format("Download: %f bps\n", result.download))
            debug_log:write(string.format("Upload: %f bps\n", result.upload))
            debug_log:write(string.format("Ping: %f ms\n", result.ping))
        end
    else
        result.error_msg = "Failed to parse speedtest results"
        if debug_log then debug_log:write("Failed to parse JSON: " .. (data or "unknown error") .. "\n") end
    end

    if debug_log then debug_log:close() end

    return result.download, result.upload, result.ping, result.error_msg
end

function action_status()
    local fs = require "nixio.fs"
    local sys = require "luci.sys"

    luci.http.prepare_content("application/json")

    -- Check if speedtest is still running
    local running = (sys.exec("pgrep -f speedtest-cli") ~= "")

    if running then
        luci.http.write_json({ status = "running" })
        return
    end

    -- Check for results file
    if fs.access("/tmp/speedtest_result") then
        local results = fs.readfile("/tmp/speedtest_result")

        -- Parse the results
        local download, upload, ping, error_msg = parse_speedtest_results(results)

        local response = {
            status = "complete",
            data = {
                download = { bps_mean = download },
                upload = { bps_mean = upload },
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

    -- Function to get manufacturer from external service
    local function get_manufacturer(mac)
        if not mac then return nil end
        local url = "http://192.168.0.113/routerease/mac-address/?mac=" .. mac
        local resp = util.exec("curl -s '" .. url .. "'")
        if resp and #resp > 0 then
            local data = json.parse(resp)
            if data and data.manufacturer then
                return data.manufacturer
            end
        end
        return nil
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
        if line:match("^%d+%.%d+%.%d+%.%d+") then
            local ip, mac, vendor = line:match("(%d+%.%d+%.%d+%.%d+)%s+([0-9a-fA-F:]+)%s+(.*)")
            if ip and mac and mac:match("^[0-9a-fA-F:]+$") and mac ~= "00:00:00:00:00:00" then
                local upper_mac = string.upper(mac)
                local hostname = hostnames[upper_mac] or "Unknown"
                local manufacturer = get_manufacturer(upper_mac)

                local dev_info = {
                    ip = ip,
                    mac = upper_mac,
                    interface = "unknown",
                    hostname = hostname,
                    manufacturer = manufacturer or "Unknown",
                    connection_type = "unknown",
                    signal = nil,
                    rx_bytes = 0,
                    tx_bytes = 0,
                    last_seen = os.time()
                }

                if hostname == "Unknown" then
                    dev_info.display_name = manufacturer or "Unknown"
                else
                    dev_info.display_name = hostname
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
            local hostname = hostnames[upper_mac] or "Unknown"
            local manufacturer = get_manufacturer(upper_mac)
            if devices[upper_mac] then
                devices[upper_mac].interface = device
                devices[upper_mac].connection_type = "wired"
                if hostname == "Unknown" then
                    devices[upper_mac].display_name = manufacturer or "Unknown"
                end
            else
                local dev_info = {
                    ip = ip,
                    mac = upper_mac,
                    interface = device,
                    hostname = hostname,
                    manufacturer = manufacturer or "Unknown",
                    connection_type = "wired",
                    signal = nil,
                    rx_bytes = 0,
                    tx_bytes = 0,
                    last_seen = os.time()
                }
                if hostname == "Unknown" then
                    dev_info.display_name = manufacturer or "Unknown"
                else
                    dev_info.display_name = hostname
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
                    devices[upper_mac].connection_type = "wifi"
                    devices[upper_mac].essid = current_essid
                    devices[upper_mac].interface = current_iface
                    local signal = line:match("Signal: ([%-0-9]+)") or "0"
                    devices[upper_mac].signal = tonumber(signal) or 0
                end
            end
        end
    end

    for _, device in pairs(devices) do
        table.insert(result, device)
    end

    luci.http.prepare_content("application/json")
    luci.http.write(json.stringify(result))
end

-- Add these functions to your router-ease.lua controller

function action_run_speedtest()
    local json = require "luci.jsonc"
    local fs = require "nixio.fs"
    local util = require "luci.util"

    -- Create a status file to track ongoing test
    fs.writefile("/tmp/speedtest_status", "running")

    -- Launch the speedtest in background to avoid timeout
    local script = [[
#!/bin/sh
python3 -c '
import json
import subprocess
import time
import os

try:
    # Run speedtest-cli with JSON output
    result = subprocess.check_output(["speedtest-cli", "--json"], universal_newlines=True)
    data = json.loads(result)

    # Format the results to match what frontend expects
    output = {
        "status": "complete",
        "data": {
            "ping": {"median": data["ping"]},
            "download": {"bps_mean": data["download"]},
            "upload": {"bps_mean": data["upload"]}
        }
    }

    # Save the results
    with open("/tmp/speedtest_result", "w") as f:
        f.write(json.dumps(output))

except Exception as e:
    # Handle errors
    error = {
        "status": "error",
        "message": str(e)
    }
    with open("/tmp/speedtest_result", "w") as f:
        f.write(json.dumps(error))

# Mark test as complete
with open("/tmp/speedtest_status", "w") as f:
    f.write("complete")
'
    ]]

    -- Execute the script in background
    local cmd = string.format("(%s) >/dev/null 2>&1 &", script)
    util.exec(cmd)

    -- Return immediate response to trigger polling
    luci.http.prepare_content("application/json")
    luci.http.write_json({status = "started"})
end

function action_speedtest_status()
    local json = require "luci.jsonc"
    local fs = require "nixio.fs"

    -- Check if test is still running
    local status = fs.readfile("/tmp/speedtest_status") or "error"
    status = status:gsub("\n", "")

    if status == "complete" then
        -- Return the completed test results
        local result = fs.readfile("/tmp/speedtest_result") or "{}"
        local data = json.parse(result)

        luci.http.prepare_content("application/json")
        luci.http.write_json(data)
    elseif status == "running" then
        -- Test still in progress
        luci.http.prepare_content("application/json")
        luci.http.write_json({status = "running"})
    else
        -- Something went wrong
        luci.http.prepare_content("application/json")
        luci.http.write_json({
            status = "error",
            message = "Failed to start speed test"
        })
    end
end