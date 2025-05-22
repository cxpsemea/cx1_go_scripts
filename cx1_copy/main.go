package main

import (
	//"crypto/tls"
	//"flag"

	"crypto/tls"
	"flag"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var oldAPI bool = false
var languageScope = []string{}
var presetScope = []string{}

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
	logger.Info("This tool will copy presets and/or queries from the 'src' environment to the 'dest' environment")

	LogLevel := flag.String("log", "INFO", "Log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL")

	APIKey1 := flag.String("src-apikey", "", "CheckmarxOne API Key (if not using client id/secret)")
	ClientID1 := flag.String("src-client", "", "CheckmarxOne Client ID (if not using API Key)")
	ClientSecret1 := flag.String("src-secret", "", "CheckmarxOne Client Secret (if not using API Key)")
	Cx1URL1 := flag.String("src-cx", "", "Optional: CheckmarxOne platform URL (if using client id/secret)")
	IAMURL1 := flag.String("src-iam", "", "Optional: CheckmarxOne IAM URL (if using client id/secret)")
	Tenant1 := flag.String("src-tenant", "", "Optional: CheckmarxOne tenant (if using client id/secret)")
	Proxy1 := flag.String("src-proxy", "", "Optional: Proxy to use when connecting to CheckmarxOne")

	APIKey2 := flag.String("dest-apikey", "", "CheckmarxOne API Key (if not using client id/secret)")
	ClientID2 := flag.String("dest-client", "", "CheckmarxOne Client ID (if not using API Key)")
	ClientSecret2 := flag.String("dest-secret", "", "CheckmarxOne Client Secret (if not using API Key)")
	Cx1URL2 := flag.String("dest-cx", "", "Optional: CheckmarxOne platform URL (if using client id/secret)")
	IAMURL2 := flag.String("dest-iam", "", "Optional: CheckmarxOne IAM URL (if using client id/secret)")
	Tenant2 := flag.String("dest-tenant", "", "Optional: CheckmarxOne tenant (if using client id/secret)")
	Proxy2 := flag.String("dest-proxy", "", "Optional: Proxy to use when connecting to CheckmarxOne")

	Scope := flag.String("scope", "", "Comma-separated list of items to copy: queries,presets")
	Languages := flag.String("languages", "", "Optional: When migrating queries, only cover the languages in this comma-separated list, eg: javascript,java")
	Presets := flag.String("presets", "", "Optional: When migrating presets, only include the presets in this comma-separated list, eg: My_Preset1,My_Preset2")

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
	if *Presets != "" {
		presetScope = strings.Split(*Presets, ",")
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
		CopyPresets(cx1client1, cx1client2, logger)
	}

	return 0
}
