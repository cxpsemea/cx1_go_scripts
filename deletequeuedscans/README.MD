# deletequeuedscans

This script cancels all scans that are currently sitting in the "Queued" status in a CheckmarxOne (Cx1) tenant. It fetches up to 1000 queued scans and cancels each one by ID.

Usage:
```
deletequeuedscans -apikey your_api_key [-cx1 https://ast.checkmarx.net] [-iam https://iam.checkmarx.net] [-tenant your_tenant] -update
```

Flags:
- `-apikey`: CheckmarxOne API Key (used if not using client id/secret)
- `-cx1` (optional): CheckmarxOne platform URL, if not already defined in the test config.yaml
- `-iam` (optional): CheckmarxOne IAM URL, if not already defined in the test config.yaml
- `-tenant` (optional): CheckmarxOne tenant, if not already defined in the test config.yaml
- `-update` (default `false`): set this to actually cancel the queued scans. If not set, the tool only logs a warning plus which scans it would cancel - no changes are made.

**WARNING**: this tool will cancel all queued scans. Use with caution, this is irreversible. There is no confirmation prompt beyond the `-update` flag; once set, each cancellation is attempted individually and failures are logged as errors but do not stop the run.

Process:
1. Connects to Cx1 using an API key via `Cx1ClientGo.NewAPIKeyClient`.
2. Retrieves up to 1000 scans with status "Queued" (no project filter).
3. Logs the count found, then for each one either calls `CancelScanByID` (if `-update` is set) or logs that it would cancel the scan (if not), logging success/failure per scan when cancelling.

There are no input/output files - all output is written to stdout via the logger.
