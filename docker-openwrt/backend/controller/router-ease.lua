module("luci.controller.router-ease", package.seeall)

function index()
    -- Create a top-level menu entry
    entry({"admin", "router-ease"}, firstchild(), "Router Ease", 60).dependent=false

    -- Add submenu entries
    entry({"admin", "router-ease", "speed-test"}, template("router-ease/speed-test"), "Speed Test", 4)
    -- API endpoints for AJAX calls
    entry({"admin", "router-ease", "run_speed_test"}, call("run_speed_test"))


end

function run_speed_test()
    local http = require("luci.http")
    local result = {}

    -- Using speedtest-netperf with proper error handling
    local cmd = io.popen("speedtest 2>&1")
    if not cmd then
        result = {
            download = "Failed to execute speedtest",
            upload = "Failed to execute speedtest",
            latency = "N/A",
            timestamp = os.date("%Y-%m-%d %H:%M:%S")
        }
    else
        local output = cmd:read("*a")
        cmd:close()

        -- Parse output with flexible pattern matching
        local download = output:match("[Dd]ownload:%s+([%d%.]+)%s+[Mm]bit/s") or
                         output:match("[Dd]ownload:%s+([%d%.]+)%s+[Mm][Bb]ps")
        local upload = output:match("[Uu]pload:%s+([%d%.]+)%s+[Mm]bit/s") or
                       output:match("[Uu]pload:%s+([%d%.]+)%s+[Mm][Bb]ps")
        local latency = output:match("[Ll]atency:%s+([%d%.]+)%s+ms") or
                        output:match("[Pp]ing:%s+([%d%.]+)%s+ms")

        -- Also try to extract server info if available
        local server = output:match("[Ss]erver:%s+(.-)[\r\n]")

        if download and upload then
            result = {
                download = download .. " Mbps",
                upload = upload .. " Mbps",
                latency = latency and (latency .. " ms") or "N/A",
                server = server or "Unknown",
                timestamp = os.date("%Y-%m-%d %H:%M:%S")
            }
        else
            -- If parsing failed, include partial output for debugging
            local error_msg = output:match("Error:[^\n]+") or "Unknown error"
            result = {
                download = "Test failed",
                upload = "Test failed",
                latency = "N/A",
                error = error_msg:sub(1, 100),
                timestamp = os.date("%Y-%m-%d %H:%M:%S")
            }
        end
    end

    http.prepare_content("application/json")
    http.write_json(result)
end