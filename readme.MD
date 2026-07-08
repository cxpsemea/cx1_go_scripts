# cx1_go_scripts

This repo contains example scripts and utilities for use with CheckmarxOne (Cx1), built on top of [Cx1ClientGo](https://github.com/cxpsemea/Cx1ClientGo). Each subfolder is a standalone Go tool with its own `README.md` describing its specific flags and behavior in detail - this top-level README explains the patterns shared across (almost) all of them.

## Shared patterns

**Connecting to CheckmarxOne**

Most tools connect using the standard Cx1ClientGo flag set:
- `-apikey` - a CheckmarxOne API key, **or**
- `-client` / `-secret` - an OAuth client ID/secret pair

along with `-cx1` (platform URL), `-iam` (IAM URL), and `-tenant` (tenant name). Some tools also accept these via a `config.yaml` instead of flags, or fall back to it when a flag is omitted. A few tools (e.g. `cxsast_query_migration_tester`) additionally connect to a legacy CxSAST server via `-sast`/`-user`/`-pass`, and `cx1_copy` connects to two separate CheckmarxOne environments at once (`-src-*` / `-dest-*`).

**Dry-run by default**

Nearly every tool that can change data defaults to a **read-only/dry-run mode**: it logs what it *would* do (which items would be created, updated, deleted, or tagged) and makes no API calls that change state. An explicit flag - typically `-update`, but sometimes `-create` and/or `-delete` - must be set to actually apply changes. There is generally no confirmation prompt beyond these flags, so it's worth reviewing the dry-run output first, especially for the destructive tools noted below.

A few tools are exceptions to this pattern and always make changes or are always read-only - these are called out explicitly per tool below.

**Logging**

Most tools accept a `-log` flag (`TRACE`, `DEBUG`, `INFO`, `WARNING`, `ERROR`, `FATAL`, default `INFO`) and write all output to stdout via the logger; only a few write files.

## Tools

### Queries

- [query-creator](query-creator/README.md) - bulk-creates or updates tenant-level custom queries in Cx1 from a folder of `.cs` files (`Language/Group/Query.cs`), or deletes matching queries with `-delete`. Dry-run unless `-create` or `-delete` is set.
- [get-cx1-queries](get-cx1-queries/README.MD) - read-only export of all custom query source/severity (Tenant/Application/Project level) in scope for a project, written to `queries.json`.
- [deletequeries](deletequeries/README.MD) - deletes all tenant-level (and optionally project-level) custom query overrides. Dry-run unless `-update` is set. Irreversible once run.
- [cx1_copy](cx1_copy/README.md) - copies custom SAST queries and/or presets from one Cx1 tenant to another (`-src-*`/`-dest-*` connections). No dry-run flag - whatever is in `-scope` is applied directly to the destination.

### Legacy CxSAST -> Cx1 migration

- [cxsast_query_dependency_check](cxsast_query_dependency_check/README.MD) - read-only report of custom query dependency chains in a CxSAST instance, to help plan a migration.
- [cxsast_query_migration_tester](cxsast_query_migration_tester/README.MD) - migrates custom CxSAST queries into Cx1. Two-step: generate a migration plan JSON (`-output`, read-only), then execute it (`-input`, makes real changes - no separate `-update` flag).

### Tags

- [cx1-app-to-tag](cx1-app-to-tag/README.md) - sets a tag on each project listing its linked application name(s). Dry-run unless `-update` is set.
- [cx1_app_bulk_tag](cx1_app_bulk_tag/README.md) - adds/removes a tag on a list of applications (e.g. to force a UI refresh). Dry-run unless `-update` is set; prompts interactively between applications unless `-all` is set.
- [cx1_project_bulk_tag](cx1_project_bulk_tag/README.md) - same as above, but for a list of projects, and runs straight through with no interactive prompt.

### Projects, applications, and groups

- [cx1-fix-app-rules](cx1-fix-app-rules/README.md) - migrates applications from old-style project-assignment rules to direct project assignments, optionally deleting the old rules (`-delete`). Dry-run unless `-update` is set.
- [cx1_project_primary_branch](cx1_project_primary_branch/README.MD) - sets the primary branch on projects that don't have one but have scan history. Supports several input modes (all projects, an application, a project ID file, or a pre-computed project/branch pair file).
- [project_group_check](project_group_check/README.md) - finds project group memberships pointing at deleted groups, and can remove them (`-update`). Supports an offline mode using saved JSON dumps instead of live API calls.
- [cx1-groups-testing](cx1-groups-testing/README.md) - creates or deletes batches of dummy test groups (`testgroup-*`) for load/scale testing. Gated by `-create`/`-delete`.
- [delete_everything](delete_everything/README.md) - **destructive**: unconditionally deletes *all* projects, applications, groups, and/or presets in the tenant scopes listed via `-scope`. Dry-run unless `-update` is set; no confirmation prompt and no filtering - intended for disposable/test tenants only.

### SAML / OIDC

- [createSAMLMappers](createSAMLMappers/README.md) - adds standard Keycloak-style SAML attribute mappers to an existing SAML IdP. Dry-run unless `-update` is set; may overwrite existing mappers of the same name.
- [createSAMLUser](createSAMLUser/README.md) - example/demo script that deletes and recreates a hardcoded example SAML user; not intended for reuse as-is. Dry-run unless `-update` is set.
- [add_oidc_client_email](add_oidc_client_email/README.md) - adds notification email addresses to OIDC clients from a CSV file. Dry-run unless `-update` is set.
- [update_oidc_client_expiry](update_oidc_client_expiry/README.md) - reports (and optionally reduces, with `-update`) OIDC client secret expiry settings that exceed a given threshold.

### Scans, results, and reports

- [deletequeuedscans](deletequeuedscans/README.MD) - cancels all queued scans (up to 1000). Dry-run unless `-update` is set. Irreversible once run.
- [reaudit-findings](reaudit-findings/README.md) - forces Cx1 to re-run automated triage on previously-triaged SAST findings by briefly flipping state to PNE and reverting. Dry-run unless `-update` is set.
- [sast-results-api-check](sast-results-api-check/README.md) - compares SAST findings between the `api/sast-results` and `api/results` endpoints for a scan and reports mismatches; can attempt a fix (`-update`) by reapplying the last predicate.
- [get_last_report](get_last_report/README.md) - downloads SAST CSV reports for recent scans (one per project+branch) into a `reports` folder. Always generates/downloads - no dry-run mode.
- [get_project_scan_stats](get_project_scan_stats/README.md) - read-only: writes daily project-creation and scan-start counts to `projects.csv`/`scans.csv`.

### Environment comparison

- [compare_cx1_envs](compare_cx1_envs/README.MD) - read-only comparison of role/permission definitions between two Cx1 environments.

## A note on destructive tools

Several tools above (`delete_everything`, `deletequeries`, `deletequeuedscans`, `createSAMLUser`) can perform irreversible deletions with no confirmation prompt beyond the `-update` flag. Always run without `-update` first to review what would happen, and avoid running these against production or shared tenants unless that's specifically the intent.
