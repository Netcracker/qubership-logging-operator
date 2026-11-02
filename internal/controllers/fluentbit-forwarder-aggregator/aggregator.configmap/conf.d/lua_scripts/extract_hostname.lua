-- input: https://docs.fluentbit.io/manual/pipeline/filters/lua#function-arguments
-- output: https://docs.fluentbit.io/manual/pipeline/filters/lua#return-values
function extract_hostname(tag, timestamp, record)
    if record["hostname"] == nil then
        local hostname = string.match(tag, "^pods%.([^%.]+)")
        if hostname ~= nil then
            record["hostname"] = hostname
            -- return 2, that means the original timestamp is not modified and the record has been modified
            -- so it must be replaced by the returned values from the record
            return 2, timestamp, record
        end
    end
    -- return 0, that means the record will not be modified
    return 0, timestamp, record
end
