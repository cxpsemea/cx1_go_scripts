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
	update := flag.Bool("update", false, "Set this to true to actually add the SAML mappers, otherwise only inform what would be added")

	httpClient := &http.Client{}

	var cx1client *Cx1ClientGo.Cx1Client
	var err error

	cx1client, err = Cx1ClientGo.NewClient(httpClient, logger)

	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	if !*update {
		logger.Warn("This tool will add default attribute mappers (firstname, lastname, username, role, group, email) to the specified SAML identity provider. Use with caution, this may overwrite existing mappers.")
		logger.Info("Running in informational mode only, no changes will be made (-update flag not set)")
	}

	logger.Infof("Connected with %v", cx1client.String())

	idp, err := cx1client.GetAuthenticationProviderByAlias(*providerName)
	if err != nil {
		logger.Fatalf("Unable to get idp: %s", err)
	}

	logger.Infof("Found IDP: %v", idp.String())

	mapperNames := []string{"firstname", "lastname", "username", "role", "group", "email"}

	for _, name := range mapperNames {
		if !*update {
			logger.Infof("Would add default mapper: %v", name)
			continue
		}
		mapper, _ := idp.MakeDefaultMapper(name)
		if err := cx1client.AddAuthenticationProviderMapper(mapper); err != nil {
			logger.Errorf("Failed to add mapper %v: %s", name, err)
		} else {
			logger.Infof("Added default mapper: %v", name)
		}
	}

}
