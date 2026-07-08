# project_group_check

This tool validates that all (or one specific) project's group memberships reference groups that actually exist in the tenant, and can optionally remove references to invalid/non-existent groups.

Usage:
```
project_group_check -update -bulk -project "MyProject" -project-file projects.json -group-file groups.json
```

Flags:
- `-update` (optional, default `false`): apply changes to remove invalid groups from projects. Only takes effect when connecting live to Cx1 (ignored, with an error logged, if `-project-file` and/or `-group-file` were provided for offline mode).
- `-bulk` (optional, default `false`): if set, all in-scope projects are processed without pausing. If not set, the tool pauses after each project and prompts `y/n` to continue (prompt is skipped if `-project` is used to target a single project).
- `-project` (optional, default `""`): name of a specific project to check; if empty, all projects are checked.
- `-project-file` (optional, default `""`): path to a JSON file containing a CheckmarxOne `/api/projects` response (`{"projects": [...]}`) to use instead of a live API call.
- `-group-file` (optional, default `""`): path to a JSON file containing a CheckmarxOne IAM groups admin API response (a JSON array of groups) to use instead of a live API call.

Dry-run vs update:
- Without `-update`, the tool only reports which projects have invalid group references (and lists all distinct invalid group IDs found) - no changes are made.
- With `-update`, and when running online (no file inputs provided), projects with invalid groups are updated via the API to keep only the valid groups.
- If both `-project-file` and `-group-file` are provided, the tool runs fully offline against the supplied JSON data; in this mode `-update` has no effect even if set.

Input file formats:
- `-project-file`: JSON object with a `projects` array, matching the shape of Cx1's `GET /api/projects` response.
- `-group-file`: JSON array of group objects, matching the shape of Cx1 IAM's groups admin endpoint response.
