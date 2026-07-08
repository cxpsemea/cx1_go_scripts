# createSAMLUser

This is an example/demo script showing how to create a SAML user in Checkmarx One using the Cx1ClientGo client. It is hardcoded to a specific example user and IdP, and is not intended to be reused as-is against other environments. The hardcoded values will come from the IdP - these can be inspected directly in an IdP like KeyCloak, and other IdPs such as Azure may allow these to be determined, however a securely-configured IdP may randomly generate these IDs on-use, in which case it is not possible to pre-define the values and you cannot use the CreateSAMLUser process.

Usage:
```
createSAMLUser
```

There are no command-line flags. The script has no configurable input - all values (user details, IdP alias, and unique SAML user ID) are hardcoded in `main.go` and must be edited directly in the source to adapt this example for another environment.

Behavior:
- If a user with email `groucho@cx.local` already exists, it is deleted first.
- A new user (`Groucho Marx`, username `groucho`, email `groucho@cx.local`) is created via `CreateSAMLUser`, associated with the IdP alias `dockerhost` and a hardcoded unique SAML identifier that corresponds to a specific user in the author's own Keycloak IdP - this ID will not work for other environments and must be replaced.

There is no dry-run mode - running the script always attempts to delete/create the user.
