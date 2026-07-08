# delete_everything

Deletes all projects, applications, groups, and/or presets in a CheckmarxOne tenant. Destructive and irreversible - there is no dry-run mode and no confirmation prompt.

## Usage

```
go run . -apikey <key> -cx1 <url> -iam <url> -tenant <tenant> -scope projects,applications,groups,presets
```

Connection flags (from Cx1ClientGo):
- `-apikey` - CheckmarxOne API key (use this or client id/secret)
- `-client` / `-secret` - CheckmarxOne OAuth client ID/secret (alternative to API key)
- `-cx1` - CheckmarxOne platform URL (required)
- `-iam` - CheckmarxOne IAM URL (required)
- `-tenant` - CheckmarxOne tenant name (required)

Tool-specific flags:
- `-scope` (required) - comma-separated list of item types to delete. Valid values: `projects`, `applications`, `groups`, `presets`. The tool exits with a fatal error if this is empty, and logs an error (but continues) for any unrecognized value in the list.

## WARNING - destructive, no confirmation, no dry-run

This tool takes irreversible action immediately on every matching item, with no preview step and no "are you sure?" prompt:

- `projects` - fetches **all** projects in the tenant (`GetProjects(0)`) and calls `DeleteProject` on every one of them.
- `applications` - fetches **all** applications in the tenant (`GetApplications(0)`) and calls `DeleteApplication` on every one of them.
- `groups` - fetches **all** groups in the tenant (`GetGroups()`) and calls `DeleteGroup` on every one of them.
- `presets` - fetches **all** custom presets in the tenant (`GetPresets(count)`) and calls `DeletePreset` on every one of them.

There is no filtering by name/tag/ID - this is unconditional, tenant-wide deletion for whichever scopes are listed. It is intended for use against disposable/test tenants only. Do not run this against a production or shared tenant.
