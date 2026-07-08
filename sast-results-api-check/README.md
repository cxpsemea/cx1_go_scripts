# sast-results-api-check

This tool compares SAST findings returned by two different Cx1 APIs (`api/sast-results` and `api/results`) for a single scan, to detect inconsistencies between them (missing entries on either side, or mismatched severity/state/status for matching entries). It can optionally attempt to fix mismatches by reapplying the last predicate.

Usage:
```
sast-results-api-check -scan <scanID> -update -comment "Fix" -log info
```

Flags:
- `-scan` (required): scan ID to check. If omitted, the tool exits with a fatal error.
- `-update` (optional, default `false`): if a finding's severity/state/status differs between the two APIs, reapply its last predicate (via a PNE-then-revert cycle) to try to force consistency. Without this flag, mismatches are only logged.
- `-comment` (optional): comment text to use when temporarily setting a finding to PNE during the fix. If not provided, the tool uses the finding's existing last-predicate comment when reverting, or "Importer Triage Fix" if none exists.
- `-log` (optional, default `info`): log level - `trace`, `debug`, `info`, `warning`, `error`, or `fatal`.

Behavior:
- Fetches results from both `api/sast-results` (`GetAllScanSASTResultsByID`) and `api/results` (`GetAllScanResultsByID`) for the given scan, and matches them by SimilarityID + result hash.
- Logs a warning for any SimilarityID/hash pair present in one API's results but not the other.
- For matched pairs, compares Severity, State, and Status; any mismatch is logged as an error, and if `-update` is set, the tool reapplies the last predicate (set to PROPOSED_NOT_EXPLOITABLE, then reverted to the original state) to attempt to resolve the inconsistency.

No dry-run flag beyond `-update` - without it, the tool only reports inconsistencies; no file inputs or outputs are used, all data comes from the Cx1 API.
