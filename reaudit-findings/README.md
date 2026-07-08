# reaudit-findings

This tool forces Checkmarx One to re-run its automated triage/audit logic on SAST findings that were previously set by an import process (or, optionally, on all previously-triaged findings), by briefly flipping each finding's state to "Proposed Not Exploitable" (PNE) and then restoring its original state. This is a workaround to trigger a re-audit while preserving the original triage decision.

Usage:
```
reaudit-findings -app "MyApplication" -update
reaudit-findings -proj "MyProject" -update -comment "Reauditing"
reaudit-findings -project-ids "id1,id2,id3" -update
reaudit-findings -project-names "ProjA,ProjB" -update
```

Flags:
- `-app` (optional): name of an application to process (all projects in the application).
- `-proj` (optional): name of a single project to process.
- `-project-ids` (optional): comma-separated list of project IDs to process.
- `-project-names` (optional): comma-separated list of project names to process.
  - Exactly one of `-app`, `-proj`, `-project-ids`, `-project-names` must be provided, checked in that order of priority. If none are provided, the tool exits with a fatal error.
- `-update` (optional, default `false`): actually apply the PNE-then-revert change. Without it, the tool only reports which findings would be updated.
- `-comment` (optional): comment text to set when temporarily changing state to PNE and when reverting. If not provided, the tool uses a default PNE comment ("Temporarily marking as PNE to trigger re-audit with original state") and, on revert, the finding's original comment if it had one, otherwise "Reauditor Triage Fix" (when `-reaudit-all` is set) or "Importer Triage Fix" (otherwise).
- `-log` (optional, default `info`): log level - `trace`, `debug`, `info`, `warning`, `error`, or `fatal`.
- `-history` (optional, default `false`): analyze the full predicate history for each finding rather than just the latest predicate. When set (and `-reaudit-all` is not set), a finding is only reaudited if its state has not changed since the last time the `importer` user set a predicate on it (i.e. no manual changes since import). Ignored if `-reaudit-all` is set.
- `-reaudit-all` (optional, default `false`): re-audit every finding that has any predicate set (last state non-empty), regardless of who set it - not just findings last triaged by the `importer` user. When set, `-history` is ignored.
- `-delay` (optional, default `5000`): milliseconds to wait after setting a finding to PNE before reverting it to its original state.
- `-intra-delay` (optional, default `500`): milliseconds intended as a delay between processing individual findings, to avoid flooding the API. Note: as of the current code, this value is parsed and stored but not actually applied anywhere in the processing loop.
- `-buffer` (optional, default `100`): number of findings processed concurrently (size of the goroutine semaphore) per project.

Scope logic per finding (only the project's latest **completed SAST** scan is considered; if no completed SAST scan exists, the project is skipped):
- Default (`-history` and `-reaudit-all` both unset): only findings whose latest predicate was created by the `importer` user are in scope.
- `-history` set, `-reaudit-all` unset: only findings where the state has not changed since the last `importer`-created predicate are in scope (using the full predicate history, not just the latest predicate).
- `-reaudit-all` set: any finding with a non-empty last predicate state is in scope, regardless of who created it (this takes priority over `-history`).

Dry-run vs update:
- Without `-update`, the tool logs which findings are in scope and would be updated, but makes no changes.
- With `-update`, in-scope findings are set to PNE, held for `-delay` ms, then reverted to their original state/comment.

No file inputs or outputs - all data is read from and written to Cx1 via the API; results are only reported via log output.
