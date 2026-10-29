-- input: https://docs.fluentbit.io/manual/pipeline/filters/lua#function-arguments
-- output: https://docs.fluentbit.io/manual/pipeline/filters/lua#return-values
function kv_parse(tag, timestamp, record)
    -- Markers from non-Qubership payloads are removed by modify filters before this function runs.
    if record["__qubership_candidate"] == nil then
        return 0, timestamp, record
    end
    if record["log"] ~= nil and type(record["log"]) ~= "table" then
        -- The record matched the qubership parser:
        -- [<time>] [<level>] [key1=value1] ... [keyN=valueN] <message>
        -- Read the [key=value] groups that directly follow the header, the same groups the parser skips before
        -- short_message. The message can contain brackets of its own, so it never decides where the groups end.
        local regex_header = "^%s*%[[^%[%]]*%]%s*%[[^%[%]]*%]"
        local regex_kv = "^%s*%[%s*([^%[%]%s=]+)%s*=([^%[%]]*)%]()"
        local s = record["log"]

        local _, header_end = string.find(s, regex_header)
        if header_end == nil then
            return 0, timestamp, record
        end

        local found = false
        local key, value, next_position = string.match(s, regex_kv, header_end + 1)
        while key ~= nil do
            found = true
            local trimmed_value = value:gsub("^%s*(.-)%s*$", "%1")
            -- Quoted keys and values are not key-value fields, and empty values are skipped.
            if trimmed_value ~= "" and not string.find(key, "\"", 1, true) and
                not string.find(trimmed_value, "\"", 1, true) then
                record[key] = trimmed_value
            end
            key, value, next_position = string.match(s, regex_kv, next_position)
        end

        if not found then
            return 0, timestamp, record
        end

        -- return 2, that means the original timestamp is not modified and the record has been modified
        -- so it must be replaced by the returned values from the record
        return 2, timestamp, record
    else
        -- return 0, that means the record will not be modified
        return 0, timestamp, record
    end
end
