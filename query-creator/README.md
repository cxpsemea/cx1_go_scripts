# Query-Creator

This tool can be used to facilitate deploying multiple queries to a Cx1 environment. It is used as followed:
```
query-creator -queries <path-to-queries> -header <headerfile.txt> -apikey %APIKEY%
```

It can be run with other authentication options, full details are available by running with the -h flag:
```
  -apikey string
        CheckmarxOne API Key (if not using client id/secret)
  -client string
        CheckmarxOne Client ID (if not using API Key)
  -cx1 string
        Optional: CheckmarxOne platform URL, if not defined in the test config.yaml
  -header string
        Optional: File containing header to be added to each query, eg: header.txt
  -iam string
        Optional: CheckmarxOne IAM URL, if not defined in the test config.yaml
  -queries string
        Folder containing queries - structure described in README.md (default "queries")
  -secret string
        CheckmarxOne Client Secret (if not using API Key)
  -tenant string
        Optional: CheckmarxOne tenant, if not defined in the test config.yaml
  -token string
        Optional: A valid access_token. If this value is provided, others will be ignored - the client will lose access when the token expires
  -severity string
        Queries will be created with this severity level (Info, Low, Medium, High, Critical) (default "Info")
  -delete
        If set, the queries found in the -queries folder will be deleted instead of created/updated
  -log string
        Log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL (default "INFO")
```

# Queries folder structure

The -queries parameter should contain a path to a folder containing the queries which should be created in Cx1. 
The folder contents should match the structure shown in the query editor/web audit:
```
Language/Group/Query.cs
```

Each query is created (or updated, if a query of the same name/group/language already exists) at the tenant level, using an Audit Session per language. If `-header` is set, its contents are prepended to every query's source. If `-delete` is set, the tool instead removes any tenant-level query overrides matching the queries found in the folder, rather than creating/updating them. 