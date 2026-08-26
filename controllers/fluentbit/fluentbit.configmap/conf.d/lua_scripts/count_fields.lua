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
    return 2, timestamp, record
end
-- Count fields after parsing
function second_count_fields(tag, timestamp, record)
    if record["log"] == nil or record["parse_status"] == "success" then
        return 0, timestamp, record
    end
    local count = 0
    for k in pairs(record) do
        if   k ~= "logfmt_candidate"
         and k ~= "parse_field_count"
         and k ~= "parse_status"
         and k ~= "orig_field_count" then
            count = count + 1
        end
    end
    local orig_count = record["orig_field_count"]
    if orig_count ~= nil and count > orig_count then
        record["parse_status"] = "success"
    else
        record["parse_status"] = "failed"
    end
    record["parse_field_count"] = count
    return 2, timestamp, record
end
