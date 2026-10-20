# Tests for Lua scripts

The `test-kvs.lua` script loads a production `parse_key_value.lua` file and exercises `kv_parse` with valid, malformed,
and empty-message log records. Run it against both production implementations:

```bash
lua scripts/lua-tests/test-kvs.lua \
  controllers/fluentbit/fluentbit.configmap/conf.d/lua_scripts/parse_key_value.lua

lua scripts/lua-tests/test-kvs.lua \
  controllers/fluentbit-forwarder-aggregator/aggregator.configmap/conf.d/lua_scripts/parse_key_value.lua
```

The `test-update-level.lua` script exercises severity-level normalization.
