-- Count fields in a record before parsing
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
-- Count fields in a record after parsing
function second_count_fields(tag, timestamp, record)
    if record["log"] ~= nil then
        local count = 0
        for _ in pairs(record) do
            count = count + 1
        end
        if record["orig_field_count"] ~= nil then
            count = count - 1 -- Subtracting orig_field_count
        end
        if record["field_count"] ~= nil then
            count = count - 2 -- Subtracting field_count and parsed
        end
        if (count > record["orig_field_count"]) then
            record["parsed"] = "true"
        else
            record["parsed"] = "false"
        end
        record["field_count"] = count
        return 2, timestamp, record
    end
end
