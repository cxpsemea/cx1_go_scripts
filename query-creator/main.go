package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var logger *logrus.Logger
var Header string

var defaultQuery = Cx1ClientGo.SASTQuery{}

func main() {
	logger = logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	logger.SetOutput(os.Stdout)

	loglevel := flag.String("log", "INFO", "Log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL")
	queriesFolder := flag.String("queries", "queries", "Folder containing queries - structure described in README.md")
	headerFile := flag.String("header", "", "Optional: File containing header to be added to each query, eg: header.txt")
	//scanId := flag.String("scan-id", "", "Optional: Create queries in the project owning this scan ID (tenant-level by default)")
	//application := flag.Bool("application", false, "Optional: Create queries in the application owning the project (requires scan-id to be set)")
	severity := flag.String("severity", "Info", "Queries will be created with this severity level (Info, Low, Medium, High, Critical)")
	deleteQueries := flag.Bool("delete", false, "If set, the queries will be deleted instead of created/updated")

	logger.Info("Starting")
	client := &http.Client{}

	/*if true {
		proxyURL, _ := url.Parse("http://127.0.0.1:8080")
		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client.Transport = transport
	}*/

	cx1client, err := Cx1ClientGo.NewClient(client, logger)
	if err != nil {
		logger.Fatalf("Failed to create client: %v", err)
	} else {
		logger.Infof("Connected with %v", cx1client.String())
	}

	switch strings.ToUpper(*loglevel) {
	case "TRACE":
		logger.Info("Setting log level to TRACE")
		logger.SetLevel(logrus.TraceLevel)
	case "DEBUG":
		logger.Info("Setting log level to DEBUG")
		logger.SetLevel(logrus.DebugLevel)
	case "INFO":
		logger.Info("Setting log level to INFO")
		logger.SetLevel(logrus.InfoLevel)
	case "WARNING":
		logger.Info("Setting log level to WARNING")
		logger.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logger.Info("Setting log level to ERROR")
		logger.SetLevel(logrus.ErrorLevel)
	case "FATAL":
		logger.Info("Setting log level to FATAL")
		logger.SetLevel(logrus.FatalLevel)
	default:
		logger.Info("Log level set to default: INFO")
	}

	//var scan *Cx1ClientGo.Scan
	//var project *Cx1ClientGo.Project

	/*if *scanId != "" {
		tscan, err := cx1client.GetScanByID(*scanId)
		if err != nil {
			logger.Fatalf("Failed to get scan: %v", err)
		}
		scan = &tscan
		tproject, err := cx1client.GetProjectByID(scan.ProjectID)
		if err != nil {
			logger.Fatalf("Failed to get project: %v", err)
		}
		project = &tproject
	}*/

	logger.Infof("Will set the severity of new queries to: %v", *severity)
	defaultQuery.Severity = strings.ToLower(*severity)

	/*if *application {
		if *scanId == "" {
			logger.Fatalf("When -application is set, -scan-id must also be set")
		}
		if len(*project.Applications) == 0 {
			logger.Fatalf("Project %v does not have any applications", project.Name)
		}
		logger.Infof("Will create queries on the application level for scanID")
		defaultQuery.Level = cx1client.QueryTypeApplication()
		defaultQuery.LevelID = (*project.Applications)[0]
	} else {
		if *scanId != "" {
			logger.Infof("Will create queries on the project level for scanID %v", *scanId)
			defaultQuery.Level = cx1client.QueryTypeProject()
			defaultQuery.LevelID = scan.ProjectID
		} else {
			logger.Infof("Will create queries on the tenant level")
			defaultQuery.Level = cx1client.QueryTypeTenant()
			defaultQuery.LevelID = cx1client.QueryTypeTenant()
		}
	}*/
	defaultQuery.Level = cx1client.QueryTypeTenant()
	defaultQuery.LevelID = cx1client.QueryTypeTenant()

	if *headerFile != "" {
		logger.Infof("Will prepend each query with contents of %v", *headerFile)
		data, err := os.ReadFile(*headerFile)
		if err != nil {
			logger.Fatalf("Failed to read header file: %v", err)
		}
		Header = string(data)
	}

	logger.Infof("Will install queries from %v", *queriesFolder)
	queries, err := LoadQueriesFromFolder(*queriesFolder)
	if err != nil {
		logger.Fatalf("Failed to load queries: %v", err)
	}

	queryLanguages := []string{}
	for _, lang := range queries.QueryLanguages {
		queryLanguages = append(queryLanguages, lang.Name)
	}
	logger.Infof("Loaded queries for the following languages: %v", strings.Join(queryLanguages, ", "))

	if *deleteQueries {
		err = CreateQueriesFromCollection(cx1client, queries, nil, true)
	} else {
		err = CreateQueriesFromCollection(cx1client, queries, nil, false)
	}
	if err != nil {
		logger.Fatalf("Failed to create queries: %v", err)
	}

	logger.Info("Done")
}

