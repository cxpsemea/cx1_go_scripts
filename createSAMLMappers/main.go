package main

import (
	"flag"
	"net/http"
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

	providerName := flag.String("provider-alias", "", "Alias (display name) of the SAML IdP which already exists in CheckmarxOne")

	httpClient := &http.Client{}

	var cx1client *Cx1ClientGo.Cx1Client
	var err error

	cx1client, err = Cx1ClientGo.NewClient(httpClient, logger)

	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	idp, err := cx1client.GetAuthenticationProviderByAlias(*providerName)
	if err != nil {
		logger.Fatalf("Unable to get idp: %s", err)
	}

	logger.Infof("Found IDP: %v", idp.String())

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
