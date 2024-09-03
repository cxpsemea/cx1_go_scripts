This repo contains some example scripts and/or handy utilities for use with CheckmarxOne.

- cx1deletequeries: deletes all custom queries on a tenant. WARNING: this can't be undone, they are all permanently deleted. To do this, a new project will be created and scans will be run - this is required to create audit sessions for each language where custom queries exist.
- createSAMLMappers: creates mappers that work for a Keycloak SAML IdP. It looked for an existing SAML IdP ("dockerhost") and adds the mappers, you can use this to add mappers to your own SAML IdP in cx1
- createSAMLUser: creates a SAML user in cx1, using the SAML IdP-internal IDs for a user. These IDs will depend on your SAML configuration and must be obtained from your SAML IdP in the first place.
- createSAMLMappers: updates an existing SAML provider in CheckmarxOne and creates some SAML mappers compatible with a Keycloak IdP 
- delete_everything: optionally deletes all projects, applications, presets, and groups
- deletequeries: deletes all tenant-level custom queries and optionally all application- and project-level custom queries if provided with a project name
- deletequeuedscans: deletes/cancels scans from the Queue, 1000 scans at a time.
- project_group_check: checks all projects for membership in groups that no longer exist, and optionally updates the projects to remove the groups.


Below utils for legacy CxSAST but related to CheckmarxOne
- query dependency check: downloads the teams list, project list, and query collection from legacy CxSAST, and tries to identify queries with dependencies on other custom queries, which require special handling when migrating (eg: custom_query1 calls custom_query2 -> need to migrate custom_query2 first before custom_query1)