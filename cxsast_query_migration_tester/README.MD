# cxsast_query_migration_tester

This tool migrates custom queries (corp/team/project-level overrides and brand-new queries) from a legacy Checkmarx SAST server into CheckmarxOne. It reads the SAST query collection via SOAP, works out the dependency graph and team hierarchy, merges chained team/project query overrides into a single CxQL body (using the bundled query-code merger, ported from go-querymerger), and pushes the result into Cx1 using WebAudit/QueryEditor sessions. Since Cx1 does not support arbitrary team-level queries the same way SAST does, matching queries are pushed as Application-level overrides in a dedicated "CxPSEMEA-Query Migration" project/application, and project-level overrides are pushed onto that same project.

The tool runs in two steps: first generate a migration plan file, then execute it.

## Usage

Step 1 - generate the list of queries to migrate:

```
cxsast_query_migration_tester -cx1 cx1url -iam iamurl -tenant tenant -apikey key -sast sasturl -user sastuser -pass sastpass -output queries.json
```

Step 2 - load the generated file and push the queries into CheckmarxOne:

```
cxsast_query_migration_tester -cx1 cx1url -iam iamurl -tenant tenant -apikey key -sast sasturl -user sastuser -pass sastpass -input queries.json
```

Flags:
- `-cx1` (required) - CheckmarxOne platform URL
- `-iam` (required) - CheckmarxOne IAM URL
- `-tenant` (required) - CheckmarxOne tenant
- `-sast` (required) - Checkmarx SAST server URL
- `-user` (required) - Checkmarx SAST username
- `-pass` (required) - Checkmarx SAST password
- `-apikey` - CheckmarxOne API Key (use this or client id/secret)
- `-client` - CheckmarxOne Client ID (alternative to API key)
- `-secret` - CheckmarxOne Client Secret (alternative to API key)
- `-output` (default `queries.json`) - output file to write the migration plan to (step 1)
- `-input` - input file (previously generated migration plan) to load and execute the migration from (step 2)
- `-teamid` - if set, only migrate queries owned by this SAST team ID
- `-projectid` - if set, only migrate queries owned by this SAST project ID
- `-queryid` - if set, only migrate this specific SAST query ID
- `-cx1-project` - optional name of an existing CheckmarxOne project to use as the migration target instead of the default "CxPSEMEA-Query Migration Project"
- `-cx1-app` - optional name of an existing CheckmarxOne application to associate the target project with
- `-log` (default `INFO`) - log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL
- `-proxy` - HTTP proxy to use for outbound requests

At least one of `-apikey`, `-client`+`-secret` must be provided along with `-cx1`, `-iam`, `-tenant`, `-sast`, `-user`, and `-pass`, otherwise the tool exits with an error.

## Generate vs. migrate mode

The tool always connects to both SAST (SOAP) and CheckmarxOne, and always computes the full in-scope query set (corp, team and project custom queries, plus their dependencies and any parent-team queries needed to satisfy overrides).

- **Generate mode** (`-output`, no `-input`): the computed query list (`QueriesList`) is marshaled to JSON and written to the output file. No changes are made in CheckmarxOne. This is the "dry run" / planning step - it lets you review or edit the plan before pushing anything.
- **Migrate mode** (`-input` supplied): the JSON plan file is loaded, query `OwningGroup` references are re-resolved against the current SAST query collection (since the JSON representation doesn't retain them), and then queries are created/overridden in CheckmarxOne. This step performs real changes: it creates the target project/application if needed, opens WebAudit/QueryEditor audit sessions per language (triggering a throw-away scan on embedded sample source files if no completed scan already exists for the target project), creates new tenant-level queries where no override chain exists, and creates Application-level or Project-level overrides with the merged/migrated CxQL source and severity. `-teamid`, `-projectid`, and `-queryid` narrow migrate mode down to a single team, project, or query; if none are given, everything in the plan is migrated. A summary of successes/failures is logged at the end.

## Files

- `file.go` embeds a zip archive (`resourceCodeZip`) of placeholder source files for every supported language. It's used to create a minimal scan on the target Cx1 project when no completed scan already exists, purely so an audit/query-editor session can be opened for that language.
- `querymerge.go` implements the CxQL query-code merger (ported from `go-querymerger`): given a chain of team/project query overrides (from most specific to the corp base), it concatenates and rewrites the source so that `base.<query>()` calls chain correctly, tagging each fragment with its origin (team/project/query/severity/timestamp) and auto-fixing broken or unsafe `base.` call patterns.
- `querieslist.go` defines `QueriesList`, the structure serialized to/from the plan JSON file: `CorpQueriesToCreate`, `CorpQueriesToMigrate`, `TeamQueriesToMigrate` (by team ID), and `ProjectQueriesToMigrate` (by project ID), each holding SAST `Query` objects with their dependency chains resolved.
- `queries.json` (generated) is the migration plan produced by generate mode and consumed by migrate mode; it is not meant to be edited by hand.
