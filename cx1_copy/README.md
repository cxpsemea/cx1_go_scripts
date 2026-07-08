# cx1_copy

Copies SAST custom queries and/or presets from one CheckmarxOne tenant ("src") to another ("dest"). Useful for syncing query overrides and presets between environments (e.g. staging to production, or between two customer tenants).

## Usage

```
go run . -scope queries,presets ^
  -src-apikey %SRC_KEY% -src-cx https://src.ast.checkmarx.net -src-iam https://src.iam.checkmarx.net -src-tenant src_tenant ^
  -dest-apikey %DEST_KEY% -dest-cx https://dest.ast.checkmarx.net -dest-iam https://dest.iam.checkmarx.net -dest-tenant dest_tenant
```

(see `run.bat` for a working example)

Connection flags - two independent sets, one per environment (this tool builds its own Cx1ClientGo clients directly with `NewAPIKeyClient`/`NewOAuthClient`, it does not use Cx1ClientGo's flag-based `NewClient`):

Source (`src`):
- `-src-apikey` - API key (use this or client id/secret)
- `-src-client` / `-src-secret` - OAuth client ID/secret (alternative to API key)
- `-src-cx` - CheckmarxOne platform URL
- `-src-iam` - CheckmarxOne IAM URL
- `-src-tenant` - tenant name
- `-src-proxy` - optional proxy URL for connections to the src environment (TLS verification is disabled when a proxy is set)

Destination (`dest`):
- `-dest-apikey` - API key (use this or client id/secret)
- `-dest-client` / `-dest-secret` - OAuth client ID/secret (alternative to API key)
- `-dest-cx` - CheckmarxOne platform URL
- `-dest-iam` - CheckmarxOne IAM URL
- `-dest-tenant` - tenant name
- `-dest-proxy` - optional proxy URL for connections to the dest environment (TLS verification is disabled when a proxy is set)

Tool flags:
- `-log` (default `INFO`) - log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL
- `-scope` (required) - comma-separated list of items to copy: `queries`, `presets`
- `-languages` - optional comma-separated list to restrict query migration to specific languages, e.g. `javascript,java`
- `-presets` - optional comma-separated list to restrict preset migration to specific preset names, e.g. `My_Preset1,My_Preset2`

The tool exits with a fatal error if `-scope` is empty.

## Behavior

There is no separate dry-run flag - whatever is listed in `-scope` is applied directly to the `dest` tenant. There is no confirmation prompt.

### Queries (`-scope` includes `queries`)

1. Checks that the `CVSS_V3_ENABLED` feature flag matches between src and dest; if it differs, query migration is aborted entirely (queries carry CVSS severity data that isn't comparable across the flag states).
2. Creates (or reuses) a project named `CxPSEMEA-Query Migration Project` on both src and dest, used to open a SAST audit session (queries can only be edited/read via an audit session tied to a scanned project). If the project has no completed scan for a given language, it uploads a small bundled sample-code zip (per-language snippet embedded in the binary) and runs a real SAST scan to create one.
3. For each SAST language (optionally filtered by `-languages`), compares custom (non-default) queries between src and dest at the tenant/project audit level:
   - If a query doesn't exist in dest, creates a new tenant-level query or a tenant override of the product query in dest, copying the source code and severity from src.
   - If a query exists in both but the source code or severity differs, updates the dest query's source and/or severity to match src.
   - If source and severity already match, no change is made and it's logged as "same between environments".
4. Deletes the temporary audit sessions on both tenants when done.

### Presets (`-scope` includes `presets`)

1. Fetches all SAST presets (with full query-family contents) from both src and dest.
2. For each src preset optionally filtered by `-presets`:
   - If a preset with the same name exists in dest and its query set already matches (subset both ways), no change is made.
   - If a preset with the same name exists but differs, its query families are overwritten with the src preset's query families (`UpdateSASTPreset`).
   - If no preset with that name exists in dest, a new preset is created with the same name, description, and query set (`CreateSASTPreset`).

## Input/output files

- `src-queries.json` / `dest-queries.json` in this folder appear to be example/reference dumps of query data, not files read by the code at runtime — the tool only interacts with Cx1 via the API, it does not read or write these JSON files itself.
- No other input files are read; there are no CLI-writable output files (all results go to the log and directly to the dest tenant).
