package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"

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
