# add_oidc_client_email

This script adds notification email addresses to existing OIDC clients in Checkmarx One, based on a list read from an input file.

Usage:
```
add_oidc_client_email -emails emails.csv -update
```

Flags:
- `-emails` (default: `emails.csv`) - Input file containing lines with: `<client_id>,<emails;to;add>`
- `-update` (default: `false`) - Enable OIDC client email update

If `-update` is not set, the script only logs what it would have changed (which clients would receive which new email addresses) and makes no changes. When `-update` is set, it calls the API to update each OIDC client whose notification emails list needs a new entry.

For each client, the script fetches the OIDC client by name, and adds any email from the input file that is not already present in the client's notification emails.

Input file format (`emails.csv`):
- CSV file, one record per line, comma-separated.
- Column 1: OIDC client ID/name.
- Column 2: one or more email addresses to add, separated by semicolons (`;`).
- Any additional columns are ignored.
- No header row.

Example:
```
cx1e2e_admin,michael.kubiaczyk@checkmarx.com;koobze@yahoo.com
```
