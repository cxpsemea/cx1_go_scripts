# cx1_app_bulk_tag

Bulk-adds and/or removes a tag on a list of CheckmarxOne applications, to trigger a full re-index/refresh of the application's projects (e.g. for the cx336-ui-fix workaround). Default is dry-run/report-only.

## Usage

```
go run . -apikey <key> -cx1 <url> -iam <url> -tenant <tenant> -apps appIds.txt -tag cx336-ui-fix -update
```

Connection flags (from Cx1ClientGo):
- `-apikey` - CheckmarxOne API key (use this or client id/secret)
- `-client` / `-secret` - CheckmarxOne OAuth client ID/secret (alternative to API key)
- `-cx1` - CheckmarxOne platform URL (required)
- `-iam` - CheckmarxOne IAM URL (required)
- `-tenant` - CheckmarxOne tenant name (required)

Tool-specific flags:
- `-log` (default `INFO`) - log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL
- `-apps` (default `appIds.txt`) - file containing one application ID per line
- `-tag` (default `cx336-ui-fix`) - tag key to add and/or remove
- `-update` (default `false`) - apply the change; without it, the tool only reports what it would do
- `-add` (default `false`) - only add the tag, skip the removal step
- `-remove` (default `false`) - only remove the tag, skip the add step
- `-delay` (default `5000`) - delay in milliseconds, multiplied by the number of projects in the application, applied after each add/remove API call
- `-all` (default `false`) - run through all applications without pausing for confirmation between each one

## Behavior

Reads application IDs from the `-apps` file (plain text, one UUID per line, no header).

Without `-update`, the tool only logs "Would update application ... by adding & removing tag ...".

With `-update`, for each application (skipping applications with zero projects):
1. Adds `-tag` to the application's tags and calls `UpdateApplication` (unless `-remove` is set)
2. Removes `-tag` from the application's tags and calls `UpdateApplication` (unless `-add` is set)
3. Sleeps `delay * project_count` milliseconds after each of the above calls

Unless `-all` is set, after each application (except the last) the tool prompts interactively on stdin: enter `n`/`no` to stop, `a`/`all` to continue without further prompts, or `d=<number>` to change the delay value mid-run.
