#!/usr/bin/env bash

# The generated API reference uses HTML tables that are not compatible with markdownlint rules.
sed -i "1i\<!-- markdownlint-disable -->" docs/api.md
