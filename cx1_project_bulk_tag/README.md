# cx1_project_bulk_tag

Bulk-adds and/or removes a tag on a list of CheckmarxOne projects, to trigger a refresh of the project (e.g. for the cx336-ui-fix workaround). Default is dry-run/report-only.

## Usage

```
go run . -apikey <key> -cx1 <url> -iam <url> -tenant <tenant> -projects projectIds.txt -tag cx336-ui-fix -update
```

Connection flags (from Cx1ClientGo):
- `-apikey` - CheckmarxOne API key (use this or client id/secret)
- `-client` / `-secret` - CheckmarxOne OAuth client ID/secret (alternative to API key)
- `-cx1` - CheckmarxOne platform URL (required)
- `-iam` - CheckmarxOne IAM URL (required)
- `-tenant` - CheckmarxOne tenant name (required)

Tool-specific flags:
- `-log` (default `INFO`) - log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL
- `-projects` (default `projectIds.txt`) - file containing one project ID per line
- `-tag` (default `cx336-ui-fix`) - tag key to add and/or remove
- `-update` (default `false`) - apply the change; without it, the tool only reports what it would do
- `-add` (default `false`) - only add the tag, skip the removal step
- `-remove` (default `false`) - only remove the tag, skip the add step
- `-delay` (default `1000`) - delay in milliseconds after each project is processed (only when `-update` is set)

## Behavior

Reads project IDs from the `-projects` file (plain text, one UUID per line, no header).

Without `-update`, the tool only logs "Would update project ... by adding & removing tag ...".

With `-update`, for each project:
1. Adds `-tag` to the project's tags and calls `UpdateProject` (unless `-remove` is set)
2. Removes `-tag` from the project's tags and calls `UpdateProject` (unless `-add` is set)
3. Sleeps `-delay` milliseconds before moving to the next project

Unlike `cx1_app_bulk_tag`, this tool runs straight through the whole list with no interactive confirmation prompts between projects.
