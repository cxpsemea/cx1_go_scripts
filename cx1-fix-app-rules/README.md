# cx1-fix-app-rules

This script migrates Checkmarx One applications from old-style project-assignment rules to the modern approach of directly assigning projects to applications, and can optionally remove the old-style rules.

Usage:
```
cx1-fix-app-rules -delete -update
```

Flags:
- `-delete` (default: `false`) - Delete old-type rules (any rule whose type is not `project.name.in`).
- `-update` (default: `false`) - Make changes to project rules. If not set, no changes are made to the system.

For each application, the script:
1. Reads all `project.name.in` rules and collects the project names they reference (values are split on `;`).
2. Compares this list against the projects actually linked to the application by ID (`app.ProjectIds`); any linked project not covered by a `project.name.in` rule is treated as an "extra" project to be explicitly assigned.
3. Logs each extra project that would be assigned to the application.
4. Logs any non-`project.name.in` ("old-style") rules found; if `-delete` is set, these rules are removed from the application.
5. If `-update` is set, the application is saved via the API with the above changes (project assignments and, if `-delete` was set, removed rules). If `-update` is not set, all of the above is only logged - no API calls are made to update applications.
