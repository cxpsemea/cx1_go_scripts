package main

import (
	"flag"
	"net/http"
	"os"

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

	APIKey := flag.String("apikey", "", "CheckmarxOne API Key (if not using client id/secret)")
	ClientID := flag.String("client", "", "CheckmarxOne Client ID (if not using API Key)")
	ClientSecret := flag.String("secret", "", "CheckmarxOne Client Secret (if not using API Key)")

	HTTPProxy := flag.String("proxy", "", "HTTP Proxy to use")

	flag.Parse()

	httpClient := &http.Client{}

	if *HTTPProxy != "" {
		proxyURL, err := url.Parse(*HTTPProxy)
		if err != nil {
			logger.Fatalf("Failed to parse url: %v", proxyURL)
		}

		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		httpClient.Transport = transport
	}

	if *Cx1URL == "" || *IAMURL == "" || *Tenant == "" || (*APIKey == "" && *ClientID == "" && *ClientSecret == "") || *SASTURL == "" || *SASTUser == "" || *SASTPass == "" {
		logger.Fatalf("Mandatory arguments are missing. Run with -h for a listing.")
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

	DoProcess(cx1client, &qc, &teamsById, &projectsById)
}
