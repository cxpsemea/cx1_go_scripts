package main

import ( 
	"net/http"
	"os"
	"github.com/sirupsen/logrus"
	"github.com/cxpsemea/Cx1ClientGo"
)

func main() {
	logger := logrus.New()
	base_url := os.Args[1]
	iam_url := os.Args[2]
	tenant := os.Args[3]
	api_key := os.Args[4]

	
	cx1client, err := Cx1ClientGo.NewAPIKeyClient( &http.Client{}, base_url, iam_url, tenant, api_key, logger )
	if err != nil {
		logger.Fatalf( "Error creating client: %s\n", err )
	}

	scans, err := cx1client.GetLastScansByStatusAndID("",1000,[]string{ "Queued" } )
	if err != nil { 
		logger.Fatalf( "Failed to get scans: %s", err )
	}
	
	for _, s := range scans {
		cx1client.DeleteScanByID( s.ScanID )
	}
}