func LoadQueriesFromFolder(folder string) (collection Cx1ClientGo.SASTQueryCollection, err error) {
	// Subdirectories in folder should follow this structure:
	// <Language>/<QueryGroup>/<QueryFiles>.cs
	err = filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // Propagate errors from walking
		}
		if info.IsDir() {
			return nil // Skip directories
		}

		// Check if it's a .cs file
		if !strings.HasSuffix(info.Name(), ".cs") {
			return nil // Skip non-.cs files
		}

		// Call LoadQueryFromFile for each .cs file
		query, err := LoadQueryFromFile(folder, path)
		if err != nil {
			logger.Errorf("Failed to load query from file %s: %v", path, err)
			return nil // Continue processing other files even if one fails
		}

		collection.AddQuery(query)
		logger.Infof("Will create/update: %v [exec: %v]", query.StringDetailed(), query.IsExecutable)
		return nil
	})

	if err != nil {
		return collection, fmt.Errorf("error walking through queries folder %v: %v", folder, err)
	}

	return collection, nil
}

func LoadQueryFromFile(rootFolder, path string) (query Cx1ClientGo.SASTQuery, err error) {
	relPath, err := filepath.Rel(rootFolder, path)
	if err != nil {
		logger.Errorf("failed to get relative path for %v from base %v: %v", rootFolder, path, err)
		return query, err
	}

	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) != 3 {
		return query, fmt.Errorf("invalid query file path: %v => %v", path, parts)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return query, err
	}

	query.Language = parts[0]
	query.Group = parts[1]
	query.Name = strings.TrimSuffix(parts[2], filepath.Ext(parts[2]))
	query.IsExecutable = query.Group != "General"
	query.Source = Header + "\n" + string(data)
	query.Severity = defaultQuery.Severity
	query.Level = defaultQuery.Level
	query.LevelID = defaultQuery.LevelID

	return query, nil
}

