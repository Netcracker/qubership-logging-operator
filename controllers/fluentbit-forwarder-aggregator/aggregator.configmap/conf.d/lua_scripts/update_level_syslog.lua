-- Fluent Bit supports only next levels:
-- "emerg", "alert", "crit", "err", "warning", "notice", "info", "debug"
-- Fluent Bit source code of gelf output:
-- https://github.com/fluent/fluent-bit/blob/master/src/flb_pack_gelf.c#L563-L592
-- this script marks non supported levels with syslog codes
-- input: https://docs.fluentbit.io/manual/pipeline/filters/lua#function-arguments
-- output: https://docs.fluentbit.io/manual/pipeline/filters/lua#return-values
local function normalize_levels(level)
  local normalized = "info"
  local detected = "info"

  if level == nil then
    return normalized, detected, true
  end

  level = string.lower(level):gsub("^%s*(.-)%s*$", "%1")
  local first_ch = string.sub(level, 1, 1)

  -- p = panic
  if first_ch == '0' or first_ch == 'p' then
    normalized = "emerg"
    detected = "critical"
  -- a = alert, f = fatal, s = severe
  elseif first_ch == '1' or first_ch == 'a' or first_ch == 'f' or first_ch == 's' then
    normalized = "alert"
    detected = "critical"
  -- c = crit
  elseif first_ch == '2' or first_ch == 'c' then
    normalized = "crit"
    detected = "critical"
  elseif first_ch == '3' then
    normalized = "err"
    detected = "error"
  -- w = warning
  elseif first_ch == '4' or first_ch == 'w' then
    normalized = "warning"
    detected = "warn"
  -- n = notice
  elseif first_ch == '5' or first_ch == 'n' then
    normalized = "notice"
    detected = "info"
  -- i = info
  elseif first_ch == '6' or first_ch == 'i' then
    normalized = "info"
    detected = "info"
  -- d = debug, v = verbose
  elseif first_ch == '7' or first_ch == 'd' or first_ch == 'v' then
    normalized = "debug"
    detected = "debug"
  elseif first_ch == 't' then
    normalized = "debug"
    detected = "trace"
  -- e, er = err, e(~=r) = emerg
  elseif first_ch == 'e' then
    if string.len(level) >=2 and string.sub(level, 2, 2) ~= 'r' then
      normalized = "emerg"
      detected = "critical"
    else
      normalized = "err"
      detected = "error"
    end
  else
    return "info", "info", true
  end

  return normalized, detected, false
end

function update_level(tag, timestamp, record)
  record["source_level"] = record["level"]
  local level_unknown
  record["level"], record["detected_level"], level_unknown = normalize_levels(record["level"])
  if level_unknown then
    record["parse_level_unknown"] = "true"
  end

  -- return 2, that means the original timestamp is not modified and the record has been modified
  return 2, timestamp, record
end
