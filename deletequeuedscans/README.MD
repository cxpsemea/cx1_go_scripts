# deletequeuedscans

This script cancels all scans that are currently sitting in the "Queued" status in a CheckmarxOne (Cx1) tenant. It fetches up to 1000 queued scans and cancels each one by ID.

Usage:
```
deletequeuedscans -apikey your_api_key [-cx1 https://ast.checkmarx.net] [-iam https://iam.checkmarx.net] [-tenant your_tenant]
```

Flags:
- `-apikey`: CheckmarxOne API Key (used if not using client id/secret)
- `-cx1` (optional): CheckmarxOne platform URL, if not already defined in the test config.yaml
- `-iam` (optional): CheckmarxOne IAM URL, if not already defined in the test config.yaml
- `-tenant` (optional): CheckmarxOne tenant, if not already defined in the test config.yaml

There is no dry-run mode and no `-update` flag - the tool always cancels every queued scan it finds, with no confirmation step. Each cancellation is attempted individually; failures are logged as errors but do not stop the run.

Process:
1. Connects to Cx1 using an API key via `Cx1ClientGo.NewAPIKeyClient`.
2. Retrieves up to 1000 scans with status "Queued" (no project filter).
3. Logs the count found, then calls `CancelScanByID` for each one, logging success/failure per scan.

There are no input/output files - all output is written to stdout via the logger.
