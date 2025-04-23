package main

import (
	//"crypto/tls"
	//"flag"

	"crypto/tls"
	"encoding/json"
	"flag"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

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
	var err error
	var cqc Cx1ClientGo.QueryCollection
	var data []byte
	InitializeQueryMigration(cx1client1)

	if data, err = os.ReadFile("queries.json"); err != nil {
		logger.Infof("Fetching queries from %v", cx1client1.String())

		var collection Cx1ClientGo.QueryCollection
		collection, err = cx1client1.GetQueries()
		if err != nil {
			logger.Fatalf("Failed to fetch queries from %v: %s", cx1client1.String(), err)
		}

		for _, lang := range collection.QueryLanguages {
			logger.Infof("Fetching %v-language queries", lang.Name)
			if err := refreshAuditSession(cx1client1, lang.Name); err != nil {
				logger.Errorf("Failed to refresh audit session: %v", err)
			} else {
				aq, err := cx1client1.GetAuditQueriesByLevelID(auditSession, cx1client1.QueryTypeProject(), testProject.ProjectID)
				if err != nil {
					logger.Errorf("Failed to get query code for Project-level %v queries for project %v: %s", lang.Name, testProject.String(), err)
				} else {
					collection.AddQueries(&aq)
				}
			}
		}

		if auditSession != nil {
			deleteAuditSession(cx1client1, logger)
		}

		cqc = collection.GetCustomQueryCollection()

		data, err = json.MarshalIndent(cqc, "", "  ")
		if err != nil {
			logger.Errorf("Failed to marshal data to json: %v", err)
			return
		}
		err = os.WriteFile("queries.json", data, 0644)
		if err != nil {
			logger.Errorf("Failed to write data to file: %v", err)
			return
		}
		logger.Infof("Saved query collection to queries.json")
	} else {
		logger.Infof("Loading queries from queries.json")
		err = json.Unmarshal(data, &cqc)
		if err != nil {
			logger.Errorf("Failed to unmarshal data from json: %v", err)
			return
		}
	}
}
