package main

import (
	//"crypto/tls"
	//"flag"

	"crypto/tls"
	"flag"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var oldAPI bool = false
var languageScope = []string{}

func main() {
	os.Exit(mainRunner())
}

func mainRunner() int {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	logger.SetOutput(os.Stdout)

	logger.Info("Starting")

	LogLevel := flag.String("log", "INFO", "Log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL")

	APIKey1 := flag.String("apikey1", "", "CheckmarxOne API Key (if not using client id/secret)")
	ClientID1 := flag.String("client1", "", "CheckmarxOne Client ID (if not using API Key)")
	ClientSecret1 := flag.String("secret1", "", "CheckmarxOne Client Secret (if not using API Key)")
	Cx1URL1 := flag.String("cx1", "", "Optional: CheckmarxOne platform URL, if not defined in the test config.yaml")
	IAMURL1 := flag.String("iam1", "", "Optional: CheckmarxOne IAM URL, if not defined in the test config.yaml")
	Tenant1 := flag.String("tenant1", "", "Optional: CheckmarxOne tenant, if not defined in the test config.yaml")
	Proxy1 := flag.String("proxy1", "", "Optional: Proxy to use when connecting to CheckmarxOne")

	APIKey2 := flag.String("apikey2", "", "CheckmarxOne API Key (if not using client id/secret)")
	ClientID2 := flag.String("client2", "", "CheckmarxOne Client ID (if not using API Key)")
	ClientSecret2 := flag.String("secret2", "", "CheckmarxOne Client Secret (if not using API Key)")
	Cx1URL2 := flag.String("cx2", "", "Optional: CheckmarxOne platform URL, if not defined in the test config.yaml")
	IAMURL2 := flag.String("iam2", "", "Optional: CheckmarxOne IAM URL, if not defined in the test config.yaml")
	Tenant2 := flag.String("tenant2", "", "Optional: CheckmarxOne tenant, if not defined in the test config.yaml")
	Proxy2 := flag.String("proxy2", "", "Optional: Proxy to use when connecting to CheckmarxOne")

	Scope := flag.String("scope", "", "Comma-separated list of items to copy: queries,presets")
	Languages := flag.String("languages", "", "Optional: When migrating queries, only cover the languages in this comma-separated list, eg: javascript,java")

	flag.Parse()

	switch strings.ToUpper(*LogLevel) {
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

	if *Scope == "" {
		logger.Fatalf("Required parameter scope is missing")
	}
	*Scope = strings.ToLower(*Scope)

	if *Languages != "" {
		languageScope = strings.Split(strings.ToLower(*Languages), ",")
	}

	httpClient1 := &http.Client{}
	if *Proxy1 != "" {
		proxyURL, err := url.Parse(*Proxy1)
		if err != nil {
			logger.Fatalf("Failed to parse proxy url %v: %s", *Proxy1, err)
		}
		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		httpClient1.Transport = transport
		logger.Infof("Running with proxy1: %v", *Proxy1)
	}
	httpClient2 := &http.Client{}
	if *Proxy2 != "" {
		proxyURL, err := url.Parse(*Proxy2)
		if err != nil {
			logger.Fatalf("Failed to parse proxy url %v: %s", *Proxy2, err)
		}
		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		httpClient2.Transport = transport
		logger.Infof("Running with proxy2: %v", *Proxy2)
	}

	var cx1client1, cx1client2 *Cx1ClientGo.Cx1Client
	var err error

	if *APIKey1 != "" {
		cx1client1, err = Cx1ClientGo.NewAPIKeyClient(httpClient1, *Cx1URL1, *IAMURL1, *Tenant1, *APIKey1, logger)
	} else {
		cx1client1, err = Cx1ClientGo.NewOAuthClient(httpClient1, *Cx1URL1, *IAMURL1, *Tenant1, *ClientID1, *ClientSecret1, logger)
	}
	if err != nil {
		logger.Fatalf("Failed to create client #1 for %v: %s", *Tenant1, err)
	}
	logger.Infof("Connected client #1 with %v", cx1client1.String())

	if *APIKey2 != "" {
		cx1client2, err = Cx1ClientGo.NewAPIKeyClient(httpClient2, *Cx1URL2, *IAMURL2, *Tenant2, *APIKey2, logger)
	} else {
		cx1client2, err = Cx1ClientGo.NewOAuthClient(httpClient2, *Cx1URL2, *IAMURL2, *Tenant2, *ClientID2, *ClientSecret2, logger)
	}
	if err != nil {
		logger.Fatalf("Failed to create client #2 for %v: %s", *Tenant2, err)
	}
	logger.Infof("Connected client #2 with %v", cx1client2.String())

	if strings.Contains(*Scope, "queries") {
		CopyQueries(cx1client1, cx1client2, logger)
	}
	if strings.Contains(*Scope, "presets") {

	}

	return 0
}

func CopyQueries(cx1client1, cx1client2 *Cx1ClientGo.Cx1Client, logger *logrus.Logger) {
	flag1, _ := cx1client1.CheckFlag("CVSS_V3_ENABLED")
	flag2, _ := cx1client2.CheckFlag("CVSS_V3_ENABLED")
	if flag1 != flag2 {
		logger.Errorf("CVSS_V3_ENABLED feature flag is different between environments - cannot migrate queries")
		return
	}

	InitializeQueryMigration(cx1client1)

	srcQColl, err := cx1client1.GetSASTQueryCollection()
	if err != nil {
		logger.Fatalf("Failed to fetch queries from %v: %s", cx1client1.String(), err)
	}

	dstQColl, err := cx1client2.GetSASTQueryCollection()
	if err != nil {
		logger.Fatalf("Failed to fetch queries from %v: %s", cx1client2.String(), err)
	}

	oldAPI, _ = cx1client2.CheckFlag("QUERY_EDITOR_SAST_BACKWARD_API_ENABLED")

	for _, lang := range srcQColl.QueryLanguages {
		if len(languageScope) != 0 && !slices.Contains(languageScope, strings.ToLower(lang.Name)) {
			logger.Infof("Language %v is not in-scope", lang.Name)
			continue
		} else {
			logger.Infof("Checking language %v for custom queries to migrate", lang.Name)

			err = getLangCustomQueries(cx1client1, lang.Name, &srcQColl, logger)
			if err != nil {
				continue
			}
			err = getLangCustomQueries(cx1client2, lang.Name, &dstQColl, logger)
			if err != nil {
				continue
			}

			for _, group := range lang.QueryGroups {
				if err = cx1client1.AuditSessionKeepAlive(auditSessions[cx1client1]); err != nil {
					logger.Errorf("Failed to refresh audit session on %v: %v", cx1client1.String(), err)
					continue
				}
				if err = cx1client2.AuditSessionKeepAlive(auditSessions[cx1client2]); err != nil {
					logger.Errorf("Failed to refresh audit session on %v: %v", cx1client2.String(), err)
					continue
				}

				for _, query := range group.Queries {
					if query.Custom {
						logger.Infof("Custom query found on %v: %v", cx1client1.String(), query.StringDetailed())

						query1, err := cx1client1.GetAuditSASTQueryByKey(auditSessions[cx1client1], query.EditorKey)
						if err != nil {
							logger.Errorf("Failed to get query source from %v: %v", cx1client1.String(), err)
							continue
						}

						query2 := dstQColl.GetQueryByLevelAndName(cx1client2.QueryTypeTenant(), cx1client2.QueryTypeTenant(), query.Language, query.Group, query.Name)
						if query2 == nil {
							logger.Infof("Query %v does not yet exist on %v", query.StringDetailed(), cx1client2.String())
							query2 = dstQColl.GetQueryByLevelAndName(cx1client2.QueryTypeProduct(), cx1client2.QueryTypeProduct(), query.Language, query.Group, query.Name)
							// create tenant-level query override & set source
							new_query, err := createOverride(cx1client2, query1, query2, logger)
							if err != nil {
								logger.Errorf("Failed to create override for query %v in %v: %v", query1.StringDetailed(), cx1client2.String(), err)
							} else {
								logger.Infof("Created override for query %v in %v", new_query.StringDetailed(), cx1client2.String())
							}
						} else {
							query2, err := cx1client2.GetAuditSASTQueryByKey(auditSessions[cx1client2], query2.EditorKey)
							if err != nil {
								logger.Errorf("Failed to get query source from %v: %v", cx1client2.String(), err)
								continue
							}

							if query1.Source != query2.Source || query1.Severity != query2.Severity {
								logger.Infof("Query source or severity for %v is different between environments and will be updated", query.StringDetailed())
								if oldAPI {
									q2 := query2.ToAuditQuery_v310()
									q2.Source = query1.Source
									q2.Severity = query1.Severity
									q2.Cwe = query1.CweID
									q2.CxDescriptionId = query1.QueryDescriptionId
									err = cx1client2.UpdateQuery_v310(q2)
									if err != nil {
										logger.Errorf("Failed to update query %v in %v", q2.ToQuery().StringDetailed(), cx1client2.String())
									}
								} else {
									if query1.Source != query2.Source {
										_, _, err = cx1client2.UpdateSASTQuerySourceByKey(auditSessions[cx1client2], query2.EditorKey, query1.Source)
										if err != nil {
											logger.Errorf("Failed to update query source %v in %v: %v", query2.StringDetailed(), cx1client2.String(), err)
											continue
										}
									}

									if query1.Severity != query2.Severity {
										_, err = cx1client2.UpdateSASTQueryMetadataByKey(auditSessions[cx1client2], query2.EditorKey, query1.GetMetadata())
										if err != nil {
											logger.Errorf("Failed to update query metadata %v in %v: %v", query2.StringDetailed(), cx1client2.String(), err)
											continue
										}
									}

									logger.Infof("Updated query %v in %v", query2.StringDetailed(), cx1client2.String())
								}
							} else {
								logger.Infof("Query source for %v is the same between environments", query.StringDetailed())
							}
						}
					}
				}
			}
		}
	}

	if auditSessions[cx1client1] != nil {
		deleteAuditSession(cx1client1, logger)
	}

	if auditSessions[cx1client2] != nil {
		deleteAuditSession(cx1client2, logger)
	}
}

func createOverride(cx1client2 *Cx1ClientGo.Cx1Client, query Cx1ClientGo.SASTQuery, query2 *Cx1ClientGo.SASTQuery, logger *logrus.Logger) (*Cx1ClientGo.SASTQuery, error) {
	if query2 == nil {
		logger.Infof("Creating new query %v", query.StringDetailed())
		new_query, _, err := cx1client2.CreateNewSASTQuery(auditSessions[cx1client2], query)
		if err != nil {
			return &new_query, err
		}

		if new_query.Severity != query.Severity {
			new_query, err = cx1client2.UpdateSASTQueryMetadata(auditSessions[cx1client2], &new_query, query.GetMetadata())
			if err != nil {
				return &new_query, err
			}
		}
		return &new_query, nil
	} else {
		logger.Infof("Creating new override %v", query.StringDetailed())
		if oldAPI {
			new_query := query.ToAuditQuery_v310().CreateTenantOverride()
			err := cx1client2.UpdateQuery_v310(new_query)
			newq := new_query.ToQuery()
			return &newq, err
		} else {
			new_query, err := cx1client2.CreateSASTQueryOverride(auditSessions[cx1client2], cx1client2.QueryTypeTenant(), query2)
			if err != nil {
				return nil, err
			}
			if new_query.Source != query.Source {
				new_query, _, err = cx1client2.UpdateSASTQuerySource(auditSessions[cx1client2], &new_query, query.Source)
				if err != nil {
					return &new_query, err
				}
			}
			if new_query.Severity != query.Severity {
				new_query, err = cx1client2.UpdateSASTQueryMetadata(auditSessions[cx1client2], &new_query, query.GetMetadata())
				if err != nil {
					return &new_query, err
				}
			}
			return &new_query, nil
		}
	}
}

func getLangCustomQueries(cx1client *Cx1ClientGo.Cx1Client, language string, queryCollection *Cx1ClientGo.SASTQueryCollection, logger *logrus.Logger) error {
	if err := refreshAuditSession(cx1client, language, logger); err != nil {
		logger.Errorf("Failed to refresh audit session: %v", err)
		return err
	}

	testProject := testProjects[cx1client]

	aq, err := cx1client.GetAuditSASTQueriesByLevelID(auditSessions[cx1client], cx1client.QueryTypeProject(), testProject.ProjectID)
	if err != nil {
		logger.Errorf("Failed to get audit queries from %v for Project-level %v queries for project %v: %s", cx1client.String(), language, testProject.String(), err)
	} else {
		queryCollection.AddCollection(&aq)
	}
	return nil
}
