local PARSED_FIELD_PREFIX = "parse_"
local PARSED_FIELDS_KEY = "parsed_fields"

local function prefix_depth(key)
    local depth = 0
    local remaining_key = key

    while string.sub(remaining_key, 1, #PARSED_FIELD_PREFIX) == PARSED_FIELD_PREFIX do
        depth = depth + 1
        remaining_key = string.sub(remaining_key, #PARSED_FIELD_PREFIX + 1)
    end

    return depth
end

-- Keep fields already present at the record root authoritative. Conflicting fields
-- extracted from the log are moved under parsed_fields and lifted later with a
-- parse_ prefix. Sorting makes prefix chains deterministic: namespace is assigned
-- parse_namespace before an original parse_namespace is assigned
-- parse_parse_namespace.
local function move_parsed_field_collisions(record)
    local parsed_log = record["log_parsed"]
    if type(parsed_log) ~= "table" then
        return
    end

    local occupied_keys = {}
    for key in pairs(record) do
        if key ~= "log_parsed" then
            occupied_keys[key] = true
        end
    end
    occupied_keys[PARSED_FIELDS_KEY] = true

    local parsed_keys = {}
    for key in pairs(parsed_log) do
        if type(key) == "string" then
            table.insert(parsed_keys, key)
        end
    end
    table.sort(parsed_keys, function(left, right)
        local left_depth = prefix_depth(left)
        local right_depth = prefix_depth(right)
        if left_depth == right_depth then
            return left < right
        end
        return left_depth < right_depth
    end)

    local conflicting_fields = {}
    for _, key in ipairs(parsed_keys) do
        local output_key = key
        while occupied_keys[output_key] do
            output_key = PARSED_FIELD_PREFIX .. output_key
        end
        occupied_keys[output_key] = true

        if output_key ~= key then
            -- The nest filter adds the first parse_ prefix when lifting this map.
            local nested_key = string.sub(output_key, #PARSED_FIELD_PREFIX + 1)
            conflicting_fields[nested_key] = parsed_log[key]
            parsed_log[key] = nil
        end
    end

    if next(conflicting_fields) ~= nil then
        record[PARSED_FIELDS_KEY] = conflicting_fields
    end
end

-- Count fields before parsing
function first_count_fields(tag, timestamp, record)
    local count = 0
    for _ in pairs(record) do
        count = count + 1
    end
    if record["log_parsed"] ~= nil then
        count = count - 1 -- Subtracting log_parsed
    end
    record["orig_field_count"] = count
    move_parsed_field_collisions(record)
    return 2, timestamp, record
end
-- Count fields after parsing
function second_count_fields(tag, timestamp, record)
    if record["log"] ~= nil then
        local count = 0
        for _ in pairs(record) do
            count = count + 1
        end
        if record["logfmt_candidate"] ~= nil then
            count = count - 1 -- Subtracting logfmt_candidate
        end
        if record["parse_field_count"] ~= nil then
            count = count - 1 -- Subtracting parse_field_count
        end
        if record["parse_status"] == "success" then
            return 0, timestamp, record
        elseif record["parse_status"] ~= nil then
            count = count - 1
        end

        if record["orig_field_count"] ~= nil then
            count = count - 1 -- Subtracting orig_field_count
            if (count > record["orig_field_count"]) then
                record["parse_status"] = "success"
            else
                record["parse_status"] = "failed"
            end
        else
            record["parse_status"] = "failed"
        end
        record["parse_field_count"] = count
        return 2, timestamp, record
    end
end
