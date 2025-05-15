package main

import (
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

var PNEComment = ""
var Update = false

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
	/*
		if true { // for debugging
			proxyURL, _ := url.Parse("http://127.0.0.1:8080")
			transport := &http.Transport{}
			transport.Proxy = http.ProxyURL(proxyURL)
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			httpClient.Transport = transport
			logger.Infof("Using proxy: %v", proxyURL.String())
		}
	*/

	ScanID := flag.String("scan", "", "Required: scan ID to check")
	LogLevel := flag.String("log", "info", "Log level: trace, debug, info, warning, error, fatal")
	ApplyChange := flag.Bool("update", false, "Set this to true to reapply the last predicate if there is an inconsistency")
	CommentText := flag.String("comment", "", "Optional: if the env requires a comment to be set when changing state to PNE, use this comment")

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

	Update = *ApplyChange
	PNEComment = *CommentText
	if *ScanID != "" {
		if err := ProcessScanResults(cx1client, *ScanID, logger); err != nil {
			logger.Errorf("Failed to process scan results: %v", err)
		} else {
			logger.Infof("Scan results processed")
		}
	} else {
		logger.Fatalf("A scan ID must be specified")
	}
}

func ProcessScanResults(cx1client *Cx1ClientGo.Cx1Client, scanID string, logger *logrus.Logger) error {
	logger.Infof("Processing scan %v", scanID)

	scan, err := cx1client.GetScanByID(scanID)
	if err != nil {
		return err
	}

	results, err := cx1client.GetAllScanResultsByID(scanID)
	if err != nil {
		return err
	}

	sast_results, err := cx1client.GetAllScanSASTResultsByID(scanID)
	if err != nil {
		return err
	}

	type ResultPair struct {
		SAST_Result *Cx1ClientGo.ScanSASTResult
		Result      *Cx1ClientGo.ScanSASTResult
	}

	simIDs := []string{}
	//					   simID	  resultHash
	result_map := make(map[string]map[string]ResultPair)

	for _, result := range sast_results {
		logger.Tracef("Parse sast_result %v/%v", result.SimilarityID, result.Data.ResultHash)
		if !slices.Contains(simIDs, result.SimilarityID) {
			simIDs = append(simIDs, result.SimilarityID)
		}

		if _, ok := result_map[result.SimilarityID]; !ok {
			result_map[result.SimilarityID] = make(map[string]ResultPair)
		}

		if _, ok := result_map[result.SimilarityID][result.Data.ResultHash]; !ok {
			result_map[result.SimilarityID][result.Data.ResultHash] = ResultPair{
				SAST_Result: &result,
			}
		} else {
			logger.Warningf("Conflict - scan has multiple api/sast-result findings with simID %v and result hash %v", result.SimilarityID, result.Data.ResultHash)
		}
	}

	for _, result := range results.SAST {
		logger.Tracef("Parse result %v/%v", result.SimilarityID, result.Data.ResultHash)
		if !slices.Contains(simIDs, result.SimilarityID) {
			simIDs = append(simIDs, result.SimilarityID)
		}

		if _, ok := result_map[result.SimilarityID]; !ok {
			result_map[result.SimilarityID] = make(map[string]ResultPair)
		}

		if val, ok := result_map[result.SimilarityID][result.Data.ResultHash]; !ok {
			result_map[result.SimilarityID][result.Data.ResultHash] = ResultPair{
				Result: &result,
			}
		} else {
			if val.Result != nil {
				logger.Warningf("Conflict - scan has multiple api/result findings with simID %v and result hash %v", result.SimilarityID, result.Data.ResultHash)
			} else {
				val.Result = &result
				result_map[result.SimilarityID][result.Data.ResultHash] = val
			}
		}
	}

	slices.Sort(simIDs)

	for _, simID := range simIDs {
		resultHashes := []string{}
		for _, result := range result_map[simID] {
			resultHash := ""
			if result.SAST_Result != nil {
				resultHash = result.SAST_Result.Data.ResultHash
			} else if result.Result != nil {
				resultHash = result.Result.Data.ResultHash
			}

			if result.SAST_Result == nil || result.Result == nil {
				logger.Warningf("Inconsistency for simID %v and result hash %v: api/sast-result gave %v while api/result gave %v", simID, resultHash, result.SAST_Result, result.Result)
				continue
			}

			if !slices.Contains(resultHashes, resultHash) {
				resultHashes = append(resultHashes, resultHash)
			}
		}

		slices.Sort(resultHashes)
		for _, hash := range resultHashes {
			result := result_map[simID][hash]
			if !compareResults(result.SAST_Result, result.Result, logger) && Update {
				// findings are different
				lastPredicate, err := cx1client.GetLastSASTResultsPredicateByID(result.SAST_Result.SimilarityID, scan.ProjectID, scan.ScanID)
				if err != nil {
					logger.Errorf("Failed to get latest predicate for scan %v finding %v: %v", scan.String(), result.SAST_Result.String(), err)
				} else {
					if err := addResultPredicate(cx1client, scan.ProjectID, scan.ScanID, result.SAST_Result.State, lastPredicate.Comment, *result.SAST_Result); err != nil {
						logger.Errorf("Failed to update result predicate for scan %v finding %v: %v", scan.String(), result.SAST_Result.String(), err)
					} else {
						logger.Infof("Updated result predicate for scan %v finding %v", scan.String(), result.SAST_Result.String())
					}
				}
			}
		}
	}

	return nil
}

func compareResults(sast_result *Cx1ClientGo.ScanSASTResult, result *Cx1ClientGo.ScanSASTResult, logger *logrus.Logger) bool {
	// both results are not nil, both results have matching simID and resultHash
	diffs := []string{}
	if sast_result.Severity != result.Severity {
		diffs = append(diffs, fmt.Sprintf("Severity: %v vs %v", sast_result.Severity, result.Severity))
	}
	if sast_result.State != result.State {
		diffs = append(diffs, fmt.Sprintf("State: %v vs %v", sast_result.State, result.State))
	}
	if sast_result.Status != result.Status {
		diffs = append(diffs, fmt.Sprintf("Status: %v vs %v", sast_result.Status, result.Status))
	}

	if len(diffs) > 0 {
		logger.Errorf("Inconsistency for simID %v and result hash %v: %v", sast_result.SimilarityID, sast_result.Data.ResultHash, strings.Join(diffs, ", "))
		return false
	} else {
		logger.Debugf("Results match for simID %v and result hash %v: %v", sast_result.SimilarityID, sast_result.Data.ResultHash, sast_result.String())
		return true
	}
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
