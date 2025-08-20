-- luacheck configuration for Fluent Bit Lua scripts
std = "lua51"

-- Allow all global variables
globals = {
    "update_level",
    "execute_real_func_test",
    "regex_msg",
    "kvs",
    "level",
    "time",
    "kv_parse_new_gen",
    "test_structure",
    "kv_parse",
    "code",
    "first_count_fields",
    "second_count_fields",
    "execute_real_func_test_new_gen",
    "execute_test",
    "execute_test_new_gen"
}

-- Ignore specific error codes
-- List of all warnings https://luacheck.readthedocs.io/en/stable/warnings.html
ignore = {
    "211", -- unused local variable
    "212", -- unused argument
    "213"  -- unused loop variable
}

-- Increase line length limit
max_line_length = 3100
