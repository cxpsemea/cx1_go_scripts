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
	BranchesFile := flag.String("branches", "", "Optional: file containing one <projectId, branchName> per line - best performance")
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

	//Projects := []Cx1ClientGo.Project{}
	ProjectBranches := make(map[string]string)

	if *ProjectsFile != "" {
		logger.Infof("Parsing list of project IDs from %v", *ProjectsFile)
		file, err := os.Open(*ProjectsFile)
		if err != nil {
			logger.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		pcount := 0
		for scanner.Scan() {
			//logger.Infof("Read line: %v", scanner.Text())
			ProjectID := scanner.Text()
			project, err := cx1client.GetProjectByID(ProjectID)
			if err != nil {
				logger.Errorf("%d: Failed to get project %v: %v", pcount+1, ProjectID, err)
			} else {
				branch, skipmsg, err := getPrimaryBranch(cx1client, &project)
				if err != nil {
					logger.Errorf("%d: %v", pcount+1, err)
				} else if skipmsg != "" {
					logger.Warningf("%d: %v", pcount+1, skipmsg)
				} else {
					ProjectBranches[ProjectID] = branch
				}
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
		logger.Infof("Application %v has %d projects", application.ApplicationID, len(*application.ProjectIds))

		for pcount, ProjectID := range *application.ProjectIds {
			project, err := cx1client.GetProjectByID(ProjectID)
			if err != nil {
				logger.Errorf("%d: Failed to get project %v: %v", pcount+1, ProjectID, err)
			} else {
				branch, skipmsg, err := getPrimaryBranch(cx1client, &project)
				if err != nil {
					logger.Errorf("%d: %v", pcount+1, err)
				} else if skipmsg != "" {
					logger.Warningf("%d: %v", pcount+1, skipmsg)
				} else {
					ProjectBranches[ProjectID] = branch
				}
			}
		}
	} else if *BranchesFile != "" {
		logger.Infof("Parsing list of project IDs and branches from %v", *BranchesFile)
		file, err := os.Open(*BranchesFile)
		if err != nil {
			logger.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		pcount := 0
		for scanner.Scan() {
			//logger.Infof("Read line: %v", scanner.Text())
			parts := strings.Split(scanner.Text(), ",")
			if len(parts) != 2 {
				logger.Errorf("%d: Failed to parse line '%v', should be in <projectId,branch> format", pcount+1, scanner.Text())
				continue
			}
			ProjectBranches[parts[0]] = parts[1]
			pcount++
		}

		if err := scanner.Err(); err != nil {
			logger.Fatal(err)
		}
	} else {
		logger.Infof("Fetching all projects")
		Projects, err := cx1client.GetAllProjects()
		if err != nil {
			logger.Fatalf("Failed to get projects: %v", err)
		}
		logger.Infof("Got %d projects", len(Projects))
		for pcount, project := range Projects {
			branch, skipmsg, err := getPrimaryBranch(cx1client, &project)
			if err != nil {
				logger.Errorf("%d: %v", pcount+1, err)
			} else if skipmsg != "" {
				logger.Warningf("%d: %v", pcount+1, skipmsg)
			} else {
				ProjectBranches[project.ProjectID] = branch
			}
		}
	}

	totalCount := len(ProjectBranches)
	logger.Infof("Processing %d projects...", totalCount)

	i := 1
	for projectId, branch := range ProjectBranches {
		progress := fmt.Sprintf("[#%d/%d] ", i, totalCount)
		if *Update {
			err = cx1client.PatchProjectByID(projectId, Cx1ClientGo.ProjectPatch{MainBranch: &branch})
			if err != nil {
				logger.Errorf("%vFailed to set primary branch '%v' on project %v: %v", progress, branch, projectId, err)
			} else {
				logger.Infof("%vSuccessfully set primary branch '%v' on project %v", progress, branch, projectId)
			}

			if i+1 < totalCount {
				time.Sleep(time.Duration(*Delay) * time.Millisecond)
			}
		} else {
			logger.Infof("%vWould update project %v by setting the primary branch to '%v'", progress, projectId, branch)
		}
		i++
	}

	logger.Infof("Done")
}

func getPrimaryBranch(cx1client *Cx1ClientGo.Cx1Client, project *Cx1ClientGo.Project) (string, string, error) {
	if project.MainBranch != "" {
		return "", fmt.Sprintf("Skipping project %v - already has primary branch '%v'", project.String(), project.MainBranch), nil
	}

	branches, err := cx1client.GetProjectBranchesByID(project.ProjectID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get branches for project %v: %v", project.String(), err)
	} else if len(branches) != 1 {
		return "", fmt.Sprintf("Skipping project %v - has %d scanned branches", project.String(), len(branches)), nil
	}

	return branches[0], "", nil
}
