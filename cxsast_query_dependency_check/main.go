package main

import (
    "net/http"
    "os"
    "strings"

    "github.com/cxpsemea/CxSASTClientGo"
    "github.com/sirupsen/logrus"

    //    "time"
    //"fmt"
    "crypto/tls"
    "net/url"

    easy "github.com/t-tomalak/logrus-easy-formatter"
)

var logger *logrus.Logger

func main() {
    logger = logrus.New()
    logger.SetLevel(logrus.TraceLevel)
    myformatter := &easy.Formatter{}
    myformatter.TimestampFormat = "2006-01-02 15:04:05.000"
    myformatter.LogFormat = "[%lvl%][%time%] %msg%\n"
    logger.SetFormatter(myformatter)
    logger.SetOutput(os.Stdout)
    logger.Info("Starting")

    server_url := os.Args[1]
    username := os.Args[2]
    password := os.Args[3]

    proxyURL, err := url.Parse("http://127.0.0.1:8080")
    transport := &http.Transport{}
    transport.Proxy = http.ProxyURL(proxyURL)
    transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

    httpClient := &http.Client{}
    //httpClient.Transport = transport

    sastclient, err := CxSASTClientGo.NewTokenClient(httpClient, server_url, username, password, logger)
    if err != nil {
        logger.Error("Error creating client: " + err.Error())
        return
    }

    logger.Infof("Created client %v", sastclient.String())

    qc, _ := sastclient.GetQueriesSOAP()

    teams, _ := sastclient.GetTeams()
    projects, _ := sastclient.GetProjects()

    teamsById := make(map[uint64]*CxSASTClientGo.Team)
    for id, t := range teams {
        teamsById[t.TeamID] = &teams[id]
    }

    projectsById := make(map[uint64]*CxSASTClientGo.Project)
    for id, p := range projects {
        projectsById[p.ProjectID] = &projects[id]
    }
    
    qc.LinkBaseQueries( &teamsById, &projectsById )
    qc.DetectDependencies( &teamsById, &projectsById )
        

    for _, lang := range qc.QueryLanguages {
        for _, group := range lang.QueryGroups {
            for _, query := range group.Queries {
                if query.IsValid && query.IsCustom() {
                    refs := qc.GetQueryDependencies(&query)
                    if len(refs) > 0 {
                        logger.Infof("%v\n\t%v\n", query.StringDetailed(), strings.Join( refs, "\n\t" ) )
                    }
                }
            }
        }
    }
}
