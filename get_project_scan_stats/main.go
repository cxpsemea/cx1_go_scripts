package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
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

	logger.Info("Starting")
	httpClient := &http.Client{}
	DaysBack := flag.Int("days", 30, "Retrieve a list of scans from the last # days")
	cx1client, err := Cx1ClientGo.NewClient(httpClient, logger)

	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	startTime := time.Now().AddDate(0, 0, -1**DaysBack)
	startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())

	projectsCreated := make(map[string]uint)
	scansCreated := make(map[string]uint)

	projectsFile, err := os.Create("projects.csv")
	if err != nil {
		logger.Fatalf("Failed to create projects.csv: %s", err)
	}
	defer projectsFile.Close()

	scansFile, err := os.Create("scans.csv")
	if err != nil {
		logger.Fatalf("Failed to create scans.csv: %s", err)
	}
	defer scansFile.Close()

	dates := []string{}
	for i := 0; i <= *DaysBack; i++ {
		curDate := startTime.AddDate(0, 0, i).Format("2006-01-02")
		dates = append(dates, curDate)
		projectsCreated[curDate] = 0
		scansCreated[curDate] = 0
	}

	p := cx1client.GetPaginationSettings()
	p.Scans = 1000
	p.Projects = 1000
	cx1client.SetPaginationSettings(p)

	logger.Infof("Retrieving a list of projects")
	projects, err := cx1client.GetAllProjects()
	if err != nil {
		logger.Fatalf("Failed to get projects: %s", err)
	}

	logger.Infof("Got %d projects", len(projects))
	for _, p := range projects {
		//logger.Infof("%v: created %v", p.String(), p.CreatedAt)
		t, err := time.Parse("2006-01-02T15:04:05.999999Z", p.CreatedAt)
		if err != nil {
			logger.Errorf("Failed to parse time: %v", p.CreatedAt)
		} else {
			tstr := t.Format("2006-01-02")
			projectsCreated[tstr] = projectsCreated[tstr] + 1
		}
	}

	logger.Infof("Writing projects to projects.csv")
	projectsFile.WriteString("sep=;\n")
	projectsFile.WriteString("Date;# Projects Created;\n")
	for _, date := range dates {
		count := projectsCreated[date]
		_, err := projectsFile.WriteString(fmt.Sprintf("%v;%d;\n", date, count))
		if err != nil {
			logger.Fatalf("Failed to write project data to file: %s", err)
		}
	}

	logger.Infof("Retrieving last %d days of scans (since %v)", *DaysBack, startTime)

	filter := Cx1ClientGo.ScanFilter{
		FromDate: startTime,
		BaseFilter: Cx1ClientGo.BaseFilter{
			Limit: p.Scans,
		},
	}
	_, scans, err := cx1client.GetAllScansFiltered(filter)
	if err != nil {
		logger.Fatalf("Failed to get scans: %s", err)
	}
	logger.Infof("Fetched %d scans", len(scans))
	for _, s := range scans {
		t, err := time.Parse("2006-01-02T15:04:05.999999Z", s.CreatedAt)
		if err != nil {
			logger.Errorf("Failed to parse time: %v", s.CreatedAt)
		} else {
			tstr := t.Format("2006-01-02")
			scansCreated[tstr] = scansCreated[tstr] + 1
		}
	}

	logger.Infof("Writing scans to scans.csv")
	scansFile.WriteString("sep=;\n")
	scansFile.WriteString("Date;# Scans Started;\n")
	for _, date := range dates {
		count := scansCreated[date]
		_, err := scansFile.WriteString(fmt.Sprintf("%v;%d;\n", date, count))
		if err != nil {
			logger.Fatalf("Failed to write scan data to file: %s", err)
		}
	}

	logger.Infof("Done")
}