func CreateQueriesFromCollection(cx1client *Cx1ClientGo.Cx1Client, collection Cx1ClientGo.SASTQueryCollection, scan *Cx1ClientGo.Scan, delete bool) error {
	logger.Infof("Fetching existing queries")
	qc, err := cx1client.GetSASTQueryCollection()
	if err != nil {
		return err
	}

	var session Cx1ClientGo.AuditSession
	defer func() {
		if session.ID != "" {
			err = cx1client.DeleteAuditSession(&session)
			if err != nil {
				logger.Errorf("Failed to close audit session: %v", err)
			} else {
				logger.Infof("Closed audit session %v", session.String())
			}
			session = Cx1ClientGo.AuditSession{}
		}
	}()

	if scan != nil {
		logger.Infof("Creating audit session for scan %v", scan.String())
		session, err = cx1client.GetAuditSessionByID("sast", scan.ProjectID, scan.ScanID)
		if err != nil {
			return fmt.Errorf("failed to get audit session: %v", err)
		} else {
			logger.Infof("Created audit session %v for scan %v covering languages: %v", session.String(), scan.String(), strings.Join(session.Languages, ", "))
			err = qc.UpdateFromSession(cx1client, &session)
			if err != nil {
				return fmt.Errorf("failed to update queries from session: %v", err)
			}
		}
	}

	for _, lang := range collection.QueryLanguages {
		if !slices.Contains(session.Languages, lang.Name) {
			if scan == nil {
				if session.ID != "" {
					err = cx1client.DeleteAuditSession(&session)
					if err != nil {
						logger.Errorf("Failed to close audit session: %v", err)
					} else {
						logger.Infof("Closed audit session %v", session.String())
					}
					session = Cx1ClientGo.AuditSession{}
				}
				logger.Infof("Current session does not cover language %v, creating a new session", lang.Name)
				session, err = cx1client.GetAuditSession("sast", strings.ToLower(lang.Name))
				if err != nil {
					return fmt.Errorf("failed to create audit session for language %v: %v", lang.Name, err)
				} else {
					logger.Infof("Created new audit session %v for language %v", session.String(), lang.Name)
					err = qc.UpdateFromSession(cx1client, &session)
					if err != nil {
						return fmt.Errorf("failed to update queries from session: %v", err)
					} else {
						logger.Infof("Retrieved queries from session for language %v", lang.Name)
					}
				}
			} else {
				logger.Warnf("Skipping queries for language %v as it is not included in the current audit session for scan %v", lang.Name, scan.String())
			}
		}

		logger.Infof("Processing queries for language %v", lang.Name)
		CreateQueriesFromLanguage(cx1client, &session, qc, lang.QueryGroups, delete)
	}

	return nil
}

func CreateQueriesFromLanguage(cx1client *Cx1ClientGo.Cx1Client, session *Cx1ClientGo.AuditSession, qc Cx1ClientGo.SASTQueryCollection, groups []Cx1ClientGo.SASTQueryGroup, delete bool) {
	if delete {
		for _, qg := range groups {
			if qg.Name != "General" {
				createQueriesFromGroup(cx1client, session, qg, qc, delete)
			}
		}
		for _, qg := range groups {
			if qg.Name == "General" {
				createQueriesFromGroup(cx1client, session, qg, qc, delete)
			}
		}
	} else {
		for _, qg := range groups {
			if qg.Name == "General" {
				createQueriesFromGroup(cx1client, session, qg, qc, delete)
			}
		}
		for _, qg := range groups {
			if qg.Name != "General" {
				createQueriesFromGroup(cx1client, session, qg, qc, delete)
			}
		}
	}
}

