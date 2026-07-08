# deletequeries

This script deletes all custom query overrides (audit-created custom queries/overrides) from a CheckmarxOne (Cx1) tenant, optionally including project-level custom queries for a specific project. For each language that has custom queries, it opens (or reuses) an Audit Session against a scan, then deletes each custom query override by its editor key.

Usage:
```
deletequeries -project "Optional Project Name"
```

Flags:
- `-project` (optional): Name of a project in CheckmarxOne. If provided, project-level custom queries (`AUDIT_QUERY_PROJECT`) for that project are also included in the deletion; the project's own audit sessions are used to delete them.

Note: CheckmarxOne connection parameters (base URL, IAM URL, tenant, API key/client credentials) are supplied via `Cx1ClientGo.NewClient`, which reads them from the client's standard config/environment rather than a flag defined in this tool.

There is no dry-run mode - this tool always attempts to delete every custom query override it finds. Each deletion is attempted individually; failures are logged as errors but do not stop the run.

Process:
1. Connects to Cx1 and (if `-project` is set) resolves the named project.
2. Retrieves the full query collection, plus project-level audit queries if a project was specified, and filters down to the custom query collection.
3. For each language present among the custom queries, it ensures a valid Audit Session exists (creating one if needed, or refreshing/recreating it if the current session doesn't cover the language or its keep-alive fails).
   - Creating a session for a language requires a completed scan on a branch named after that language. If no such scan exists for the "CxPSEMEA-Query Migration Project" (or the specified `-project`), the tool builds a minimal source zip for that language and triggers a new SAST scan, then waits for it to complete before opening the audit session.
4. Deletes every custom query in that language via `DeleteQueryOverrideByKey`, logging success or failure per query.
5. On exit, the last open Audit Session is deleted via a deferred cleanup call.

File formats:
- `file.go` contains no logic - it only defines `resourceCodeZip`, a large embedded byte array representing a ZIP archive with one placeholder source file per supported language (e.g. `code/javaFile.java`, `code/pythonFile.py`, etc.). `main.go`'s `makeZip` function extracts only the file(s) matching the target language from this embedded archive and repackages them into a new zip, which is uploaded to trigger the throwaway scan used to open an audit session.
