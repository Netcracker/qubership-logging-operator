# rca-cases — cause and draft reply per known problem

For each case id that `scripts/match_rca.py` returns, read the matching section
below. A match is a hint: use the case only when its caveat holds, and emit the
draft reply as a suggestion the operator confirms. Placeholders like
`{{opensearch_url}}` are filled by the operator when the value is not in the ticket.

## ism-config-cosmetic

**Cause:** cosmetic bug in the OpenSearch index-management plugin before
2.10.0.0 — logged at ERROR but should be DEBUG. No functional impact, no
data loss.
**Reply:** This is a known cosmetic log line, not a fault. Ignore it, or
create any ISM policy (even an empty one) to make the index appear and
silence it.
**Caveat:** none.

## timestamp-tz-mismatch

**Cause:** the Graylog UI renders `timestamp` in the viewer's timezone while
the `message` text keeps the source application's timezone. Not a logging
defect.
**Reply:** Align them either by setting the node timezone to UTC, or by
changing the Graylog user's display timezone (this only affects how
`timestamp` is shown, never the `message` text).
**Caveat:** if the author insists the `message` text itself is rewritten,
this case does not apply — hand off to L2.

## deflector-exists-as-index

**Cause:** the `_deflector` suffix is reserved by Graylog; an index with that
suffix was created (usually because inputs were not stopped during an
upgrade), so the alias cannot attach.
**Reply:** Delete the offending index. For future upgrades, stop the Graylog
inputs first so this does not recur.
**Caveat:** if the operator cannot confirm which index is offending, or
deleting it risks log loss, this case does not apply — hand off to L2.

## fields-limit-1000-exceeded

**Cause:** older Logging versions let the FluentBit/FluentD parser extract
noisy `key=value` pairs that explode the index mapping until the 1000-field
limit is hit.
**Reply:** Upgrade Logging to a version with the fixed parser. The
accumulated noise fields can be cleaned with the painless script in the
runbook.
**Caveat:** none.

## dr-no-vip-cyclic-redirect

**Cause:** DR no-vIP topology with a Load Balancer plus HTTPS termination —
the Graylog UI is reachable only when SNI passthrough is configured for the
Graylog route.
**Reply:** Add the Graylog Route URL to `os_sni_passthrough.map` on the Load
Balancers and retry.
**Caveat:** only the DR no-vIP topology. If the install uses a vIP or a
different LB product, this case does not apply — hand off to L2.

## index-read-only-after-disk-cleanup

**Cause:** after a disk fill, OpenSearch sets the index read-only flag and
does not clear it automatically once space is freed.
**Reply:** Clear it against the OpenSearch endpoint, then keep all index sets
under ~85% of the data disk so it does not recur:

```sh
curl -X PUT -H 'Content-Type: application/json' \
  -d '{"index.blocks.read_only_allow_delete": null}' \
  '{{opensearch_url}}/_all/_settings'
```

**Caveat:** only trivial once the author confirms the disk is below the
watermark. If the disk is still full, hand off to L2 for the
capacity/rotation issue.

## nodes-info-unavailable-tls

**Cause:** usually an expired TLS certificate, or one missing the required
Subject Alternative Names (e.g. `graylog-service.<namespace>.svc`).
**Reply:** Re-issue the certificate with the correct alt names and restart
the Graylog service so the new cert loads.
**Caveat:** match only when the ticket mentions TLS, an expired cert, or
shows a cert error. The same message can come from an OpenSearch outage —
that is not this case; hand off to L2.
</content>
