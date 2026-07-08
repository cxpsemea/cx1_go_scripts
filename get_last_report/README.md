# get_last_report

This tool fetches scans run in the last N months, and for each project+branch combination that hasn't already been processed, generates and downloads a SAST CSV report (ScanSummary, ExecutiveSummary, ScanResults) into a `reports` subfolder.

Usage:
```
get_last_report -months 3 -pause 5
```

Flags:
- `-months` (optional, default `3`): number of months back to retrieve scans from.
- `-pause` (optional, default `5`): seconds to pause between report generation requests.

Behavior:
- On first run, fetches all scans since `-months` ago and stores their scan IDs in `scanids.txt` (one per line). On subsequent runs, if `scanids.txt` already exists, it is read instead of re-fetching from the API.
- For each scan ID, only the first scan encountered per project+branch is processed (subsequent scans for an already-seen project+branch are skipped), and only if a report file for that scan doesn't already exist.
- Reports are saved as `reports/<scanid>.csv`.

No dry-run/update flag - this tool always generates and downloads reports for unprocessed project+branch scans; there is no read-only mode.
