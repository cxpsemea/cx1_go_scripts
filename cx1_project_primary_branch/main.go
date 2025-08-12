package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

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

	LogLevel := flag.String("log", "INFO", "Log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL")
	ProjectsFile := flag.String("projects", "", "Optional: file containing 1 project ID per line (otherwise check all projects)")
	ApplicationName := flag.String("appName", "", "Optional: name of application containing projects to update")
	Update := flag.Bool("update", false, "Apply the change or just inform")
	Delay := flag.Int("delay", 1000, "Delay in milliseconds between projects")

	logger.Info("Starting")
	client := &http.Client{}
	/*
		if false {
			proxyURL, _ := url.Parse("http://127.0.0.1:8080")
			transport := &http.Transport{}
			transport.Proxy = http.ProxyURL(proxyURL)
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			client.Transport = transport
		}
	*/

	cx1client, err := Cx1ClientGo.NewClient(client, logger)
	if err != nil {
		logger.Fatalf("Failed to create client: %v", err)
	} else {
		logger.Infof("Connected with %v", cx1client.String())
	}

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

	if *Update {
		logger.Warnf("This will update projects by setting the primary branch for projects without a primary branch and only one scanned branch")
	} else {
		logger.Info("This will not make any changes, only inform")
	}

	Projects := []Cx1ClientGo.Project{}

	if *ProjectsFile != "" {
		logger.Infof("Parsing list of project IDs from %v", *ProjectsFile)
		file, err := os.Open(*ProjectsFile)
		if err != nil {
			logger.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		pcount := 1
		for scanner.Scan() {
			//logger.Infof("Read line: %v", scanner.Text())
			ProjectID := scanner.Text()
			project, err := cx1client.GetProjectByID(ProjectID)
			if err != nil {
				logger.Errorf("%d: Failed to get project %v: %v", pcount, ProjectID, err)
			} else {
				Projects = append(Projects, project)
			}
			pcount++
		}

		if err := scanner.Err(); err != nil {
			logger.Fatal(err)
		}
	} else if *ApplicationName != "" {
		logger.Infof("Fetching projects belonging to application %v", *ApplicationName)
		application, err := cx1client.GetApplicationByName(*ApplicationName)
		if err != nil {
			logger.Fatalf("Failed to get application %v: %v", *ApplicationName, err)
		}

		if len(*application.ProjectIds) == 0 {
			logger.Fatalf("Application %v has no projects", application.String())
		}

		for id, pid := range *application.ProjectIds {
			project, err := cx1client.GetProjectByID(pid)
			if err != nil {
				logger.Errorf("%d: Failed to get project %v: %v", id, pid, err)
			} else {
				Projects = append(Projects, project)
			}
		}
	} else {
		logger.Infof("Fetching all projects")
		Projects, err = cx1client.GetAllProjects()
		if err != nil {
			logger.Fatalf("Failed to get projects: %v", err)
		}
	}

	totalCount := len(Projects)
	logger.Infof("Processing %d projects...", totalCount)

	for i, project := range Projects {
		progress := fmt.Sprintf("[#%d/%d] ", i+1, totalCount)
		if project.MainBranch == "" {

			branches, err := cx1client.GetProjectBranchesByID(project.ProjectID)
			if err != nil {
				logger.Errorf("%vFailed to get branches for project %v: %v", progress, project.String(), err)
			} else if len(branches) != 1 {
				logger.Infof("%vSkipping project %v - has %d scanned branches", progress, project.String(), len(branches))
			} else {
				if *Update {
					err = cx1client.PatchProject(&project, Cx1ClientGo.ProjectPatch{MainBranch: &branches[0]})
					if err != nil {
						logger.Errorf("%vFailed to set primary branch '%v' on project %v: %v", progress, branches[0], project.String(), err)
					} else {
						logger.Infof("%vSuccessfully set primary branch '%v' on project %v", progress, branches[0], project.String())
					}

					if i+1 < totalCount {
						time.Sleep(time.Duration(*Delay) * time.Millisecond)
					}
				} else {
					logger.Infof("%vWould update project %v by setting the primary branch to '%v'", progress, project.String(), branches[0])
				}
			}
		} else {
			logger.Infof("%vSkipping project %v - already has primary branch '%v'", progress, project.String(), project.MainBranch)
		}
	}

	logger.Infof("Done")
}
