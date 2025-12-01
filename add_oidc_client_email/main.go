package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

	"encoding/csv"
	"io"

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
	EmailsList := flag.String("emails", "emails.csv", "Input file containing lines with: <client_id>,<emails;to;add>")
	DoUpdate := flag.Bool("update", false, "Enable OIDC client expiry update")
	cx1client, err := Cx1ClientGo.NewClient(httpClient, logger)

	if err != nil {
		logger.Fatalf("Error creating client: %s", err)
	}

	logger.Infof("Connected with %v", cx1client.String())

	if emailMap, err := parseUpdates(*EmailsList); err != nil {
		logger.Fatalf("Failed to parse input file %v: %v", *EmailsList, err)
	} else {
		logger.Infof("Parsed input file %v with %d clients to update", *EmailsList, len(emailMap))

		if *DoUpdate {
			logger.Infof("Will update OIDC clients listed in %v by adding the corresponding email addresses", *EmailsList)
		} else {
			logger.Infof("Will not make any changes, only inform (-update flag not set)")
		}

		for clientID, emails := range emailMap {
			oidcClient, err := cx1client.GetClientByName(clientID)
			if err != nil {
				logger.Errorf("Failed to retrieve OIDC client named %v: %v", clientID, err)
			} else {
				changed := false
				logger.Infof("Request to update client %v with additional emails: %v", clientID, strings.Join(emails, ", "))

				for _, email := range emails {
					if !slices.Contains(oidcClient.NotificationEmails, email) {
						oidcClient.NotificationEmails = append(oidcClient.NotificationEmails, email)
						changed = true
					}
				}

				if changed {
					if *DoUpdate {
						err := cx1client.UpdateClient(oidcClient)
						if err != nil {
							logger.Errorf("Failed to update OIDC client named %v: %v", clientID, err)
						} else {
							logger.Infof("Updated OIDC client named %v with emails: %v", clientID, strings.Join(oidcClient.NotificationEmails, ", "))
						}
					} else {
						logger.Infof("Would have updated OIDC client named %v with emails: %v", clientID, strings.Join(oidcClient.NotificationEmails, ", "))
					}
				} else {
					logger.Infof("No changes required for OIDC client named %v, current emails: %v", clientID, strings.Join(oidcClient.NotificationEmails, ", "))
				}
			}
		}
	}
}

func parseUpdates(inputFile string) (map[string][]string, error) {
	// The input file will have multiple lines following the format:
	// <client_id>,<emails;to;add>,[possible extra columns to be ignored]
	file, err := os.Open(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file %s: %w", inputFile, err)
	}
	defer file.Close()

	emailMap := make(map[string][]string)
	reader := csv.NewReader(file)
	// Allow variable number of fields per row to ignore extra columns
	reader.FieldsPerRecord = -1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			return nil, fmt.Errorf("error reading csv record: %w", err)
		}

		if len(record) < 2 {
			return nil, fmt.Errorf("malformed line, expected at least 2 columns, got %d for record: %v", len(record), record)
		}

		clientID := strings.TrimSpace(record[0])
		emails := strings.Split(record[1], ";")
		emailMap[clientID] = emails
	}

	return emailMap, nil
}
