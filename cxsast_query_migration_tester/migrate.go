package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/cxpsemea/CxSASTClientGo"
	"github.com/sirupsen/logrus"
)

var auditSession *Cx1ClientGo.AuditSession
var languageMap map[string]string
var QueryMapping map[uint64]uint64
var cx1qc Cx1ClientGo.QueryCollection

/*
dest can be nil if this is a brand new query with no base to override.
in that case, make a tenant-level no-result query
*/
func MigrateCorpQuery(cx1client *Cx1ClientGo.Cx1Client, sastqc *CxSASTClientGo.QueryCollection, source *CxSASTClientGo.Query, logger *logrus.Logger) (*Cx1ClientGo.Query, error) {
	if err := refreshAuditSession(cx1client, sastqc, source.Language); err != nil {
		return nil, err
	}

	if source.OwningGroup.PackageType == CxSASTClientGo.CORP_QUERY { // we are migrating a corp query to a tenant query
		if source.BaseQueryID == source.QueryID { // it's a brand-new corp query
			return createEmptyCorpQuery(cx1client, source, logger)
		} else {
			return createCorpOverride(cx1client, sastqc, source, logger)
		}
	} else {
		return createNewCorpQuery(cx1client, source, logger)
	}
}

func MigrateTeamQuery(cx1client *Cx1ClientGo.Cx1Client, sastqc *CxSASTClientGo.QueryCollection, source *CxSASTClientGo.Query, logger *logrus.Logger) (*Cx1ClientGo.Query, error) {
	if err := refreshAuditSession(cx1client, sastqc, source.Language); err != nil {
		return nil, err
	}

	if cx1q, err := createTeamOverride(cx1client, sastqc, source, logger); err != nil {
		return nil, err
	} else {
		return cx1q, nil
	}
}

func MigrateProjectQuery(cx1client *Cx1ClientGo.Cx1Client, sastqc *CxSASTClientGo.QueryCollection, source *CxSASTClientGo.Query, logger *logrus.Logger) (*Cx1ClientGo.Query, error) {
	if err := refreshAuditSession(cx1client, sastqc, source.Language); err != nil {
		return nil, err
	}

	if cx1q, err := createProjectOverride(cx1client, sastqc, source, logger); err != nil {
		return nil, err
	} else {
		return cx1q, nil
	}
}

/////

func getCx1RootQuery(cx1client *Cx1ClientGo.Cx1Client, sastqc *CxSASTClientGo.QueryCollection, source *CxSASTClientGo.Query) (*Cx1ClientGo.Query, error) {
	sastRootId := sastqc.GetRootQueryID(source.QueryID)
	if sastRootId == 0 {
		return nil, fmt.Errorf("unable to find SAST root query for %v", source.StringDetailed())
	}

	return getCx1Query(cx1client, sastRootId, source)
}

func getCx1Query(cx1client *Cx1ClientGo.Cx1Client, sastRootId uint64, source *CxSASTClientGo.Query) (*Cx1ClientGo.Query, error) {
	baseQuery := cx1client.GetCx1QueryFromSAST(sastRootId, source.Language, source.Group, source.Name, &QueryMapping, &cx1qc)
	if baseQuery == nil {
		return nil, fmt.Errorf("unable to find Cx1 query for %v", source.StringDetailed())
	}

	return baseQuery, nil
}

func checkQueryEquivalent(cx1client *Cx1ClientGo.Cx1Client, source *CxSASTClientGo.Query, cx1QueryBase *Cx1ClientGo.Query) (*Cx1ClientGo.Query, bool) {
	cx1q := cx1qc.GetQueryByLevelAndID(Cx1ClientGo.AUDIT_QUERY_TENANT, Cx1ClientGo.AUDIT_QUERY_TENANT, cx1QueryBase.QueryID)
	if cx1q != nil {
		auditQuery, err := cx1client.GetAuditQueryByKey(auditSession, cx1q.EditorKey)
		if err != nil {
			logger.Warningf("Destination query for %v exists as %v, but an error occurred while fetching the source: %s", source.StringDetailed(), cx1q.StringDetailed(), err)
		} else {
			cx1q.MergeQuery(auditQuery)

			if cx1q.Source == source.Source {
				logger.Infof("No need to migrate %v: same source exists in %v", source.StringDetailed(), cx1q.StringDetailed())
				return cx1q, true
			} else {
				logger.Infof("Destination query for %v exists as %v but source is different", source.StringDetailed(), cx1q.StringDetailed())
				logger.Infof("SAST:\n%v\n\nCx1:\n%v\n\n", source.Source, cx1q.Source)
			}
		}
	} else {
		logger.Infof("Destination query for %v doesn't exist yet", source.StringDetailed())
	}
	return nil, false
}

