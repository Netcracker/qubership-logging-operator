# Lua script tests

## Level normalization

Run from the repository root with Lua 5.1 or later, or LuaJIT:

```bash
lua scripts/lua-tests/test-update-level.lua
```

The test loads both production `update_level_syslog.lua` scripts and checks each against the same expectations.
It exits with an error on the first failed assertion. It covers severity mappings, case and whitespace handling,
unknown and missing levels, preservation of an existing `source_level` from the payload, and callback return values.
Normalization must use `level` even when the payload supplies a different `source_level` or `detected_level`.

These are function tests: they do not execute parsers, Fluent Bit tag matching, or HTTP routing.

## Key-value parsing

`test-kvs.lua` exercises `kv_parse` with different log patterns.
