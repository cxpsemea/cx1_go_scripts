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
	logger.SetLevel(logrus.TraceLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	logger.SetOutput(os.Stdout)

	logger.Info("Starting")
	httpClient := &http.Client{}
	MinimumExpiry := flag.Uint64("expiry", 180, "Minimum days for expiry")
	DoUpdate := flag.Bool("update", false, "Erun-denable OIDC client expiry update")
	cx1client, err := Cx1ClientGo.NewClient(httpClient, logger)

	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	clients, err := cx1client.GetClients()
	if err != nil {
		logger.Fatalf("Failed to get clients: %s", err)
	}

	logger.Infof("Checking for OIDC Clients with expiry > %d", *MinimumExpiry)

	for _, c := range clients {
		if c.Creator != "" && c.SecretExpirationDays > *MinimumExpiry {
			logger.Infof("Client %v expires in %d days", c.String(), c.SecretExpirationDays)
			if *DoUpdate {
				c.SecretExpirationDays = *MinimumExpiry
				if err := cx1client.UpdateClient(c); err != nil {
					logger.Errorf("Failed to update client %v: %s", c.String(), err)
				} else {
					logger.Infof("Updated client expiry to %d", *MinimumExpiry)
				}
			}
		}
	}
}
