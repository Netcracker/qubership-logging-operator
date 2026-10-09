-- Run from the repository root with Lua 5.1+ or LuaJIT.
local script_paths = {
    "controllers/fluentbit/fluentbit.configmap/conf.d/lua_scripts/update_level_syslog.lua",
    "controllers/fluentbit-forwarder-aggregator/aggregator.configmap/conf.d/lua_scripts/update_level_syslog.lua",
}

local configurations = {
    {
        name = "Fluent Bit",
        filter = "controllers/fluentbit/fluentbit.configmap/conf.d/filters/filter-nonsupported-levels.conf",
        output = "controllers/fluentbit/fluentbit.configmap/conf.d/outputs/output-http.conf",
    },
    {
        name = "Fluent Bit aggregator",
        filter = "controllers/fluentbit-forwarder-aggregator/aggregator.configmap/conf.d/filters/" ..
            "filter-nonsupported-levels.conf",
        output = "controllers/fluentbit-forwarder-aggregator/aggregator.configmap/conf.d/outputs/output-http.conf",
    },
}

local expected_match_regex = "^(?!out_(audit|k8s_event|nginx|access|int|pods|system|default)$).*"
local expected_routing_tags = {
    out_access = true,
    out_audit = true,
    out_default = true,
    out_int = true,
    out_k8s_event = true,
    out_nginx = true,
    out_pods = true,
    out_system = true,
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

local function read_file(path)
    local file = assert(io.open(path, "r"))
    local content = file:read("*a")
    file:close()
    return content
end

local function directive_value(configuration, directive)
    for line in configuration:gmatch("[^\r\n]+") do
        local key, value = line:match("^%s*(%S+)%s+([^%s]+)%s*$")
        if key == directive then
            return value
        end
    end
    error("missing " .. directive .. " directive")
end

-- The filter is a Go template: the exclusion renders only when HTTP routing is enabled, otherwise it matches every tag.
local function assert_routing_conditional(filter, context)
    local condition = filter:match("{{%-%s*if%s+and%s+([^}]-)%s*}}")
    assert(condition, context .. ": missing the HTTP routing condition")
    assert(condition:find("%.Output%.Http%.Routing%.Enabled$"), context .. ": condition does not end with Routing.Enabled")
    assert(filter:find("{{%-%s*else%s*}}"), context .. ": missing the else branch")
    local before_else, after_else = filter:match("^(.-){{%-%s*else%s*}}(.*)$")
    assert(before_else:find("Match_regex"), context .. ": Match_regex must be in the routing-enabled branch")
    assert(not after_else:find("Match_regex"), context .. ": Match_regex must not render when routing is disabled")
    assert_equal(directive_value(after_else, "Match"), "*", context .. " routing-disabled Match")
end

local function routing_tags(configuration)
    local tags = {}
    for line in configuration:gmatch("[^\r\n]+") do
        local tag = line:match("^%s*Rule%s+.*%s+(out_[%w_]+)%s+false%s*$")
        if tag ~= nil then
            tags[tag] = true
        end
    end
    return tags
end

local function assert_same_keys(actual, expected, context)
    for key in pairs(expected) do
        assert(actual[key], context .. ": missing " .. key)
    end
    for key in pairs(actual) do
        assert(expected[key], context .. ": unexpected " .. key)
    end
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

for _, configuration in ipairs(configurations) do
    local filter = read_file(configuration.filter)
    assert_equal(directive_value(filter, "Match_regex"), expected_match_regex,
        configuration.name .. " level filter Match_regex")
    assert_routing_conditional(filter, configuration.name .. " level filter")

    local output = read_file(configuration.output)
    assert_same_keys(routing_tags(output), expected_routing_tags, configuration.name .. " HTTP routing tags")
    print(configuration.name .. ": level filter routing tags passed")
end
