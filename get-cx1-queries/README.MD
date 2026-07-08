# get-cx1-queries

This tool exports the full set of custom query source code (Tenant, Application, and Project level) that is in scope for a given CheckmarxOne project. It connects to CheckmarxOne, finds every custom query language/group/query in scope for the project, opens a WebAudit/QueryEditor session per language (triggering a throw-away scan on embedded sample source if the project has no completed scan yet), fetches the source code and severity at each applicable level, and writes it all to `queries.json`.

## Usage

```
get-cx1-queries -project-name "MyProject"
```
or
```
get-cx1-queries -project-id "12345678-...-uuid"
```

Flags:
- `-project-name` - name of the project in CheckmarxOne to fetch custom queries for
- `-project-id` - ID of the project in CheckmarxOne to fetch custom queries for

Exactly one of `-project-name` or `-project-id` must be provided; if both are empty the tool exits with an error. CheckmarxOne connection parameters (URL, IAM URL, tenant, API key/client credentials) are supplied via the standard `Cx1ClientGo.NewClient` environment/config mechanism rather than command-line flags.

## Behavior

This tool is read-only with respect to query content - it only ever reads query source and never creates or modifies queries. It does, however, cause side effects on the CheckmarxOne tenant: it may create a new project ("CxPSEMEA-Query Migration Project") and trigger a scan if the target project has no completed scan for a given language, and it opens (and later deletes, via a deferred cleanup) one or more WebAudit/QueryEditor audit sessions per language, which can take several minutes for projects with many languages.

## Output format

The tool writes `queries.json` with the following structure:

```
{
  "languages": {
    "<Language>": {
      "<QueryGroup>": {
        "<QueryName>": {
          "Tenant":      { "Source": "...", "Severity": "..." },
          "Application": { "Source": "...", "Severity": "..." },
          "Project":     { "Source": "...", "Severity": "..." }
        }
      }
    }
  }
}
```

For each custom query, the source code and severity are recorded under whichever of `Tenant`, `Application`, or `Project` levels actually exist for that query; levels that don't apply are left as empty `QueryInfo` structs.

## Files

- `file.go` embeds a zip archive (`resourceCodeZip`) of placeholder source files for every supported language (Apex, ASP, Cobol, C++, C#, Dart, Go, Groovy, Java, JavaScript, Kotlin, Lua, Objective-C, Perl, PHP, PL/SQL, SCA, Python, RPG, Ruby, Scala, Swift, VB6, VB.NET, VBScript). `makeZip` extracts just the file matching the requested language, used to create a minimal scan on the target project so an audit/query-editor session can be opened when no completed scan already exists for that language.
