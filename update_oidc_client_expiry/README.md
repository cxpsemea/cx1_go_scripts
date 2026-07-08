# update_oidc_client_expiry

This tool finds OIDC clients (with a non-empty Creator, i.e. user-created clients) whose configured secret expiry exceeds a minimum threshold, reports their expiry status, and can optionally reduce their expiry to that threshold.

Usage:
```
update_oidc_client_expiry -expiry 180 -update
```

Flags:
- `-expiry` (optional, default `180`): minimum number of days for secret expiry; clients configured with an expiry greater than this are reported (and updated, if `-update` is set).
- `-update` (optional, default `false`): if set, matching clients have their `SecretExpirationDays` updated to the `-expiry` value.

Dry-run vs update:
- Without `-update`, the tool only logs each matching client's current expiry setting and whether its secret has already expired (and how many days ago) or will expire in the future (and in how many days) - no changes are made.
- With `-update`, matching clients are updated via the API to set `SecretExpirationDays` to the `-expiry` value.

No file inputs or outputs - all data is read from and written to Cx1 via the API; results are only reported via log output.
