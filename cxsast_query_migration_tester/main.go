package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/cxpsemea/CxSASTClientGo"
	"github.com/sirupsen/logrus"

	//    "time"
	//"fmt"
	"crypto/tls"
	"net/url"

	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var logger *logrus.Logger

func main() {
	logger = logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	logger.SetOutput(os.Stdout)
	logger.Info("Starting")

	Cx1URL := flag.String("cx1", "", "CheckmarxOne platform URL")
	IAMURL := flag.String("iam", "", "CheckmarxOne IAM URL")
	Tenant := flag.String("tenant", "", "CheckmarxOne tenant")
	SASTURL := flag.String("sast", "", "Checkmarx SAST server URL")
	SASTUser := flag.String("user", "", "Checkmarx SAST username")
	SASTPass := flag.String("pass", "", "Checkmarx SAST password")
	LogLevel := flag.String("log", "INFO", "Log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL")

	APIKey := flag.String("apikey", "", "CheckmarxOne API Key (if not using client id/secret)")
	ClientID := flag.String("client", "", "CheckmarxOne Client ID (if not using API Key)")
	ClientSecret := flag.String("secret", "", "CheckmarxOne Client Secret (if not using API Key)")

	Output := flag.String("output", "queries.json", "Output file to generate with the target list of CheckmarxOne queries")
	Input := flag.String("input", "", "Input file to load and process, creating the list of queries in CheckmarxOne")

	TeamID := flag.Uint64("teamid", 0, "Create only queries for this team ID")
	ProjectID := flag.Uint64("projectid", 0, "Create only queries for this project ID")
	QueryID := flag.Uint64("queryid", 0, "Create only this specific query ID")

	Cx1Project := flag.String("cx1-project", "", "Optional: Name of the project in CheckmarxOne to target with the query migration")
	Cx1Application := flag.String("cx1-app", "", "Optional: Name of the application in CheckmarxOne to target with the query migration")

	HTTPProxy := flag.String("proxy", "", "HTTP Proxy to use")

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

	httpClient := &http.Client{}

	if *HTTPProxy != "" {
		proxyURL, err := url.Parse(*HTTPProxy)
		if err != nil {
			logger.Fatalf("Failed to parse proxy url: %v", proxyURL)
		}

		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		httpClient.Transport = transport
	}

	if *Cx1URL == "" || *IAMURL == "" || *Tenant == "" || (*APIKey == "" && *ClientID == "" && *ClientSecret == "") || *SASTURL == "" || *SASTUser == "" || *SASTPass == "" {
		logger.Fatalf("Mandatory arguments are missing. Run with -h for a listing. Expected usage is two steps:\n\t1. Including -output parameter to generate the list of queries to be written to CheckmarxOne.\n\t2. Including -input parameter to load the list and push the queries to CheckmarxOne.")
	}

	var cx1client *Cx1ClientGo.Cx1Client
	var err error

	if *APIKey != "" {
		cx1client, err = Cx1ClientGo.NewAPIKeyClient(httpClient, *Cx1URL, *IAMURL, *Tenant, *APIKey, logger)
	} else {
		cx1client, err = Cx1ClientGo.NewOAuthClient(httpClient, *Cx1URL, *IAMURL, *Tenant, *ClientID, *ClientSecret, logger)
	}

	if err != nil {
		logger.Fatalf("Error creating Cx1 client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	sastclient, err := CxSASTClientGo.NewTokenClient(httpClient, *SASTURL, *SASTUser, *SASTPass, logger)
	if err != nil {
		logger.Fatalf("Error creating CxSAST client: %s", err)
	}
	logger.Infof("Connected with %v", sastclient.String())

	qc, _ := sastclient.GetQueriesSOAP()

	teams, _ := sastclient.GetTeams()
	projects, _ := sastclient.GetProjects()

	teamsById := make(map[uint64]*CxSASTClientGo.Team)
	for id, t := range teams {
		teamsById[t.TeamID] = &teams[id]
	}

	projectsById := make(map[uint64]*CxSASTClientGo.Project)
	for id, p := range projects {
		projectsById[p.ProjectID] = &projects[id]
	}

	qc.LinkBaseQueries(&teamsById, &projectsById)
	qc.DetectDependencies(&teamsById, &projectsById)

	InitializeQueryMigration(cx1client)

	projectsPerTeam := make(map[uint64][]uint64)

	for _, project := range projectsById {
		if _, ok := projectsPerTeam[project.TeamID]; !ok {
			projectsPerTeam[project.TeamID] = []uint64{}
		}
		projectsPerTeam[project.TeamID] = append(projectsPerTeam[project.TeamID], project.ProjectID)
	}

	SetTargets(cx1client, *Cx1Project, *Cx1Application)

	if *Input != "" {
		RunMigrator(cx1client, &qc, &teamsById, &projectsById, &projectsPerTeam, *Input, *TeamID, *ProjectID, *QueryID, *Cx1Project, *Cx1Application)
	} else {
		RunGenerator(cx1client, &qc, &teamsById, &projectsById, *Output)
	}
}

func RunGenerator(cx1client *Cx1ClientGo.Cx1Client, qc *CxSASTClientGo.QueryCollection, teamsById *map[uint64]*CxSASTClientGo.Team, projectsById *map[uint64]*CxSASTClientGo.Project, outfile string) {
	ql := GenerateMigrationList(cx1client, qc, teamsById, projectsById)
	data, err := json.Marshal(ql)
	if err != nil {
		logger.Fatalf("Failed to marshal data: %s", err)
	}

	err = os.WriteFile(outfile, data, 0644)
	if err != nil {
		logger.Fatalf("Failed to write to %v: %s", outfile, err)
	}
	logger.Infof("Created output file %v", outfile)
}

func RunMigrator(cx1client *Cx1ClientGo.Cx1Client, qc *CxSASTClientGo.QueryCollection, teamsById *map[uint64]*CxSASTClientGo.Team, projectsById *map[uint64]*CxSASTClientGo.Project, projectsPerTeam *map[uint64][]uint64, infile string, teamID, projectID, queryID uint64, cx1proj, cx1app string) {
	data, err := os.ReadFile(infile)
	if err != nil {
		logger.Fatalf("Failed to load data: %s", err)
	}

	var ql QueriesList
	err = json.Unmarshal(data, &ql)
	if err != nil {
		logger.Fatalf("Failed to unmarshal data from %v: %s", infile, err)
	}

	// loaded queries in queries list do not have the query.OwningGroup set, need to fix that
	ql.FixGroups(qc)

	if teamID != 0 {
		MigrateTeamQueries(cx1client, qc, teamsById, projectsById, ql, teamID)
	} else if projectID != 0 {
		MigrateProjectQueries(cx1client, qc, teamsById, projectsById, ql, projectID)
	} else if queryID != 0 {
		MigrateQuery(cx1client, qc, teamsById, projectsById, ql, queryID)
	} else {
		MigrateQueries(cx1client, qc, teamsById, projectsById, projectsPerTeam, ql)
	}

	Summary()
}
