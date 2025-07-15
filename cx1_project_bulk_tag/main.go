package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"net/http"
	"net/url"
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
	ProjectsFile := flag.String("projects", "projectIds.txt", "File containing 1 project ID per line")
	ProjectTag := flag.String("tag", "cx336-ui-fix", "Tag to add and remove")
	Update := flag.Bool("update", false, "Apply the change or just inform")
	AddOnly := flag.Bool("add", false, "Only add the tag, do not remove")
	Delay := flag.Int("delay", 5, "Delay in seconds between projects")
	RemoveOnly := flag.Bool("remove", false, "Only remove the tag, do not add")

	logger.Info("Starting")
	client := &http.Client{}
	if false {
		proxyURL, _ := url.Parse("http://127.0.0.1:8080")
		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client.Transport = transport
	}

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
		logger.Warnf("This will update project tags by adding and removing tag: %v", *ProjectTag)
	} else {
		logger.Info("This will not make any changes, only inform")
	}
	ProjectIDs := []string{}

	{
		file, err := os.Open(*ProjectsFile)
		if err != nil {
			logger.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			//logger.Infof("Read line: %v", scanner.Text())
			ProjectIDs = append(ProjectIDs, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			logger.Fatal(err)
		}
	}

	logger.Infof("Checking %d projects...", len(ProjectIDs))

	for _, pid := range ProjectIDs {
		project, err := cx1client.GetProjectByID(pid)
		if err != nil {
			logger.Errorf("Failed to get project %v: %v", pid, err)
		} else {
			if *Update {
				if !*RemoveOnly {
					project.Tags[*ProjectTag] = ""
					err = cx1client.UpdateProject(&project)
					if err != nil {
						logger.Errorf("Failed to add tag %v to project %v: %v", *ProjectTag, project.String(), err)
					} else {
						logger.Infof("Successfully updated project %v (added tag)", project.String())
					}
				}

				if !*AddOnly {
					delete(project.Tags, *ProjectTag)
					err = cx1client.UpdateProject(&project)
					if err != nil {
						logger.Errorf("Failed to remove tag %v from project %v: %v", *ProjectTag, project.String(), err)
					} else {
						logger.Infof("Successfully updated project %v (removed tag)", project.String())
					}
				}
				time.Sleep(time.Duration(*Delay) * time.Second)
			} else {
				logger.Infof("Would update project %v by adding & removing tag %v", project.String(), *ProjectTag)
			}
		}
	}

	logger.Infof("Done")
}
