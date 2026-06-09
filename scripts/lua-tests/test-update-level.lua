-- different test strings
local test_strings = {
    -- correct: valid syslog levels, from 0 to 7
    "emerg",
    "alert",
    "crit",
    "err",
    "warning",
    "notice",
    "info",
    "debug",
    -- correct: full level names
    "emergency",
    "alert",
    "critical",
    "error",
    "warning",
    "notice",
    "info",
    "debug",
    -- correct: levels using capital letters
    "EMERG",
    "ALERT",
    "CRTI",
    "ERR",
    "WARNING",
    "NOTICE",
    "INFO",
    "DEBUG",
    -- correct: full level names using upper case
    "EMERGENCY",
    "ALERT",
    "CRITICAL",
    "ERROR",
    "WARNING",
    "NOTICE",
    "INFO",
    "DEBUG",
    -- correct: other short or full level names forms
    "warn",
    "fatal",
    "trace",
    -- incorrect: short level names
    "emg",
    "alrt",
    "art",
    "alt",
    "crt",
    "wrg",
    "wrn",
    "inf",
    "dbg",
    -- incorrect: various combinations of levels
    "er",
    "E",
    "wa",
    "war",
    "W",
    "ntc",
    "noti",
    "N",
    "in",
    "I",
    "deb",
    "D",
    "fat",
    "F",
    -- incorrect: words which were parsed as levels
    "number",
}

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

-- test functions
-- call "like real" functions
function execute_real_func_test()
    for i, test_string in ipairs(test_strings) do
        local test_structure = {}
        test_structure["level"] = test_string

        local start_time = os.time()
        local code, time, new_test_structure = update_level("test", i, test_structure)
        local end_time = os.time()

        for k,v in pairs(new_test_structure) do
            if (k == "level" and code == 2) then
                print("Original string:", test_string)
                print("Call kv_parse = ", start_time)
                print("Complete update_level = ", end_time, "Execution time =", end_time - start_time)
                print("Code:", code, "Processing order:", time)
                print("Update level:", test_string, "=>", v)
                print("Detected level:", test_string, "=>", new_test_structure["detected_level"])
                print("------------------------------------------------------------------------")
            end
            if (k == "level" and code == 0) then
                -- although this level can be ignore by script, but this level will validate by regex
                print ("Level was ignored by script:", test_string)
            end
        end
    end
end

print("====================================================================")
print("Run test to check function which will use Fluent")
print("====================================================================")
execute_real_func_test()
