package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

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

	logger.Info("Starting")

	httpClient := &http.Client{}
	if true {
		proxyURL, _ := url.Parse("http://127.0.0.1:8080")
		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		httpClient.Transport = transport
		logger.Infof("Using proxy: %v", proxyURL.String())
	}

	Application := flag.String("app", "", "Optional: Name of application to process")
	Project := flag.String("proj", "", "Optional: Name of project to process")
	ApplyChange := flag.Bool("update", false, "Set this to true to actually apply changes, otherwise simply inform")

	cx1client, err := Cx1ClientGo.NewClient(httpClient, logger)
	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}
	logger.Infof("Connected with: %v", cx1client.String())

	if *Application != "" {
		err := ProcessApplicationTriage(cx1client, *Application, *ApplyChange, logger)
		if err != nil {
			logger.Fatalf("Failed to process application %v: %v", *Application, err)
		}
	} else if *Project != "" {
		err := ProcessProjectTriage(cx1client, *Project, *ApplyChange, logger)
		if err != nil {
			logger.Fatalf("Failed to process project %v: %v", *Project, err)
		}
	} else {
		logger.Fatalf("A project or application must be specified")
	}
}

func ProcessApplicationTriage(cx1client *Cx1ClientGo.Cx1Client, application string, applyChange bool, logger *logrus.Logger) error {
	logger.Infof("Processing application %v", application)

	if app, err := cx1client.GetApplicationByName(application); err != nil {
		return err
	} else {
		for _, projID := range app.ProjectIds {
			if proj, err := cx1client.GetProjectByID(projID); err != nil {
				return err
			} else {
				if err := processProject(cx1client, proj, applyChange, logger); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func ProcessProjectTriage(cx1client *Cx1ClientGo.Cx1Client, project string, applyChange bool, logger *logrus.Logger) error {
	if proj, err := cx1client.GetProjectByName(project); err != nil {
		return err
	} else {
		return processProject(cx1client, proj, applyChange, logger)
	}
}

func processProject(cx1client *Cx1ClientGo.Cx1Client, project Cx1ClientGo.Project, applyChange bool, logger *logrus.Logger) error {
	logger.Infof("Processing project %v", project.String())

	scanFilter := Cx1ClientGo.ScanFilter{
		ProjectID: project.ProjectID,
		Statuses:  []string{"Completed"},
	}

	last_scan, err := cx1client.GetLastScansByEngineFiltered("sast", 1, scanFilter)
	if err != nil {
		return err
	}
	if len(last_scan) == 0 {
		return fmt.Errorf("project %v has no successful SAST scans", project.String())
	}

	results, err := cx1client.GetAllScanResultsByID(last_scan[0].ScanID)
	if err != nil {
		return err
	}

	logger.Infof("Got %d SAST results", len(results.SAST))
	updatedCount := 0
	errCount := 0
	inScope := 0

	for _, result := range results.SAST {
		lastPredicate, err := cx1client.GetLastSASTResultsPredicateByID(result.SimilarityID, project.ProjectID, last_scan[0].ScanID)
		if err != nil {
			logger.Errorf("Failed to get latest predicate for project %v finding %v: %v", project.String(), result.String(), err)
		} else {
			if !strings.EqualFold(lastPredicate.CreatedBy, "importer") {
				continue
			} else {
				inScope++
				if applyChange {
					predicate := result.CreateResultsPredicate(project.ProjectID, last_scan[0].ScanID)
					predicate.State = "PROPOSED_NOT_EXPLOITABLE"
					if err = cx1client.AddSASTResultsPredicates([]Cx1ClientGo.SASTResultsPredicates{predicate}); err != nil {
						logger.Errorf("Failed to update project %v result %v to PNE (temporary): %v", project.String(), result.String(), err)
						errCount++
					} else {

						predicate.State = lastPredicate.State
						predicate.Comment = lastPredicate.Comment
						if err = cx1client.AddSASTResultsPredicates([]Cx1ClientGo.SASTResultsPredicates{predicate}); err != nil {
							logger.Errorf("Failed to update project %v result %v back to %v: %v", project.String(), result.String(), lastPredicate.State, err)
							errCount++
						} else {
							updatedCount++
						}
					}
				} else {
					logger.Infof("Would update project %v result %v", project.String(), result.String())
				}
			}
		}
	}

	if inScope == 0 {
		logger.Infof("No results in-scope (by 'importer') for project %v", project.String())
	} else {
		logger.Infof("%d/%d results in-scope (by 'importer') for project %v", inScope, len(results.SAST), project.String())
		if applyChange {
			if errCount > 0 {
				logger.Errorf("Only updated %d/%d results for project %v", updatedCount, inScope, project.String())
			} else {
				logger.Infof("All %d 'importer' results updated for project %v", updatedCount, project.String())
			}
		} else {
			logger.Infof("Update skipped - 'update' flag not set")
		}
	}

	return nil
}
