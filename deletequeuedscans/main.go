package main

import (
	"flag"
	"net/http"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	api_key := flag.String("apikey", "", "CheckmarxOne API Key (if not using client id/secret)")
	//ClientID := flag.String("client1", "", "CheckmarxOne Client ID (if not using API Key)")
	//ClientSecret := flag.String("secret1", "", "CheckmarxOne Client Secret (if not using API Key)")
	base_url := flag.String("cx1", "", "Optional: CheckmarxOne platform URL, if not defined in the test config.yaml")
	iam_url := flag.String("iam", "", "Optional: CheckmarxOne IAM URL, if not defined in the test config.yaml")
	tenant := flag.String("tenant", "", "Optional: CheckmarxOne tenant, if not defined in the test config.yaml")
	flag.Parse()

	cx1client, err := Cx1ClientGo.NewAPIKeyClient(&http.Client{}, *base_url, *iam_url, *tenant, *api_key, logger)
	if err != nil {
		logger.Fatalf("Error creating client: %s\n", err)
	}

	scans, err := cx1client.GetLastScansByStatusAndID("", 1000, []string{"Queued"})
	if err != nil {
		logger.Fatalf("Failed to get scans: %s", err)
	}
	logger.Info("Found ", len(scans), " queued scans")

	for _, s := range scans {
		logger.Info("Deleting scan ID ", s.ScanID, " Name: ", s.ProjectName)
		err = cx1client.CancelScanByID(s.ScanID)
		if err != nil {
			logger.Errorf("Failed to delete scan ID %s: %s", s.ScanID, err)
		}
	}
}
