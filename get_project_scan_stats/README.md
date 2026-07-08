# get_project_scan_stats

This tool retrieves the number of projects created and scans started per day, over the last N days, and writes the daily counts to two CSV files.

Usage:
```
get_project_scan_stats -days 30
```

Flags:
- `-days` (optional, default `30`): number of days back to report on (from midnight, `-days` days before today, through today).

Output files (written to the current directory, semicolon-delimited, Excel-friendly with a `sep=;` marker line):
- `projects.csv`: header `Date;# Projects Created;`, one row per day in the range with the count of projects created that day.
- `scans.csv`: header `Date;# Scans Started;`, one row per day in the range with the count of scans started that day.

No dry-run/update flag - this tool is read-only against Cx1 and always (re)writes `projects.csv` and `scans.csv`.
