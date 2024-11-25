package main

import (
	//"crypto/tls"
	//"flag"
	"flag"
	"fmt"
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

	DeleteGroups := flag.Bool("delete", false, "Toggle to delete all previously-created groups")
	NumberOfGroups := flag.Int("count", 100, "Number of groups to create")

	logger.Info("Starting")
	httpClient := &http.Client{}
	cx1client, err := Cx1ClientGo.NewClient(httpClient, logger)

	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	if *DeleteGroups {
		groups, err := cx1client.GetGroupsByName("testgroup-")
		if err != nil {
			logger.Fatalf("Failed to get groups: %s", err)
		}
		for _, g := range groups {
			err := cx1client.DeleteGroup(&g)
			if err != nil {
				logger.Errorf("Failed to delete group %v: %s", g.String(), err)
			} else {
				logger.Infof("Deleted group %v", g.String())
			}
		}
	} else {
		for i := 1; i <= *NumberOfGroups; i++ {
			logger.Infof("Creating group batch %d", i)
			err := createGroups(cx1client, i)
			if err != nil {
				logger.Errorf("Failed while creating group batch %d: %s", i, err)
			}
		}
	}

	logger.Infof("Done!")
}

func createGroups(cx1client *Cx1ClientGo.Cx1Client, id int) error {
	groupName := fmt.Sprintf("testgroup-%04d", id)
	group, err := cx1client.CreateGroup(groupName)

	if err != nil {
		return err
	}

	if _, err = cx1client.CreateChildGroup(&group, "Owners"); err != nil {
		return err
	}
	if _, err = cx1client.CreateChildGroup(&group, "Scanners"); err != nil {
		return err
	}
	if _, err = cx1client.CreateChildGroup(&group, "Reviewers"); err != nil {
		return err
	}
	if _, err = cx1client.CreateChildGroup(&group, "Readers"); err != nil {
		return err
	}
	if _, err = cx1client.CreateChildGroup(&group, "Service Accounts"); err != nil {
		return err
	}
	return nil
}