/////

func createNewCorpQuery(cx1client *Cx1ClientGo.Cx1Client, source *CxSASTClientGo.Query, logger *logrus.Logger) (*Cx1ClientGo.Query, error) {
	NewQuery := Cx1ClientGo.Query{
		Source:             source.Source,
		Name:               source.Name,
		Group:              source.Group,
		Language:           source.Language,
		Severity:           Cx1ClientGo.GetSeverity(uint(source.Severity)),
		CweID:              int64(source.CWE),
		IsExecutable:       source.IsExecutable,
		QueryDescriptionId: int64(source.DescriptionID),
		Level:              Cx1ClientGo.AUDIT_QUERY_TENANT,
	}

	cx1q := cx1qc.GetQueryByLevelAndName(Cx1ClientGo.AUDIT_QUERY_TENANT, Cx1ClientGo.AUDIT_QUERY_TENANT, source.Language, source.Group, source.Name)
	if cx1q != nil {
		cx1q, equiv := checkQueryEquivalent(cx1client, source, cx1q)
		if equiv {
			logger.Infof("Creating new tenant query not necessary: %v already exists as %v", source.StringDetailed(), cx1q.StringDetailed())
			return cx1q, nil
		}
	}

	logger.Infof("Creating new tenant query: %v (from source %v)", NewQuery.StringDetailed(), source.StringDetailed())
	newCorpQuery, err := cx1client.CreateNewQuery(auditSession, NewQuery)

	return &newCorpQuery, err
}

func createEmptyCorpQuery(cx1client *Cx1ClientGo.Cx1Client, source *CxSASTClientGo.Query, logger *logrus.Logger) (*Cx1ClientGo.Query, error) {
	NewQuery := Cx1ClientGo.Query{
		Source:             "//empty",
		Name:               source.Name,
		Group:              source.Group,
		Language:           source.Language,
		Severity:           Cx1ClientGo.GetSeverity(uint(source.Severity)),
		CweID:              int64(source.CWE),
		IsExecutable:       source.IsExecutable,
		QueryDescriptionId: int64(source.DescriptionID),
		Level:              Cx1ClientGo.AUDIT_QUERY_TENANT,
	}

	cx1q := cx1qc.GetQueryByLevelAndName(Cx1ClientGo.AUDIT_QUERY_TENANT, Cx1ClientGo.AUDIT_QUERY_TENANT, source.Language, source.Group, source.Name)
	if cx1q != nil {
		NewSource := *source
		NewSource.Source = "//empty"
		cx1q, equiv := checkQueryEquivalent(cx1client, &NewSource, cx1q)
		if equiv {
			logger.Infof("Creating new empty tenant query not necessary: %v already exists as %v", source.StringDetailed(), cx1q.StringDetailed())
			return cx1q, nil
		}
	}

	logger.Infof("Creating new empty tenant query: %v (from source %v)", NewQuery.StringDetailed(), source.StringDetailed())

	newCorpQuery, err := cx1client.CreateNewQuery(auditSession, NewQuery)

	return &newCorpQuery, err
}

