package main

import (
	"net/http"
	"os"

	// "fmt"

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
	logger.Info("This is just an example showing how to create a SAML user using the Cx1 client. The unique ID hardcoded in this example is for a specific user in my own Keycloak IdP and will not work for anyone else. The user details and unique ID should be replaced appropriately.")

	httpClient := &http.Client{}

	var cx1client *Cx1ClientGo.Cx1Client
	var err error

	cx1client, err = Cx1ClientGo.NewClient(httpClient, logger)

	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	user, err := cx1client.GetUserByEmail("groucho@cx.local")
	if err == nil {
		logger.Infof("User groucho@cx.local already exists, deleting")
		err = cx1client.DeleteUser(&user)
		if err != nil {
			logger.Errorf("Failed to delete user groucho: %s", err)
		}
	}

	user = Cx1ClientGo.User{
		Enabled:   true,
		FirstName: "Groucho",
		LastName:  "Marx",
		UserName:  "groucho",
		Email:     "groucho@cx.local",
	}

	user, err = cx1client.CreateSAMLUser(user, "dockerhost", "G-98cb32af-6f91-42fe-aad4-360c4420b8bb", "g-98cb32af-6f91-42fe-aad4-360c4420b8bb")
	if err != nil {
		logger.Errorf("Failed to create SAML user: %s", err)
	}
}
