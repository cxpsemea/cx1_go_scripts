package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	// "fmt"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var auditSession *Cx1ClientGo.AuditSession
var testProject *Cx1ClientGo.Project
var languageMap map[string]string

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	logger.SetOutput(os.Stdout)

	logger.Info("Starting")

	httpClient := &http.Client{}
	var cx1client *Cx1ClientGo.Cx1Client
	var err error

	ProjectName := flag.String("project-name", "", "Name of a project in CheckmarxOne")
	ProjectID := flag.String("project-id", "", "ID of a project in CheckmarxOne")
	cx1client, err = Cx1ClientGo.NewClient(httpClient, logger)

	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	if *ProjectName == "" && *ProjectID == "" {
		logger.Fatalf("Either project-name or project-id parameter is required")
	}

	if *ProjectName != "" {
		proj, err := cx1client.GetProjectByName(*ProjectName)
		if err != nil {
			logger.Fatalf("Failed to get project named %v: %s", *ProjectName, err)
		}
		testProject = &proj
	} else {
		proj, err := cx1client.GetProjectByID(*ProjectID)
		if err != nil {
			logger.Fatalf("Failed to get project with ID %v: %s", *ProjectID, err)
		}
		testProject = &proj
	}

	logger.Infof("This application will retrieve the source code for all custom queries in scope for project %v", testProject.String())
	logger.Infof("This will include Tenant-level queries, Application-level queries (if available), and Project-level queries.")
	logger.Warnf("This process will create multiple WebAudit/QueryEditor sessions and can take multiple minutes.")

	InitializeQueryMigration(cx1client)
	defer deleteAuditSession(cx1client, logger)

	qc, err := cx1client.GetQueries()
	if err != nil {
		logger.Fatalf("Error getting the query collection: %s", err)
	}

	if *ProjectName != "" {
		aq, err := cx1client.GetQueriesByLevelID(Cx1ClientGo.AUDIT_QUERY_PROJECT, testProject.ProjectID)
		if err != nil {
			logger.Fatalf("Failed to get queries for project %v: %s", testProject.String(), err)
		}
		qc.AddQueries(&aq)
	}

	cqc := qc.GetCustomQueryCollection()

	for lid := range cqc.QueryLanguages {
		logger.Infof("Fetching %v-language queries", cqc.QueryLanguages[lid].Name)
		if err := refreshAuditSession(cx1client, cqc.QueryLanguages[lid].Name); err != nil {
			logger.Fatalf("Failed to refresh audit session: %v", err)
		} else {
			aq, err := cx1client.GetAuditQueriesByLevelID(auditSession, cx1client.QueryTypeProject(), testProject.ProjectID)
			if err != nil {
				logger.Errorf("Failed to get query code for Project-level %v queries for project %v: %s", cqc.QueryLanguages[lid].Name, testProject.String(), err)
			} else {
				qc.AddQueries(&aq)
				//logger.Infof("Fetched Project-level %v queries for project %v", cqc.QueryLanguages[lid].Name, testProject.String())
				for _, q := range aq {
					//logger.Infof("Query: %v -> %v -> %v", q.Language, q.Group, q.Name)
					query, err := cx1client.GetAuditQueryByKey(auditSession, q.EditorKey)
					if err != nil {
						logger.Errorf("Failed to get source for query %v -> %v -> %v: %s", q.Language, q.Group, q.Name, err)
					} else {
						qc.AddQuery(query)
					}
				}
			}
		}
	}

	// want to put the output in a bit more readable format
	type QueryInfo struct {
		Source   string `json:"Source"`
		Severity string `json:"Severity"`
	}
	type QueryLevels struct {
		Tenant      QueryInfo `json:"Tenant"`
		Application QueryInfo `json:"Application"`
		Project     QueryInfo `json:"Project"`
	}
	var Output struct {
		Language map[string]map[string]QueryLevels `json:"languages"`
	}

	Output.Language = make(map[string]map[string]QueryLevels)

	cqc = qc.GetCustomQueryCollection()
	logger.Infof("Project %v has the following custom queries in scope", testProject.String())
	for lid := range cqc.QueryLanguages {
		lang := cqc.QueryLanguages[lid].Name
		if _, ok := Output.Language[lang]; !ok {
			Output.Language[lang] = make(map[string]QueryLevels)
		}
		for gid := range cqc.QueryLanguages[lid].QueryGroups {
			for _, q := range cqc.QueryLanguages[lid].QueryGroups[gid].Queries {
				logger.Infof("[%v] %v -> %v -> %v", q.Level, q.Language, q.Group, q.Name)

				var query QueryLevels
				if val, ok := Output.Language[q.Language][q.Group]; ok {
					query = val
				}

				switch q.Level {
				case cx1client.QueryTypeTenant():
					query.Tenant.Source = q.Source
					query.Tenant.Severity = q.Severity
				case cx1client.QueryTypeApplication():
					query.Application.Source = q.Source
					query.Application.Severity = q.Severity
				case cx1client.QueryTypeProject():
					query.Project.Source = q.Source
					query.Project.Severity = q.Severity
				}

				Output.Language[q.Language][q.Group] = query
			}
		}
	}

	data, err := json.Marshal(Output)
	if err != nil {
		logger.Fatalf("Failed to marshal query collection to json: %s", err)
	}

	err = os.WriteFile("queries.json", data, 0777)
	if err != nil {
		logger.Fatalf("Failed to write queries to queries.json: %s", err)
	} else {
		logger.Infof("Wrote custom query collection for project %v to queries.json", testProject.String())
	}

}

func refreshAuditSession(cx1client *Cx1ClientGo.Cx1Client, language string) error {
	if auditSession != nil && auditSession.HasLanguage(language) {
		if err := cx1client.AuditSessionKeepAlive(auditSession); err != nil {
			if err = createAuditSession(cx1client, language); err != nil {
				return err
			}
		}
	} else {
		if err := createAuditSession(cx1client, language); err != nil {
			return err
		}
	}

	cx1client.GetAuditQueriesByLevelID(auditSession, "Corp", "Corp")

	return cx1client.AuditSessionKeepAlive(auditSession)
}

func deleteAuditSession(cx1client *Cx1ClientGo.Cx1Client, logger *logrus.Logger) {
	if auditSession == nil {
		return
	}

	if err := cx1client.AuditDeleteSession(auditSession); err != nil {
		logger.Errorf("Failed to delete audit session: %s", err)
	}
}

func createAuditSession(cx1client *Cx1ClientGo.Cx1Client, language string) error {
	var err error
	if testProject == nil {
		project, err := cx1client.GetOrCreateProjectByName("CxPSEMEA-Query Migration Project")
		if err != nil {
			return err
		}
		testProject = &project
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

	auditSession = &session

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
		}
	}

	zipWriter.Close()
	return contents.Bytes(), nil
}