func createOverride(cx1client *Cx1ClientGo.Cx1Client, sastqc *CxSASTClientGo.QueryCollection, source *CxSASTClientGo.Query, level string, logger *logrus.Logger) (*Cx1ClientGo.Query, error) {
	var NewQuery Cx1ClientGo.Query
	var err error

	baseQuery, err := getCx1RootQuery(cx1client, sastqc, source)
	if err != nil {
		return nil, err
	}
	if cx1q, equiv := checkQueryEquivalent(cx1client, source, baseQuery); equiv {
		logger.Infof("Creating new %v query override not necessary: %v already exists as %v", level, source.StringDetailed(), cx1q.StringDetailed())
		return cx1q, nil
	}

	logger.Infof("Creating new %v query override of: %v (from source %v)", level, baseQuery.StringDetailed(), source.StringDetailed())
	NewQuery, err = cx1client.CreateQueryOverride(auditSession, level, baseQuery)
	if err != nil {
		return nil, err
	}

	appQuery, err := cx1client.UpdateQuerySource(auditSession, &NewQuery, source.Source)
	if err != nil {
		return nil, err
	}

	metadata := appQuery.GetMetadata()
	metadata.Severity = Cx1ClientGo.GetSeverity(uint(source.Severity))
	appQuery, err = cx1client.UpdateQueryMetadata(auditSession, &appQuery, metadata)
	if err != nil {
		return nil, err
	}

	return &appQuery, err
}

func createTeamOverride(cx1client *Cx1ClientGo.Cx1Client, sastqc *CxSASTClientGo.QueryCollection, source *CxSASTClientGo.Query, logger *logrus.Logger) (*Cx1ClientGo.Query, error) {
	return createOverride(cx1client, sastqc, source, Cx1ClientGo.AUDIT_QUERY_APPLICATION, logger)
}

func createProjectOverride(cx1client *Cx1ClientGo.Cx1Client, sastqc *CxSASTClientGo.QueryCollection, source *CxSASTClientGo.Query, logger *logrus.Logger) (*Cx1ClientGo.Query, error) {
	return createOverride(cx1client, sastqc, source, Cx1ClientGo.AUDIT_QUERY_PROJECT, logger)
}

func createCorpOverride(cx1client *Cx1ClientGo.Cx1Client, sastqc *CxSASTClientGo.QueryCollection, source *CxSASTClientGo.Query, logger *logrus.Logger) (*Cx1ClientGo.Query, error) {
	return createOverride(cx1client, sastqc, source, Cx1ClientGo.AUDIT_QUERY_TENANT, logger)
}

func refreshAuditSession(cx1client *Cx1ClientGo.Cx1Client, sastqc *CxSASTClientGo.QueryCollection, language string) error {
	if auditSession != nil {
		if auditSession.HasLanguage(language) {
			return cx1client.AuditSessionKeepAlive(auditSession)
		} else {
			logger.Debugf("Audit session %v is not suitable for this query, deleting", auditSession.ID)
			_ = cx1client.AuditDeleteSession(auditSession)
		}
	}

	if err := createAuditSession(cx1client, language); err != nil {
		return err
	}

	return cx1client.AuditSessionKeepAlive(auditSession)
}

func createAuditSession(cx1client *Cx1ClientGo.Cx1Client, language string) error {
	testProject, _, err := cx1client.GetOrCreateProjectInApplicationByName("CxPSEMEA-Query Migration Project", "CxPSEMEA-Query Migration Application")
	if err != nil {
		return err
	}

	filter := Cx1ClientGo.ScanFilter{
		Limit:    1,
		Offset:   0,
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
	logger.Debugf("Created audit session with languages: %v", auditSession.Languages)

	aq, err := cx1client.GetAuditQueriesByLevelID(&session, Cx1ClientGo.AUDIT_QUERY_PROJECT, testProject.ProjectID)
	if err != nil {
		return fmt.Errorf("error getting queries: %s", err)
	}

	cx1qc.AddQueries(&aq)

	return nil
}

func InitializeQueryMigration(cx1client *Cx1ClientGo.Cx1Client) error {
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

	var err error
	cx1qc, err = cx1client.GetQueries()
	if err != nil {
		return err
	}

	QueryMapping, err = cx1client.GetQueryMappings()
	if err != nil {
		return err
	}

	return nil
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
