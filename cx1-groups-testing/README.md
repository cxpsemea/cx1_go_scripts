# cx1-groups-testing

Test-data generator/cleanup tool for CheckmarxOne groups. It creates batches of dummy groups named testgroup-## (each with 5 standard child groups) for load/scale testing, and can delete them again afterwards.

## Usage

```
go run . -apikey <key> -cx1 <url> -iam <url> -tenant <tenant> -create -count 100
go run . -apikey <key> -cx1 <url> -iam <url> -tenant <tenant> -delete
```

Connection flags (from Cx1ClientGo, all read via `flag.Parse()`):
- `-apikey` - CheckmarxOne API key (use this or client id/secret)
- `-client` / `-secret` - CheckmarxOne OAuth client ID/secret (alternative to API key)
- `-cx1` - CheckmarxOne platform URL (required)
- `-iam` - CheckmarxOne IAM URL (required)
- `-tenant` - CheckmarxOne tenant name (required)

Tool-specific flags:
- `-count` (default `100`) - number of groups to create, only used when `-create` is set
- `-create` (default `false`) - set this to actually create the groups; otherwise the tool only logs which groups it would create
- `-delete` (default `false`) - set this to actually delete all previously-created groups matching the testgroup-* pattern; otherwise the tool only logs which groups it would delete

## Behavior

Both the create and delete passes run every time the tool is executed. By default (neither `-create` nor `-delete` set), the tool only logs what it would do - no changes are made:

- Delete pass: fetches all groups whose name starts with `testgroup-`. If `-delete` is set, deletes them (including their child groups); otherwise logs which groups would be deleted. This only ever touches groups created by this tool (the `testgroup-` prefix), not arbitrary groups in the tenant.
- Create pass: if `-create` is set, creates `-count` top-level groups named `testgroup-0001`, `testgroup-0002`, ... , each with 5 child groups: `Owners`, `Scanners`, `Reviewers`, `Readers`, `Service Accounts`. If `-create` is not set, logs which groups would be created.

There is no confirmation prompt before create or delete actions run - `-create`/`-delete` are the only gates.
