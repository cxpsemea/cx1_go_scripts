# createSAMLUser

This is an example/demo script showing how to create a SAML user in Checkmarx One using the Cx1ClientGo client. It is hardcoded to a specific example user and IdP, and is not intended to be reused as-is against other environments. The hardcoded values will come from the IdP - these can be inspected directly in an IdP like KeyCloak, and other IdPs such as Azure may allow these to be determined, however a securely-configured IdP may randomly generate these IDs on-use, in which case it is not possible to pre-define the values and you cannot use the CreateSAMLUser process.

Usage:
```
createSAMLUser -update
```

The only command-line flag is `-update` (default `false`) - set this to actually delete/create the example user. If not set, the script only logs a warning plus what it would do - no changes are made. All other values (user details, IdP alias, and unique SAML user ID) are hardcoded in `main.go` and must be edited directly in the source to adapt this example for another environment.

Behavior (once `-update` is set):
- If a user with email `groucho@cx.local` already exists, it is deleted first.
- A new user (`Groucho Marx`, username `groucho`, email `groucho@cx.local`) is created via `CreateSAMLUser`, associated with the IdP alias `dockerhost` and a hardcoded unique SAML identifier that corresponds to a specific user in the author's own Keycloak IdP - this ID will not work for other environments and must be replaced.

**WARNING**: this tool will delete the existing user `groucho@cx.local` (if present) and recreate it as a SAML user. Use with caution, this is irreversible. Without `-update`, the script is safe to run to preview what would happen.
