-- Run from the repository root with Lua 5.1+ or LuaJIT.
local script_paths = {
    "controllers/fluentbit/fluentbit.configmap/conf.d/lua_scripts/update_level_syslog.lua",
    "controllers/fluentbit-forwarder-aggregator/aggregator.configmap/conf.d/lua_scripts/update_level_syslog.lua",
}

local level_cases = {
    { inputs = { "0", "panic", "emerg", "emergency", "EMERG" }, normalized = "emerg", detected = "critical" },
    { inputs = { "1", "alert", "Fatal", "severe" }, normalized = "alert", detected = "critical" },
    { inputs = { "2", "crit", "critical", "CRITICAL" }, normalized = "crit", detected = "critical" },
    { inputs = { "3", "err", "error", "ERROR", "er", "E" }, normalized = "err", detected = "error" },
    { inputs = { "4", "warn", "WARN", "warning", "  WARN  " }, normalized = "warning", detected = "warn" },
    { inputs = { "5", "notice", "NOTICE" }, normalized = "notice", detected = "info" },
    { inputs = { "6", "info", "INFO" }, normalized = "info", detected = "info" },
    { inputs = { "7", "debug", "DEBUG", "verbose", "V" }, normalized = "debug", detected = "debug" },
    { inputs = { "trace", "TRACE", "  trace  " }, normalized = "debug", detected = "trace" },
    { inputs = { "", "   ", "???", "unknown", "8" }, normalized = "info", detected = "info", unknown = "true" },
}

local function assert_equal(actual, expected, context)
    assert(actual == expected,
        context .. ": expected " .. tostring(expected) .. ", got " .. tostring(actual))
end

local function check(update, record, expected, context)
    local timestamp = 1234567890.125
    local code, result_timestamp, result = update("custom.application", timestamp, record)
    assert_equal(code, 2, context .. " return code")
    assert_equal(result_timestamp, timestamp, context .. " timestamp")
    assert_equal(result, record, context .. " record identity")
    for _, field in ipairs({ "level", "detected_level", "source_level", "parse_level_unknown" }) do
        assert_equal(result[field], expected[field], context .. " " .. field)
    end
    assert_equal(result.message, "unchanged", context .. " message")
end

for _, path in ipairs(script_paths) do
    -- Reset the callback so a script that stops defining it cannot reuse the previous copy.
    update_level = nil
    dofile(path)
    assert(type(update_level) == "function", path .. ": missing update_level callback")
    local update = update_level
    local count = 0

    for _, case in ipairs(level_cases) do
        for _, input in ipairs(case.inputs) do
            check(update, { level = input, message = "unchanged" }, {
                level = case.normalized,
                detected_level = case.detected,
                source_level = input,
                parse_level_unknown = case.unknown,
            }, path .. " input=" .. string.format("%q", input))
            count = count + 1
        end
    end

    check(update, { message = "unchanged" }, {
        level = "info", detected_level = "info", parse_level_unknown = "true",
    }, path .. " missing level")
    count = count + 1

    -- Payload fields must not replace the level extracted by parsers as the normalization input.
    for _, source in ipairs({ "trace", "", false, 0 }) do
        check(update, {
            level = "error", source_level = source, detected_level = "trace", message = "unchanged",
        }, {
            level = "err", detected_level = "error", source_level = source,
        }, path .. " existing source_level=" .. tostring(source))
        count = count + 1
    end

    check(update, { source_level = "trace", message = "unchanged" }, {
        level = "info", detected_level = "info", source_level = "trace", parse_level_unknown = "true",
    }, path .. " existing source_level without level")
    count = count + 1

    print(path .. ": " .. count .. " cases passed")
end
