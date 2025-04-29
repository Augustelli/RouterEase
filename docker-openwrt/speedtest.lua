module("luci.controller.speedtest", package.seeall)

function index()
    entry({"admin", "network", "speedtest"}, template("router-ease/speedtest"), _("Speed Test"), 90)
    entry({"admin", "network", "speedtest", "run"}, call("action_run_speedtest"), nil)
    entry({"admin", "network", "speedtest", "status"}, call("action_status"), nil)
end

function action_run_speedtest()
    local sys = require "luci.sys"
    local fs = require "nixio.fs"

    -- Clear old results if they exist
    if fs.access("/tmp/speedtest_results.txt") then
        fs.unlink("/tmp/speedtest_results.txt")
    end

    -- Run speedtest-netperf.sh and redirect output to a file
    sys.call("which speedtest-netperf.sh > /tmp/speedtest_path.txt")
    sys.call("speedtest-netperf.sh > /tmp/speedtest_results.txt 2>&1 &")

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