package main

import (
	"crypto/tls"
	"flag"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var logger *logrus.Logger

func main() {
	logger = logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	myformatter := &easy.Formatter{}
	myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
	myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
	logger.SetFormatter(myformatter)
	logger.SetOutput(os.Stdout)

	logger.Info("Starting")

	httpClient := &http.Client{}

	DeleteOldRules := flag.Bool("delete", false, "Delete old-type rules")
	Change := flag.Bool("update", false, "Make changes to project rules")

	cx1client, err := Cx1ClientGo.NewClient(httpClient, logger)
	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	projcount, err := cx1client.GetProjectCount()
	if err != nil {
		logger.Fatalf("Failed to get project count: %s", err)
	}
	projects, err := cx1client.GetProjects(projcount)
	if err != nil {
		logger.Fatalf("Failed to get projects: %s", err)
	}

	projectsByID := make(map[string]*Cx1ClientGo.Project)
	for id, p := range projects {
		projectsByID[p.ProjectID] = &projects[id]
	}

	appcount, err := cx1client.GetApplicationCount()
	if err != nil {
		logger.Fatalf("Failed to get application count: %s", err)
	}
	apps, err := cx1client.GetApplications(uint(appcount))
	if err != nil {
		logger.Fatalf("Failed to get applications: %s", err)
	}

	for _, a := range apps {
		processApp(cx1client, &a, &projectsByID, *DeleteOldRules, *Change, logger)
	}
}

func processApp(cx1client *Cx1ClientGo.Cx1Client, app *Cx1ClientGo.Application, projectsByID *map[string]*Cx1ClientGo.Project, delete, update bool, logger *logrus.Logger) {
	projectNames := []string{}
	extraProjects := []*Cx1ClientGo.Project{}
	oldRules := false

	logger.Infof("Processing application %v", app.String())

	for _, r := range app.Rules {
		if r.Type == "project.name.in" {
			pns := strings.Split(r.Value, ";")
			projectNames = append(projectNames, pns...)
		} else {
			oldRules = true
		}
	}

	//logger.Infof(" - app has %d projects by name, and %d project IDs", len(projectNames), len(app.ProjectIds))

	for _, pid := range app.ProjectIds {
		proj, ok := (*projectsByID)[pid]
		if !ok {
			logger.Errorf("Unknown project ID %v", pid)
		}

		if !slices.Contains(projectNames, proj.Name) {
			if !slices.Contains(extraProjects, proj) {
				extraProjects = append(extraProjects, proj)
			}
		}
	}

	if len(extraProjects) > 0 || oldRules {
		if len(extraProjects) > 0 {
			//logger.Infof("Application %v has %d extra projects to add by name:", app.String(), len(extraProjects))
			for _, p := range extraProjects {
				app.AssignProject(p)
				logger.Infof(" - to add project: %v", p.String())
			}
		}

		if oldRules {
			for id, r := range app.Rules {
				if r.Type != "project.name.in" {
					logger.Infof(" - to remove old-style rule %v with value %v", r.Type, r.Value)
					if delete {
						app.RemoveRule(&app.Rules[id])
					}
				}
			}
		}

		if update {
			if err := cx1client.UpdateApplication(app); err != nil {
				logger.Errorf("Failed to update application %v: %s", app.String(), err)
			} else {
				logger.Infof("Application %v updated", app.String())
			}
		}
	}

}
