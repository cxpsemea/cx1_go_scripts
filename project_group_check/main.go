package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

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
	logger.Info("The purpose of this tool is to validate that all projects have membership in valid groups.")

	change := flag.Bool("update", false, "Enable making updates to fix the issue")
	bulk := flag.Bool("bulk", false, "Update all projects (if true), otherwise do one project at a time")

	projectName := flag.String("project", "", "Name of a specific project to check")

	projectsFile := flag.String("project-file", "", "File containing a CheckmarxOne /api/projects response json")
	groupsFile := flag.String("group-file", "", "File containing a CheckmarxOne IAM /admin/../groups response json")

	doUpdate := false
	goOnline := true
	InvalidGroupIDs := []string{}

	var cx1client *Cx1ClientGo.Cx1Client
	var err error

	httpClient := &http.Client{}
	cx1client, err = Cx1ClientGo.NewClient(httpClient, logger)
	if err != nil {
		if *groupsFile != "" && *projectsFile != "" {
			logger.Infof("Both group-file and project-file were provided: working in offline mode")
		} else {
			logger.Fatalf("Error creating client: %s", err)
		}
	} else {
		logger.Infof("Connected: %v", cx1client.String())
	}

	if *change {
		if *groupsFile == "" && *projectsFile == "" {
			logger.Warnf("Running with update flag set - projects will be updated to remove invalid groups")
			doUpdate = true
		} else {
			logger.Errorf("Update flag was set but will be ignored because a group-file and/or project-file were provided. Update is only allowed when connecting to Cx1.")
		}
	} else {
		logger.Infof("Running without update flag set - no changes will be made")
	}

	if *bulk {
		logger.Infof("Running with bulk flag set - all projects will be processed")
	} else {
		logger.Infof("Running without bulk flag set - projects will be processed one at a time, with a prompt to continue")
	}

	var groups []Cx1ClientGo.Group

	if *groupsFile != "" {
		if data, err := os.ReadFile(*groupsFile); err != nil {
			logger.Fatalf("Failed to read specified groups file %v: %s", *groupsFile, err)
		} else {
			if err = json.Unmarshal(data, &groups); err != nil {
				logger.Fatalf("Failed to unmarshal groups file %v: %s", *groupsFile, err)
			}
		}
		logger.Infof("%d groups read from file %v", len(groups), *groupsFile)
	} else {
		groups, err = cx1client.GetGroups()
		if err != nil {
			logger.Fatalf("Failed to get groups: %s", err)
		}
	}

	var projects []Cx1ClientGo.Project

	if *projectsFile != "" {
		if data, err := os.ReadFile(*projectsFile); err != nil {
			logger.Fatalf("Failed to read specified projects file %v: %s", *projectsFile, err)
		} else {
			var response struct {
				Projects []Cx1ClientGo.Project `json:"projects"`
			}

			if err = json.Unmarshal(data, &response); err != nil {
				logger.Fatalf("Failed to unmarshal specified projects file %v: %s", *projectsFile, err)
			} else {
				projects = response.Projects
			}
		}
		logger.Infof("%d projects read from file %v", len(projects), *projectsFile)
	} else {
		projectCount, err := cx1client.GetProjectCount()
		if err != nil {
			logger.Fatalf("Failed to get project count: %s", err)
		}

		projects, err = cx1client.GetProjects(projectCount)
		if err != nil {
			logger.Fatalf("Failed to get projects: %s", err)
		}
	}

	logger.Infof("Processing %d projects and %d groups", len(projects), len(groups))
	if *projectName != "" {
		logger.Infof("Will evaluate only the specified project: %v", *projectName)
	}

	groupsByID := make(map[string]*Cx1ClientGo.Group)
	for i, group := range groups {
		groupsByID[group.GroupID] = &groups[i]
	}

	checkedProjects := 0
	fixedProjects := 0

	for _, project := range projects {
		if *projectName == "" || *projectName == project.Name {
			checkedProjects++
			goodGroupIDs := []string{}
			badGroupIDs := []string{}
			for _, groupId := range project.Groups {
				if _, ok := groupsByID[groupId]; ok {
					if !slices.Contains(goodGroupIDs, groupId) {
						goodGroupIDs = append(goodGroupIDs, groupId)
					}
				} else {
					if !slices.Contains(badGroupIDs, groupId) {
						badGroupIDs = append(badGroupIDs, groupId)
					}

					if !slices.Contains(InvalidGroupIDs, groupId) {
						InvalidGroupIDs = append(InvalidGroupIDs, groupId)
					}
				}
			}

			if len(badGroupIDs) > 0 {
				fixedProjects++
				logger.Warnf("Project %v contains %d valid groups and %d invalid groups.", project.String(), len(goodGroupIDs), len(badGroupIDs))
				logger.Warnf("Invalid groups: %v", strings.Join(badGroupIDs, ", "))
				if doUpdate && goOnline {
					logger.Infof("Updating project to remove the invalid groups")
					project.Groups = goodGroupIDs
					if err = cx1client.UpdateProject(&project); err != nil {
						logger.Errorf("Failed to update project %v: %s", project.String(), err)
					} else {
						logger.Infof("Updated project %v - removed %d invalid groups, %d valid groups remain.", project.String(), len(badGroupIDs), len(goodGroupIDs))
					}
				}
			} else {
				logger.Infof("Project %v contains %d valid groups and no invalid groups.", project.String(), len(goodGroupIDs))
			}

			if !*bulk && *projectName == "" {
				logger.Infof("Pausing between projects - continue? [y/n]")
				var str string
				fmt.Scan(&str)
				if !strings.EqualFold(str, "y") {
					logger.Infof("Process terminated by user.")
					break
				}

			}
		}
	}

	if *projectName != "" && checkedProjects == 0 {
		logger.Errorf("No projects were found matching the name: %v", *projectName)
	}

	logger.Infof("Finished processing. %d of %d projects were found to include invalid groups. The following %d invalid groups were found:", fixedProjects, len(projects), len(InvalidGroupIDs))
	for id, gid := range InvalidGroupIDs {
		logger.Infof("%d: %v", id+1, gid)
	}
}