func createQueriesFromGroup(cx1client *Cx1ClientGo.Cx1Client, session *Cx1ClientGo.AuditSession, qg Cx1ClientGo.SASTQueryGroup, qc Cx1ClientGo.SASTQueryCollection, delete bool) {
	if delete {
		for _, query := range qg.Queries {
			existingQuery := qc.GetQueryByLevelAndName(query.Level, query.LevelID, query.Language, query.Group, query.Name)
			err := cx1client.AuditSessionKeepAlive(session)
			if err != nil {
				logger.Errorf("Failed to keep audit session alive: %v", err)
				return
			}
			if existingQuery != nil {
				existingQuery, err := cx1client.GetAuditSASTQueryByKey(session, existingQuery.EditorKey)
				if err != nil {
					logger.Errorf("Failed to get existing query %v: %v", existingQuery.StringDetailed(), err)
				} else {
					err = cx1client.DeleteQueryOverrideByKey(session, existingQuery.EditorKey)
					if err != nil {
						logger.Errorf("Failed to delete query %v: %v", existingQuery.StringDetailed(), err)
					} else {
						logger.Infof("Deleted existing query: %v", existingQuery.StringDetailed())
					}
				}
			} else {
				logger.Infof("Query %v does not exist", query.StringDetailed())
			}
		}
	} else {
		// Create the queries first - so that if an earlier query calls a later query, it won't fail since the later query doesn't exist yet
		for _, query := range qg.Queries {
			err := cx1client.AuditSessionKeepAlive(session)
			if err != nil {
				logger.Errorf("Failed to keep audit session alive: %v", err)
				return
			}
			if qc.GetQueryByLevelAndName(query.Level, query.LevelID, query.Language, query.Group, query.Name) == nil {
				baseQuery := qc.GetClosestQueryByLevelAndName(cx1client.QueryTypeTenant(), cx1client.QueryTypeTenant(), query.Language, query.Group, query.Name)
				if baseQuery != nil {
					logger.Infof("Closest existing query found is %v", baseQuery.StringDetailed())
				} else {
					logger.Infof("No existing query found for %v - will create new", query.StringDetailed())
					var newCorpQuery Cx1ClientGo.SASTQuery
					newCorpQuery = query
					newCorpQuery.Source = "result = All.NewCxList();"

					newCorpQuery, fails, err := cx1client.CreateNewSASTQuery(session, newCorpQuery)
					if err != nil {
						logger.Errorf("Failed to create query %v: %v", query.StringDetailed(), err)
						if len(fails) > 0 {
							for _, f := range fails {
								logger.Errorf("  - %v", f)
							}
						}
						continue
					}
					qc.AddQuery(newCorpQuery)
					logger.Infof("Created new tenant query: %v", newCorpQuery.StringDetailed())
					baseQuery = &newCorpQuery
				}

				if baseQuery.Level != query.Level {
					newOverride, err := cx1client.CreateSASTQueryOverride(session, query.Level, baseQuery)
					if err != nil {
						logger.Errorf("Failed to create override for %v: %v", query.StringDetailed(), err)
						continue
					}
					qc.AddQuery(newOverride)
					logger.Infof("Created new override: %v", newOverride.StringDetailed())
				}
			}
		}

		for _, query := range qg.Queries {
			existingQuery := qc.GetQueryByLevelAndName(query.Level, query.LevelID, query.Language, query.Group, query.Name)
			err := cx1client.AuditSessionKeepAlive(session)
			if err != nil {
				logger.Errorf("Failed to keep audit session alive: %v", err)
				return
			}
			if existingQuery != nil {
				auditQuery, err := cx1client.GetAuditSASTQueryByKey(session, existingQuery.EditorKey)
				if err != nil {
					logger.Errorf("Failed to get existing query %v: %v", existingQuery.StringDetailed(), err)
					continue
				}
				existingQuery.MergeQuery(auditQuery)
				logger.Infof("Existing query found: %v", existingQuery.StringDetailed())
			} else {
				logger.Errorf("No existing query found for %v", query.StringDetailed())
				continue
			}

			if existingQuery.Source != query.Source {
				logger.Debugf("Updating source from:\n%v\n\nTo:\n%v\n\n", existingQuery.Source, query.Source)
				updatedQuery, fails, err := cx1client.UpdateSASTQuerySource(session, *existingQuery, query.Source)
				if err != nil {
					logger.Errorf("Failed to update query source %v: %v", query.StringDetailed(), err)
					if len(fails) > 0 {
						for _, f := range fails {
							logger.Errorf("  - %v", f)
						}
					}
					continue
				}
				logger.Infof("Updated source for query %v", query.StringDetailed())
				existingQuery = &updatedQuery
			}

			if !strings.EqualFold(existingQuery.Severity, query.Severity) {
				newMeta := existingQuery.GetMetadata()
				newMeta.Severity = query.Severity
				newMeta.IsExecutable = query.IsExecutable
				updatedQuery, err := cx1client.UpdateSASTQueryMetadata(session, *existingQuery, newMeta)
				if err != nil {
					logger.Errorf("Failed to update query metadata %v: %v", query.StringDetailed(), err)
					continue
				}
				existingQuery = &updatedQuery
				logger.Infof("Updated severity for query %v to %v", query.StringDetailed(), query.Severity)
			}

			logger.Infof("Final query state: %v", existingQuery.StringDetailed())
		}
	}
}
