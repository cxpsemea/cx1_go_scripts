package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var PNEComment = ""
var HistorySearch = false

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
	/*if true {
		proxyURL, _ := url.Parse("http://127.0.0.1:8080")
		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		httpClient.Transport = transport
		logger.Infof("Using proxy: %v", proxyURL.String())
	}*/

	Application := flag.String("app", "", "Optional: Name of application to process")
	Project := flag.String("proj", "", "Optional: Name of project to process")
	ProjectIDs := flag.String("project-ids", "", "Optional: Comma-separated list of project IDs to process")
	ProjectNames := flag.String("project-names", "", "Optional: Comma-separated list of project names to process")
	ApplyChange := flag.Bool("update", false, "Set this to true to actually apply changes, otherwise simply inform")
	CommentText := flag.String("comment", "", "Optional: if the env requires a comment to be set when changing state to PNE, use this comment")
	LogLevel := flag.String("log", "info", "Log level: trace, debug, info, warning, error, fatal")
	History := flag.Bool("history", false, "Optional: analyze the full predicate history (otherwise check only the latest predicate for 'importer' user)")

	cx1client, err := Cx1ClientGo.NewClient(httpClient, logger)
	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}
	logger.Infof("Connected with: %v", cx1client.String())

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

	if *CommentText != "" {
		PNEComment = *CommentText
	}

	HistorySearch = *History

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
	} else if *ProjectIDs != "" {
		err := ProcessProjectIDsTriage(cx1client, *ProjectIDs, *ApplyChange, logger)
		if err != nil {
			logger.Fatalf("Failed to process projects with ids %v: %v", *ProjectIDs, err)
		}
	} else if *ProjectNames != "" {
		err := ProcessProjectNamesTriage(cx1client, *ProjectNames, *ApplyChange, logger)
		if err != nil {
			logger.Fatalf("Failed to process projects with names %v: %v", *ProjectNames, err)
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
		for id, projID := range app.ProjectIds {
			if id%10 == 0 && id != 0 {
				logger.Infof("Progress: project %d of %d", id, len(app.ProjectIds))
			}
			if proj, err := cx1client.GetProjectByID(projID); err != nil {
				logger.Errorf("Failed to get project %v for application %v: %v", projID, app.String(), err)
			} else {
				if err := processProject(cx1client, proj, applyChange, logger); err != nil {
					logger.Warnf("Failed to process project %v for application %v: %v", proj.String(), app.String(), err)
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

func ProcessProjectIDsTriage(cx1client *Cx1ClientGo.Cx1Client, projectIDs string, applyChange bool, logger *logrus.Logger) error {
	ids := strings.Split(projectIDs, ",")
	for _, id := range ids {
		if proj, err := cx1client.GetProjectByID(id); err != nil {
			logger.Errorf("Failed to find project with id %v: %v", id, err)
		} else {
			if err = processProject(cx1client, proj, applyChange, logger); err != nil {
				logger.Errorf("Failed to process project %v: %v", proj.String(), err)
			}
		}
	}
	return nil
}

func ProcessProjectNamesTriage(cx1client *Cx1ClientGo.Cx1Client, projectNames string, applyChange bool, logger *logrus.Logger) error {
	names := strings.Split(projectNames, ",")
	for _, name := range names {

		if proj, err := cx1client.GetProjectByName(name); err != nil {
			logger.Errorf("Failed to find project with name %v: %v", name, err)
		} else {
			if err = processProject(cx1client, proj, applyChange, logger); err != nil {
				logger.Errorf("Failed to process project %v: %v", proj.String(), err)
			}
		}
	}
	return nil
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
		logger.Infof("Project %v has no successful SAST scans and will be skipped", project.String())
		return nil
	}

	results, err := cx1client.GetAllScanResultsByID(last_scan[0].ScanID)
	if err != nil {
		return err
	}

	updatedCount := 0
	errCount := 0
	inScope := 0

	for _, result := range results.SAST {
		if HistorySearch {
			//lastPredicate, err := cx1client.GetLastSASTResultsPredicateByID(result.SimilarityID, project.ProjectID, last_scan[0].ScanID)
			predicateHistory, err := cx1client.GetSASTResultsPredicatesByID(result.SimilarityID, project.ProjectID, last_scan[0].ScanID)
			if err != nil {
				logger.Warnf("Failed to get predicates for project %v finding %v: %v", project.String(), result.String(), err)
			} else {
				if changed, importedPredicate := historyChangedSinceImport(predicateHistory); !changed {
					if importedPredicate == nil {
						logger.Debugf("Finding %v was not imported", result.String())
						continue
					} else {
						inScope++
						if applyChange {
							if err := addResultPredicate(cx1client, project.ProjectID, last_scan[0].ScanID, importedPredicate.State, importedPredicate.Comment, result); err != nil {
								logger.Warnf("Failed to update project %v: %v", project.String(), err)
								errCount++
							} else {
								updatedCount++
								logger.Debugf("Updated project %v finding %v", project.String(), result.String())
							}
						} else {
							logger.Infof("Would update project %v result %v", project.String(), result.String())
						}
					}
				} else {
					logger.Debugf("Finding %v has already been updated manually", result.String())
				}
			}
		} else {
			lastPredicate, err := cx1client.GetLastSASTResultsPredicateByID(result.SimilarityID, project.ProjectID, last_scan[0].ScanID)
			if err != nil {
				logger.Warnf("Failed to get latest predicate for project %v finding %v: %v", project.String(), result.String(), err)
			} else {
				if !strings.EqualFold(lastPredicate.CreatedBy, "importer") {
					logger.Debugf("Finding %v had an update since import, skipping", result.String())
					continue
				} else {
					inScope++

					if applyChange {
						if err := addResultPredicate(cx1client, project.ProjectID, last_scan[0].ScanID, lastPredicate.State, lastPredicate.Comment, result); err != nil {
							logger.Warnf("Failed to update project %v: %v", project.String(), err)
							errCount++
						} else {
							updatedCount++
							logger.Debugf("Updated project %v finding %v", project.String(), result.String())
						}
					} else {
						logger.Infof("Would update project %v result %v", project.String(), result.String())
					}
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

func historyChangedSinceImport(predicateHistory []Cx1ClientGo.SASTResultsPredicates) (bool, *Cx1ClientGo.SASTResultsPredicates) {
	importerId := -1
	stateChanged := false
	importedState := ""

	for i := len(predicateHistory) - 1; i >= 0; i-- {
		predicate := predicateHistory[i]
		//fmt.Printf("Checking predicate %d: %v by %v\n", i, predicate.State, predicate.CreatedBy)
		if strings.EqualFold(predicate.CreatedBy, "importer") {
			importerId = i
			importedState = predicate.State
			//fmt.Printf(" - imported state: %v\n", predicate.State)
			stateChanged = false
		} else {
			if importerId > -1 && predicate.State != importedState {
				stateChanged = true
				//fmt.Printf(" - state changed: %v\n", predicate.State)
			}
		}
	}

	if importerId > -1 {
		return stateChanged, &predicateHistory[importerId]
	}

	return false, nil
}

func addResultPredicate(cx1client *Cx1ClientGo.Cx1Client, projectId, scanId, originalState, originalComment string, result Cx1ClientGo.ScanSASTResult) error {
	predicate := result.CreateResultsPredicate(projectId, scanId)
	predicate.State = "PROPOSED_NOT_EXPLOITABLE"
	if PNEComment != "" {
		predicate.Comment = PNEComment
	}
	if err := cx1client.AddSASTResultsPredicates([]Cx1ClientGo.SASTResultsPredicates{predicate}); err != nil {
		return fmt.Errorf("failed to update result %v to PNE (temporary): %v", result.String(), err)
	} else {
		predicate.State = originalState
		if originalComment != "" {
			predicate.Comment = originalComment
		} else {
			predicate.Comment = "Importer Triage Fix"
		}
		if err = cx1client.AddSASTResultsPredicates([]Cx1ClientGo.SASTResultsPredicates{predicate}); err != nil {
			return fmt.Errorf("failed to update result %v back to %v: %v", result.String(), originalState, err)
		}
	}
	return nil
}
