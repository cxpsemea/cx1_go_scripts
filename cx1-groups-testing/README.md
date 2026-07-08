# cx1-groups-testing

Test-data generator/cleanup tool for CheckmarxOne groups. It creates batches of dummy groups (each with 5 standard child groups) for load/scale testing, and can delete them again afterwards.

## Usage

```
go run . -apikey <key> -cx1 <url> -iam <url> -tenant <tenant> -count 100
go run . -apikey <key> -cx1 <url> -iam <url> -tenant <tenant> -delete
```

Connection flags (from Cx1ClientGo, all read via `flag.Parse()`):
- `-apikey` - CheckmarxOne API key (use this or client id/secret)
- `-client` / `-secret` - CheckmarxOne OAuth client ID/secret (alternative to API key)
- `-cx1` - CheckmarxOne platform URL (required)
- `-iam` - CheckmarxOne IAM URL (required)
- `-tenant` - CheckmarxOne tenant name (required)

Tool-specific flags:
- `-count` (default `100`) - number of groups to create, only used when `-delete` is not set
- `-delete` (default `false`) - toggle to delete all previously-created groups instead of creating new ones

## Behavior

There is no dry-run mode - the tool always makes changes when run.

- Default mode (no `-delete`): creates `*NumberOfGroups*` top-level groups named `testgroup-0001`, `testgroup-0002`, ... up to the `-count` value, each with 5 child groups: `Owners`, `Scanners`, `Reviewers`, `Readers`, `Service Accounts`.
- `-delete` mode: fetches all groups whose name starts with `testgroup-` and deletes them (including their child groups). This only ever touches groups created by this tool (the `testgroup-` prefix), not arbitrary groups in the tenant.

There is no confirmation prompt before create or delete actions run.
