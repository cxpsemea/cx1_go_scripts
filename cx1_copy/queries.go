package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
)

var auditSessions map[*Cx1ClientGo.Cx1Client]*Cx1ClientGo.AuditSession = make(map[*Cx1ClientGo.Cx1Client]*Cx1ClientGo.AuditSession)

var testProjects map[*Cx1ClientGo.Cx1Client]*Cx1ClientGo.Project = make(map[*Cx1ClientGo.Cx1Client]*Cx1ClientGo.Project)

var languageMap map[string]string

func refreshAuditSession(cx1client *Cx1ClientGo.Cx1Client, language string, logger *logrus.Logger) error {
	var auditSession *Cx1ClientGo.AuditSession
	var ok bool

	if auditSession, ok = auditSessions[cx1client]; ok && auditSession != nil && auditSession.HasLanguage(language) {
		if err := cx1client.AuditSessionKeepAlive(auditSession); err != nil {
			_ = cx1client.AuditDeleteSession(auditSession)
			if err = createAuditSession(cx1client, language); err != nil {
				return err
			}
			auditSession = auditSessions[cx1client]
		}
	} else {
		if auditSession != nil {
			deleteAuditSession(cx1client, logger)
		}
		if err := createAuditSession(cx1client, language); err != nil {
			return err
		}
		auditSession = auditSessions[cx1client]
	}

	cx1client.GetAuditSASTQueriesByLevelID(auditSession, "Corp", "Corp")

	return cx1client.AuditSessionKeepAlive(auditSession)
}

func deleteAuditSession(cx1client *Cx1ClientGo.Cx1Client, logger *logrus.Logger) {
	var auditSession *Cx1ClientGo.AuditSession
	var ok bool

	auditSession, ok = auditSessions[cx1client]

	if !ok || auditSession == nil {
		return
	}

	logger.Infof("Deleting audit session with ID: %v", auditSession.ID)

	if err := cx1client.AuditDeleteSession(auditSession); err != nil {
		logger.Errorf("Failed to delete audit session: %s", err)
	}
	auditSession = nil
	auditSessions[cx1client] = nil
}

func createAuditSession(cx1client *Cx1ClientGo.Cx1Client, language string) error {
	var err error
	var testProject *Cx1ClientGo.Project
	var ok bool
	testProject, ok = testProjects[cx1client]
	if !ok || testProject == nil {
		project, err := cx1client.GetOrCreateProjectByName("CxPSEMEA-Query Migration Project")
		if err != nil {
			return err
		}
		testProject = &project
		testProjects[cx1client] = testProject
	}

	filter := Cx1ClientGo.ScanFilter{
		BaseFilter: Cx1ClientGo.BaseFilter{
			Limit:  1,
			Offset: 0,
		},
		Branches: []string{language},
		Statuses: []string{"Completed"},
	}
	scans, err := cx1client.GetLastScansByIDFiltered(testProject.ProjectID, filter)
	if err != nil {
		return err
	}

	var lastscan Cx1ClientGo.Scan

	if len(scans) == 0 {
		zipFile, err := makeZip(&resourceCodeZip, language)
		if err != nil {
			return err
		}
		sastScanConfig := Cx1ClientGo.ScanConfiguration{
			ScanType: "sast",
		}

		uploadURL, err := cx1client.UploadBytes(&zipFile)
		if err != nil {
			return err
		}

		lastscan, err = cx1client.ScanProjectZipByID(testProject.ProjectID, uploadURL, language, []Cx1ClientGo.ScanConfiguration{sastScanConfig}, map[string]string{})
		if err != nil {
			return err
		}

		lastscan, err = cx1client.ScanPollingDetailed(&lastscan)
		if err != nil {
			return err
		}

		if lastscan.Status != "Completed" {
			return fmt.Errorf("scan did not complete successfully")
		}
	} else {
		lastscan = scans[0]
	}

	session, err := cx1client.GetAuditSessionByID("sast", testProject.ProjectID, lastscan.ScanID)
	if err != nil {
		return err
	}

	auditSessions[cx1client] = &session

	return nil
}

func InitializeQueryMigration(cx1client *Cx1ClientGo.Cx1Client) {
	languageMap = make(map[string]string)

	languageMap["code/apexFile.cls"] = "Apex"
	languageMap["code/aspFile.asp"] = "ASP"
	languageMap["code/cobolFile.cbl"] = "Cobol"
	languageMap["code/cplusplusFile.cpp"] = "CPP"
	languageMap["code/csharpFile.cs"] = "CSharp"
	languageMap["code/dartFile.dart"] = "Dart"
	languageMap["code/Dockerfile"] = "IAST"
	languageMap["code/goFile.go"] = "Go"
	languageMap["code/groovyFile.groovy"] = "Groovy"
	languageMap["code/javaFile.java"] = "Java"
	languageMap["code/javaScriptFile.js"] = "JavaScript"
	languageMap["code/kotlinFile.kt"] = "Kotlin"
	languageMap["code/luaFile.lua"] = "Lua"
	languageMap["code/objectivecFile.m"] = "Objc"
	languageMap["code/perlFile.pl"] = "Perl"
	languageMap["code/phpFile.php"] = "PHP"
	languageMap["code/plsqlFile.sql"] = "PLSQL"
	languageMap["code/pom.xml"] = "SCA"
	languageMap["code/pythonFile.py"] = "Python"
	languageMap["code/README.md"] = "None"
	languageMap["code/rpgleFile.rpgle"] = "RPG"
	languageMap["code/rubyFile.rb"] = "Ruby"
	languageMap["code/scalaFile.scala"] = "Scala"
	languageMap["code/swiftFile.swift"] = "Swift"
	languageMap["code/vb6File.bas"] = "VB6"
	languageMap["code/vbnetFile.vb"] = "VbNet"
	languageMap["code/vbscriptFile.vbs"] = "VbScript"
}

func makeZip(zipContents *[]byte, language string) ([]byte, error) {
	contents := new(bytes.Buffer)

	zipReader, err := zip.NewReader(bytes.NewReader(*zipContents), int64(len(*zipContents)))
	if err != nil {
		return contents.Bytes(), err
	}

	zipWriter := zip.NewWriter(contents)

	included_files := 0
	// Read all the files from zip archive
	for _, zipFile := range zipReader.File {
		if languageMap[zipFile.Name] == language {
			w1, err := zipWriter.Create(zipFile.Name)
			if err != nil {
				return contents.Bytes(), err
			}

			f, err := zipFile.Open()
			if err != nil {
				return contents.Bytes(), err
			}
			unzippedFileBytes, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				return contents.Bytes(), err
			}

			if _, err := io.Copy(w1, bytes.NewReader(unzippedFileBytes)); err != nil {
				return contents.Bytes(), err
			}
			included_files++
		}
	}

	zipWriter.Close()
	if included_files == 0 {
		return []byte{}, fmt.Errorf("no files found for language %v", language)
	}
	return contents.Bytes(), nil
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
