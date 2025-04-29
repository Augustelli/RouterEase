module("luci.controller.qr", package.seeall)


function index()
    -- Create a top-level menu entry
    entry({"admin", "router-ease"}, firstchild(), "QR Connection", 60).dependent=false

    -- Add submenu entries
    entry({"admin", "router-ease", "speed-test"}, template("router-ease/speed-test"), "QR connection code", 60)

    -- API endpoints for AJAX calls
    entry({"admin", "router-ease", "get_wifi_info"}, call("get_wifi_info"), nil).leaf = true

end

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