# cxsast_query_dependency_check

This script inspects a CxSAST (legacy Checkmarx SAST) instance and reports the dependency relationships between custom queries, to help plan a CxSAST-to-Cx1 query migration. It connects to CxSAST, retrieves all queries (via SOAP), teams, and projects, links overridden queries back to their base queries, and detects which custom queries depend on other queries (both custom dependencies and dependencies on product-default queries). It then logs, for each custom query, the list of queries it depends on (marking each as `NEW` or `Override` relative to its base query).

Usage:
```
cxsast_query_dependency_check -sast https://sast.url -user username -pass password [-projectId 123 | -projectName "Name"] [-nocorp] [-noprod] [-log INFO] [-proxy http://proxy:port]
```

Flags:
- `-sast` (required): CxSAST server URL
- `-user` (required): CxSAST username
- `-pass` (required): CxSAST password
- `-projectId` (optional): CxSAST Project ID; if set (or `-projectName`), only that project's query dependency chain (project + team-hierarchy + corp queries) is analyzed
- `-projectName` (optional): CxSAST Project Name; alternative to `-projectId`
- `-nocorp`: Don't show Corporate-query dependencies (use if all corp queries were already migrated)
- `-noprod`: Don't show Product-default query dependencies (use if the CxSAST and Cx1 product query versions match)
- `-proxy` (optional): HTTP proxy to use for the CxSAST connection
- `-log` (default `INFO`): Log level (TRACE, DEBUG, INFO, WARNING, ERROR, FATAL)

This is a read-only reporting tool - it makes no changes to CxSAST. It only queries data and logs findings.

Behavior:
- If `-projectId` or `-projectName` is given, the tool builds the query dependency lists specific to that project: all project-level custom queries, all custom queries owned by teams in the project's team hierarchy, and (unless `-nocorp`) all corp-level queries. It then logs each query's override chain and any custom/product dependencies it has.
- If neither is given, the tool scans every custom, valid query across all corp/team/project query groups in the system and reports dependencies the same way.

There are no input/output files - all results are written to stdout via the logger.
