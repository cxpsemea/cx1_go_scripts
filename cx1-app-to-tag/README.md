# cx1-app-to-tag

This script sets a tag on each Checkmarx One project listing the name(s) of the application(s) it is linked to, so that application membership can be seen via project tags.

Usage:
```
cx1-app-to-tag -tag App -max 0 -sort -separate -clean -update -missing
```

Flags:
- `-tag` (default: `App`) - Tag key to use; the tag value will be set to the application name(s). Must not be empty (the script fails if it is).
- `-max` (default: `0`) - The maximum number of applications that will be set in the tag, per project. Use `0` for no limit.
- `-sort` (default: `false`) - If true, applications will be sorted by name. If false, they will be added in the same order returned by the API.
- `-separate` (default: `false`) - If true, multiple tags named `Key_1 ... Key_N` will be set, one per application. If false, one tag will be used to store a comma-separated list of application names.
- `-clean` (default: `false`) - If true, previous tags matching the provided tag key (or `Key_#` pattern) will be removed before adding new ones; otherwise they are left untouched but may be overwritten by new values. Automatically disabled if `-missing` is also set (the two are incompatible).
- `-update` (default: `false`) - No changes will be made unless this flag is set to true.
- `-missing` (default: `false`) - Update only the projects that are missing the tag - useful if errors caused a few to fail, use this flag to fill the gaps.

If `-update` is not set, the script only logs which projects would be changed and what tags would be added/removed, without calling the update API. When `-update` is set, it calls `UpdateProject` for every project that needs a tag change.

For each project, the script determines its linked applications (by ID, resolved against the full list of applications), builds the tag value(s) accordingly, and applies the `-max`, `-sort`, `-separate`, and `-clean` options as described above. Projects with no linked applications, or where the application IDs don't resolve, are logged and skipped/reported as errors.
