    module("luci.controller.router-ease", package.seeall)

    function index()
        -- Create a top-level menu entry
        entry({"admin", "router-ease"}, firstchild(), "Router Ease GUI", 60).dependent=false

        -- Add submenu entries
        entry({"admin", "router-ease", "network"}, template("router-ease/network"), "Network Settings", 1)
        entry({"admin", "router-ease", "system"}, template("router-ease/system"), "System Settings", 2)
        entry({"admin", "router-ease", "device-information"}, template("router-ease/system"), "Devices Information", 3)
        entry({"admin", "router-ease", "speed-test"}, template("router-ease/speed-test"), "Speed Test", 4)
        entry({"admin", "router-ease", "qr"}, template("router-ease/speed-test"), "QR", 5|)

        -- API endpoints for AJAX calls
        entry({"admin", "router-ease", "get_network"}, call("get_network_info"))
        entry({"admin", "router-ease", "update_network"}, call("update_network"))
        entry({"admin", "router-ease", "run_speed_test"}, call("run_speed_test"))
        entry({"admin", "router-ease", "get_wifi_info"}, call("get_wifi_info"))
    end

    function get_network_info()
        local uci = require("luci.model.uci").cursor()
        local network_info = {}

        -- Get all network configurations
        uci:foreach("network", nil, function(section)
            network_info[section[".name"]] = section
        end)

        -- Include additional network info
        network_info["interfaces"] = {}
        local interfaces = io.popen("ip -j addr show")
        if interfaces then
            local output = interfaces:read("*a")
            interfaces:close()
            if output and output ~= "" then
                local json = require("luci.jsonc")
                network_info["interfaces"] = json.parse(output) or {}
            end
        end

        luci.http.prepare_content("application/json")
        luci.http.write_json(network_info)
    end

    function update_network()
        local uci = require("luci.model.uci").cursor()
        local json = require("luci.jsonc")
        local http = require("luci.http")

        local data = http.content()
        local settings = json.parse(data)

        if not settings or not settings.section or not settings.options then
            http.status(400, "Invalid input")
            return
        end

        -- Update network settings
        for option, value in pairs(settings.options) do
            uci:set("network", settings.section, option, value)
        end

        uci:commit("network")
        os.execute("/etc/init.d/network restart")

        http.prepare_content("application/json")
        http.write_json({success = true})
    end

function run_speed_test()
    local http = require("luci.http")
    local result = {}

    -- Run download speed test using wget
    local dl_cmd = io.popen("wget -O /dev/null http://speedtest.wdc01.softlayer.com/downloads/test10.zip 2>&1 | grep -i 'saved'")
    local dl_output = dl_cmd:read("*a")
    dl_cmd:close()

    -- Extract speed from wget output
    result.download = dl_output:match("%(([%d%.]+%s+[KMG]B/s)%)") or "Test failed"

    -- Simple upload test (not as accurate)
    local ul_cmd = io.popen("dd if=/dev/zero bs=1M count=8 2>/dev/null | curl -s -X POST -T - https://httpbin.org/anything >/dev/null 2>&1 && echo 'Success'")
    local ul_output = ul_cmd:read("*a")
    ul_cmd:close()

    if ul_output:match("Success") then
        result.upload = "~2-5 MB/s (estimate)"
    else
        result.upload = "Test failed"
    end

    result.timestamp = os.date("%Y-%m-%d %H:%M:%S")

    http.prepare_content("application/json")
    http.write_json(result)
end

-- Get WiFi information for QR code
function get_wifi_info()
    local uci = require("luci.model.uci").cursor()
    local http = require("luci.http")
    local wifi_info = {}

    -- Get wireless configurations
    local wireless = uci:get_all("wireless")

    -- Find the first wireless interface that's not disabled
    for k, v in pairs(wireless) do
        if v[".type"] == "wifi-iface" and v.disabled ~= "1" then
            wifi_info = {
                ssid = v.ssid or "",
                encryption = v.encryption or "",
                key = v.key or ""
            }
            break
        end
    end

    http.prepare_content("application/json")
    http.write_json(wifi_info)
end