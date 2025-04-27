    module("luci.controller.router-ease", package.seeall)

    function index()
        -- Create a top-level menu entry
        entry({"admin", "router-ease"}, firstchild(), "My OpenWRT", 60).dependent=false

        -- Add submenu entries
        entry({"admin", "router-ease", "network"}, template("router-ease/network"), "Network Settings", 1)
        entry({"admin", "router-ease", "system"}, template("router-ease/system"), "System Settings", 2)

        -- API endpoints for AJAX calls
        entry({"admin", "router-ease", "get_network"}, call("get_network_info"))
        entry({"admin", "router-ease", "update_network"}, call("update_network"))
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