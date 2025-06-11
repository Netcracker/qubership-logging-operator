-- Fluent Bit supports only next levels:
-- "emerg", "alert", "crit", "err", "warning", "notice", "info", "debug"
-- Fluent Bit source code of gelf output:
-- https://github.com/fluent/fluent-bit/blob/master/src/flb_pack_gelf.c#L563-L592
-- this script marks non supported levels with syslog codes
-- input: https://docs.fluentbit.io/manual/pipeline/filters/lua#function-arguments
-- output: https://docs.fluentbit.io/manual/pipeline/filters/lua#return-values
function update_level(tag, timestamp, record)
  record["source_level"] = record["level"]
  if (record["level"] ~= nil) then
    record["level"] = string.lower(record["level"]):gsub("^%s*(.-)%s*$", "%1")
    local first_ch = string.sub(record["level"], 1, 1)
    if first_ch == '0' then
      record["level"] = "emerg"
    -- a = alert, f = fatal, s = severe
    elseif first_ch == '1' or first_ch == 'a' or first_ch == 'f' or first_ch == 's' then
      record["level"] = "alert"
    -- c = crit
    elseif first_ch == '2' or first_ch == 'c' then
      record["level"] = "crit"
    elseif first_ch == '3' then
      record["level"] = "err"
    -- w = warning
    elseif first_ch == '4' or first_ch == 'w' then
      record["level"] = "warning"
    -- n = notice
    elseif first_ch == '5' or first_ch == 'n' then
      record["level"] = "notice"
    -- i = info
    elseif first_ch == '6' or first_ch == 'i' then
      record["level"] = "info"
    -- d = debug, t = trace
    elseif first_ch == '7' or first_ch == 'd' or first_ch == 't' then
      record["level"] = "debug"
    -- e, er = err, e(~=r) = emerg
    elseif first_ch == 'e' then
      if string.len(record["level"]) >=2 and string.sub(record["level"], 2, 2) ~= 'r' then
        record["level"] = "emerg"
      else
        record["level"] = "err"
      end
    else
      record["level_unknown"] = "true"
      record["level"] = "info"
    end
  else
    record["level_unknown"] = "true"
    record["level"] = "info"
  end

  -- return 2, that means the original timestamp is not modified and the record has been modified
  return 2, timestamp, record
end
