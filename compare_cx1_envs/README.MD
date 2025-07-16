This script compares the roles specified in the -roles flag between two environments. 
Usage:
compare_cx1_envs -cx1 cx1url -iam1 iam1url -tenant1 .. -apikey1 .. -cx2 cx2url -iam2 iam2url -tenant2 .. -apikey2 .. -roles "ast-scanner,TestRole,another_role"

Output example:

```
[INFO][2025-07-16 08:45:38.348] Starting
[INFO][2025-07-16 08:45:38.349] Setting log level to INFO
[INFO][2025-07-16 08:45:38.922] Connected client #1 with tenant1 on cx1url
[INFO][2025-07-16 08:45:39.902] Connected client #2 with tenant2 on cx2url
[INFO][2025-07-16 08:45:40.436] Role ast-scanner is the same between tenant1 and tenant2
[INFO][2025-07-16 08:45:40.437]  - 25 sub-roles in common: view-queries, create-scan, view-results, assign-to-application, manage-reports, view-risk-management, create-schedule-scan, update-project, view-projects, view-scans, view-risk-management-tab, view-preset, create-preset, view-applications, update-schedule-scan, view-project-params, update-preset, delete-schedule-scan, update-risk-management, update-scan, view-schedule-scans, view-risk-management-dashboard, view-tenant-params, update-project-params, create-project
[WARNING][2025-07-16 08:45:40.895] Role TestRole is different between tenant1 and tenant2
[INFO][2025-07-16 08:45:40.896]  - 2 sub-roles in common: create-webhook, manage-keys
[WARNING][2025-07-16 08:45:40.896]  - 2 sub-roles are missing from tenant1: add-package, dast-create-environment
[WARNING][2025-07-16 08:45:40.896]  - 4 sub-roles are extra in tenant1: manage-users, manage-clients, view-results, iam-admin

```