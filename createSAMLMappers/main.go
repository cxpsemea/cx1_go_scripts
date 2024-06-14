package main

import (
	"crypto/tls"
	"flag"
	"net/http"
	"net/url"
	"os"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	logger.SetOutput(os.Stdout)

	logger.Info("Starting")

	Cx1URL := flag.String("cx1", "", "CheckmarxOne platform URL")
	IAMURL := flag.String("iam", "", "CheckmarxOne IAM URL")
	Tenant := flag.String("tenant", "", "CheckmarxOne tenant")

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

	if *Cx1URL == "" || *IAMURL == "" || *Tenant == "" || (*APIKey == "" && *ClientID == "" && *ClientSecret == "") {
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
		logger.Fatalf("Error creating client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	idp, err := cx1client.GetAuthenticationProviderByAlias("dockerhost")
	if err != nil {
		logger.Fatalf("Unable to get idp: %s", err)
	}

	logger.Infof("Created IDP: %v", idp.String())

	mapper, _ := idp.MakeDefaultMapper("firstname")
	_ = cx1client.AddAuthenticationProviderMapper(mapper)
	mapper, _ = idp.MakeDefaultMapper("lastname")
	_ = cx1client.AddAuthenticationProviderMapper(mapper)
	mapper, _ = idp.MakeDefaultMapper("username")
	_ = cx1client.AddAuthenticationProviderMapper(mapper)
	mapper, _ = idp.MakeDefaultMapper("role")
	_ = cx1client.AddAuthenticationProviderMapper(mapper)
	mapper, _ = idp.MakeDefaultMapper("group")
	_ = cx1client.AddAuthenticationProviderMapper(mapper)
	mapper, _ = idp.MakeDefaultMapper("email")
	_ = cx1client.AddAuthenticationProviderMapper(mapper)

}
