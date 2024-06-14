This repo contains some example scripts and/or handy utilities for use with CheckmarxOne.

- cx1deletequeries: deletes all custom queries on a tenant. WARNING: this can't be undone, they are all permanently deleted. To do this, a new project will be created and scans will be run - this is required to create audit sessions for each language where custom queries exist.
- createSAMLMappers: creates mappers that work for a Keycloak SAML IdP. It looked for an existing SAML IdP ("dockerhost") and adds the mappers, you can use this to add mappers to your own SAML IdP in cx1
- createSAMLUser: creates a SAML user in cx1, using the SAML IdP-internal IDs for a user. These IDs will depend on your SAML configuration and must be obtained from your SAML IdP in the first place.
- deletescans: deletes/cancels scans from the Queue, 1000 scans at a time.