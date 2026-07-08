# createSAMLMappers

This script creates the standard set of default attribute mappers (firstname, lastname, username, role, group, email) on an existing SAML identity provider in Checkmarx One. The structure of these attribute mappers is tailored specifically for a KeyCloak SAML provider - using Azure or other IdP will require changes to the mapper configurations depending on the IdP configuration. 

Usage:
```
createSAMLMappers -provider-alias "MySamlIdP" -update
```

Flags:
- `-provider-alias` (default: `""`) - Alias (display name) of the SAML IdP which already exists in Checkmarx One. This is effectively required: if the IdP alias doesn't match an existing provider, the script fails when looking it up.
- `-update` (default `false`) - set this to actually add the mappers to the identity provider. If not set, the script only logs a warning plus which mappers it would add - no changes are made.

The script looks up the SAML identity provider by its alias, then, if `-update` is set, creates default mappers for `firstname`, `lastname`, `username`, `role`, `group`, and `email` on that provider. **WARNING**: this may overwrite existing mappers with the same name - use with caution. Without `-update`, the script is safe to run to preview which mappers would be added.
