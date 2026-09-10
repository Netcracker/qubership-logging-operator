-- Count only application and metadata fields, excluding pipeline state.
local function count_fields(record)
    local count = 0
    for key in pairs(record) do
        if key ~= "json_candidate" and key ~= "logfmt_candidate"
            and key ~= "orig_field_count" and key ~= "parse_field_count"
            and key ~= "parse_status" and key ~= "parse_format" then
            count = count + 1
        end
    end
    return count
end

function first_count_fields(tag, timestamp, record)
    if record["log"] == nil then
        return 0, timestamp, record
    end
    record["orig_field_count"] = count_fields(record)
    return 2, timestamp, record
end

function second_count_fields(tag, timestamp, record)
    local original = record["orig_field_count"]
    if original == nil or record["log"] == nil then
        return 0, timestamp, record
    end
    local count = count_fields(record)
    record["parse_status"] = count > original and "success" or "failed"
    record["parse_field_count"] = count
    return 2, timestamp, record
end
