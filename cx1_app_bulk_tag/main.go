package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
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
	AppsFile := flag.String("apps", "appIds.txt", "File containing 1 project ID per line")
	AppTag := flag.String("tag", "cx336-ui-fix", "Tag to add and remove")
	Update := flag.Bool("update", false, "Apply the change or just inform")
	AddOnly := flag.Bool("add", false, "Only add the tag, do not remove")
	Delay := flag.Int("delay", 1000, "Delay in milliseconds between projects")
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
		logger.Warnf("This will update project tags by adding and removing tag: %v", *AppTag)
	} else {
		logger.Info("This will not make any changes, only inform")
	}
	AppIDs := []string{}

	{
		file, err := os.Open(*AppsFile)
		if err != nil {
			logger.Fatalf("Failed to open file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			//logger.Infof("Read line: %v", scanner.Text())
			AppIDs = append(AppIDs, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			logger.Fatal(err)
		}
	}

	totalCount := len(AppIDs)
	logger.Infof("Checking %d apps...", totalCount)

	for i, appid := range AppIDs {
		app, err := cx1client.GetApplicationByID(appid)
		if err != nil {
			logger.Errorf("Failed to get application %v: %v", appid, err)
		} else {
			progress := fmt.Sprintf("[#%d/%d] ", i+1, totalCount)

			if *Update {
				if !*RemoveOnly {
					app.Tags[*AppTag] = ""
					err = cx1client.UpdateApplication(&app)
					if err != nil {
						logger.Errorf("%vFailed to add tag %v to application %v: %v", progress, *AppTag, app.String(), err)
					} else {
						logger.Infof("%vSuccessfully updated application %v (added tag)", progress, app.String())
					}
				}

				if !*AddOnly {
					delete(app.Tags, *AppTag)
					err = cx1client.UpdateApplication(&app)
					if err != nil {
						logger.Errorf("%vFailed to remove tag %v from application %v: %v", progress, *AppTag, app.String(), err)
					} else {
						logger.Infof("%vSuccessfully updated application %v (removed tag)", progress, app.String())
					}
				}
				time.Sleep(time.Duration(*Delay) * time.Millisecond)
			} else {
				logger.Infof("%vWould update application %v by adding & removing tag %v", progress, app.String(), *AppTag)
			}
		}
	}

	logger.Infof("Done")
}
