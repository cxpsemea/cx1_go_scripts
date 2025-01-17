package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

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

	logger.Info("This tool will fetch a list of scans run in the last [LastMonths] months and store it in scanids.txt if the file doesn't already exist, otherwise it will retrieve the list from scanids.txt. For each scan, if it is for a previously-unprocessed Project + Branch, it will generate and download a CSV format report and place it into the reports subfolder, named <scanid>.csv.")
	httpClient := &http.Client{}
	LastMonths := flag.Int("months", 3, "Number of months back when retrieving scan reports")
	Pause := flag.Int("pause", 5, "Number of seconds to pause between report generation requests")
	cx1client, err := Cx1ClientGo.NewClient(httpClient, logger)

	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	var lines []string

	if _, err := os.Stat("scanids.txt"); err == nil {
		// path/to/whatever exists
		content, err := os.ReadFile("scanids.txt")
		if err != nil {
			logger.Fatalf("Failed to read file scanids.txt: %s", err)
		}

		lines = strings.Split(string(content), "\n")
		if lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		logger.Infof("Read %d scan ids", len(lines))
	} else {
		f, err := os.Create("scanids.txt")

		if err != nil {
			logger.Fatalf("Failed to create scanids.txt: %s", err)
		}
		defer f.Close()

		filter := Cx1ClientGo.ScanFilter{
			FromDate: time.Now().AddDate(0, -1*(*LastMonths), 0),
		}
		count, scans, err := cx1client.GetAllScansFiltered(filter)
		if err != nil {
			logger.Fatalf("Failed to get scans: %s", err)
		}
		logger.Infof("Fetched %d scans", count)

		for _, s := range scans {
			f.WriteString(fmt.Sprintf("%v\n", s.ScanID))
			lines = append(lines, s.ScanID)
		}

	}

	logger.Infof("Processing %d scan IDs", len(lines))
	if err = os.Mkdir("reports", 0755); err != nil && !os.IsExist(err) {
		logger.Fatalf("Failed to create reports directory: %s", err)
	}

	ProjectBranches := make(map[string][]string)

	for _, scanid := range lines {
		scan, err := cx1client.GetScanByID(scanid)
		if err != nil {
			logger.Errorf("Failed to get scan %v: %s", scanid, err)
		} else {
			filename := fmt.Sprintf("reports/%v.csv", scanid)
			branches, processed := ProjectBranches[scan.ProjectID]
			exists := false
			if _, err := os.Stat(filename); err == nil {
				exists = true
			}

			if !exists && !processed && !slices.Contains(branches, scan.Branch) {
				ProjectBranches[scan.ProjectID] = append(ProjectBranches[scan.ProjectID], scan.Branch)

				reportId, err := cx1client.RequestNewReportByID(scanid, scan.ProjectID, scan.Branch, "csv", []string{"SAST"}, []string{"ScanSummary", "ExecutiveSummary", "ScanResults"})
				if err != nil {
					logger.Errorf("Failed to request new report for scan id %v: %s", scanid, err)
				} else {
					url, err := cx1client.ReportPollingByID(reportId)
					if err != nil {
						logger.Errorf("Failed while generating report for scan id %v: %s", scanid, err)
					} else {
						report, err := cx1client.DownloadReport(url)
						if err != nil {
							logger.Errorf("Failed to download report for scan id %v: %s", scanid, err)
						} else {
							if err := os.WriteFile(filename, report, 0644); err != nil {
								logger.Errorf("Failed while saving report to %v: %s", filename, err)
							} else {
								logger.Infof("Saved report to %v", filename)
							}
						}
					}
				}

				time.Sleep(time.Second * time.Duration(*Pause))
			} else {
				logger.Infof("Skipping scan id %v: belongs to already-processed project %v branch %v", scanid, scan.ProjectName, scan.Branch)
			}
		}
	}
}
