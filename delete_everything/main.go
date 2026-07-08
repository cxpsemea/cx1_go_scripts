package main

import (
	"flag"
	"net/http"
	"os"
	"strings"

	// "fmt"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	logger.SetOutput(os.Stdout)

	logger.Info("Starting")
	Scope := flag.String("scope", "", "Comma-separated list of things to delete: projects,applications,groups,presets")
	Update := flag.Bool("update", false, "Set this to true to actually delete, otherwise only inform what would be deleted")

	httpClient := &http.Client{}

	var cx1client *Cx1ClientGo.Cx1Client
	var err error

	cx1client, err = Cx1ClientGo.NewClient(httpClient, logger)

	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	if *Scope == "" {
		logger.Fatalf("Scope parameter is required. Run with -h for a listing of options.")
	}

	if !*Update {
		logger.Warn("This tool will delete ALL projects/applications/groups/presets in the selected scope. Use with caution, this is irreversible.")
		logger.Info("Running in informational mode only, no changes will be made (-update flag not set)")
	}

	logger.Infof("Connected with %v", cx1client.String())

	scopes := strings.Split(strings.ToLower(*Scope), ",")

	for _, s := range scopes {
		switch s {
		case "projects":
			deleteProjects(cx1client, logger, *Update)
		case "applications":
			deleteApplications(cx1client, logger, *Update)
		case "groups":
			deleteGroups(cx1client, logger, *Update)
		case "presets":
			deletePresets(cx1client, logger, *Update)
		default:
			logger.Errorf("Unknown scope parameter: %v", s)
		}
	}

}

func deleteProjects(cx1client *Cx1ClientGo.Cx1Client, logger *logrus.Logger, update bool) {
	if projects, err := cx1client.GetProjects(0); err != nil {
		logger.Errorf("Failed to get projects: %s", err)
	} else {
		for _, project := range projects {
			if !update {
				logger.Infof("Would delete project %v", project.String())
				continue
			}
			if err = cx1client.DeleteProject(&project); err != nil {
				logger.Errorf("Failed to delete project %v: %s", project.String(), err)
			} else {
				logger.Infof("Deleted project %v", project.String())
			}
		}
	}
}

func deleteApplications(cx1client *Cx1ClientGo.Cx1Client, logger *logrus.Logger, update bool) {
	if applications, err := cx1client.GetApplications(0); err != nil {
		logger.Errorf("Failed to get applications: %s", err)
	} else {
		for _, application := range applications {
			if !update {
				logger.Infof("Would delete application %v", application.String())
				continue
			}
			if err = cx1client.DeleteApplication(&application); err != nil {
				logger.Errorf("Failed to delete application: %s %v", application.String(), err)
			} else {
				logger.Infof("Deleted application %v", application.String())
			}
		}
	}
}

func deleteGroups(cx1client *Cx1ClientGo.Cx1Client, logger *logrus.Logger, update bool) {
	if groups, err := cx1client.GetGroups(); err != nil {
		logger.Errorf("Failed to get groups: %s", err)
	} else {
		for _, group := range groups {
			if !update {
				logger.Infof("Would delete group %v", group.String())
				continue
			}
			if err = cx1client.DeleteGroup(&group); err != nil {
				logger.Errorf("Failed to delete group %v: %s", group.String(), err)
			} else {
				logger.Infof("Deleted group %v", group.String())
			}
		}
	}
}

func deletePresets(cx1client *Cx1ClientGo.Cx1Client, logger *logrus.Logger, update bool) {
	if count, err := cx1client.GetPresetCount(); err != nil {
		logger.Errorf("Failed to get preset count: %s", err)
	} else {
		if presets, err := cx1client.GetPresets(count); err != nil {
			logger.Errorf("Failed to get presets: %s", err)
		} else {
			for _, preset := range presets {
				if !update {
					logger.Infof("Would delete preset %v", preset.String())
					continue
				}
				if err = cx1client.DeletePreset(&preset); err != nil {
					logger.Errorf("Failed to delete preset %v: %s", preset.String(), err)
				} else {
					logger.Infof("Deleted preset %v", preset.String())
				}
			}
		}
	}
}
