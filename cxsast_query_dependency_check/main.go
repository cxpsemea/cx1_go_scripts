package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"slices"
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

	SASTURL := flag.String("sast", "", "Checkmarx SAST server URL")
	SASTUser := flag.String("user", "", "Checkmarx SAST username")
	SASTPass := flag.String("pass", "", "Checkmarx SAST password")
	LogLevel := flag.String("log", "INFO", "Log level: TRACE, DEBUG, INFO, WARNING, ERROR, FATAL")

	ProjectID := flag.Uint64("projectId", 0, "Checkmarx SAST Project ID (optional)")
	ProjectName := flag.String("projectName", "", "Checkmarx SAST Project Name (optional)")

	NoCorp := flag.Bool("nocorp", false, "Don't show Corporate-query dependencies (use if all corp queries were already migrated)")
	NoProd := flag.Bool("noprod", false, "Don't show Product-default query dependencies (use if CxSAST and Cx1 versions match)")

	HTTPProxy := flag.String("proxy", "", "HTTP Proxy to use")

	flag.Parse()

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

	httpClient := &http.Client{}

	if *HTTPProxy != "" {
		proxyURL, err := url.Parse(*HTTPProxy)
		if err != nil {
			logger.Fatalf("Failed to parse url: %v", proxyURL)
		}

		transport := &http.Transport{}
		transport.Proxy = http.ProxyURL(proxyURL)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		httpClient.Transport = transport
	}

	if *SASTURL == "" || *SASTUser == "" || *SASTPass == "" {
		logger.Fatalf("Mandatory arguments are missing. Run with -h for a listing.")
	}

	sastclient, err := CxSASTClientGo.NewTokenClient(httpClient, *SASTURL, *SASTUser, *SASTPass, logger)
	if err != nil {
		logger.Fatalf("Error creating CxSAST client: %s", err)
	}

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

	qc.LinkBaseQueries(&teamsById, &projectsById)
	qc.DetectDependencies(&teamsById, &projectsById)

	if *ProjectID > 0 || *ProjectName != "" {
		var targetProject *CxSASTClientGo.Project
		if *ProjectID > 0 {
			var ok bool
			targetProject, ok = projectsById[*ProjectID]
			if !ok {
				logger.Fatalf("Unable to find project with ID %d", *ProjectID)
			}
		} else {
			for _, p := range projectsById {
				if strings.EqualFold(*ProjectName, p.Name) {
					targetProject = p
					break
				}
			}
			if targetProject == nil {
				logger.Fatalf("Unable to find project with name %v", *ProjectName)
			}
		}

		/*
			want to list out the query dependencies for this project
			-> all project-level custom queries + requirements
			-> all team-level custom queries + requirements
			-> all corp queries
		*/

		projectQueries := []uint64{}
		teamQueries := []uint64{}
		corpQueries := []uint64{}

		teamHierarchy := []uint64{}
		for t, ok := teamsById[targetProject.TeamID]; ok && t != nil; t = teamsById[t.ParentID] {
			teamHierarchy = append(teamHierarchy, t.TeamID)
		}

		for _, lang := range qc.QueryLanguages {
			for _, group := range lang.QueryGroups {
				for _, query := range group.Queries {
					if group.OwningProjectID == targetProject.ProjectID && !slices.Contains(projectQueries, query.QueryID) { // add all project queries for sure
						logger.Infof("Adding project %v query %v to list", targetProject.String(), query.StringDetailed())
						projectQueries = append(projectQueries, query.QueryID)
					} else if group.PackageType == CxSASTClientGo.TEAM_QUERY && slices.Contains(teamHierarchy, group.OwningTeamID) && !slices.Contains(teamQueries, query.QueryID) { // add all team-level queries in the hierarchy
						team := teamsById[group.OwningTeamID]
						teamQueries = append(teamQueries, query.QueryID)
						logger.Infof("Adding team %v query %v to list", team.String(), query.StringDetailed())
					} else if group.PackageType == CxSASTClientGo.CORP_QUERY && !*NoCorp { // add all corp queries
						corpQueries = append(corpQueries, query.QueryID)
						logger.Infof("Adding corp query %v to list", query.StringDetailed())
					}
				}
			}
		}

		logger.Infof("Processing query dependencies")

		for _, qid := range projectQueries {
			list := qc.OverrideList(qid)
			logger.Infof("Project query:\n\t%v", strings.Join(list, "\n\t"))
			query := qc.GetQueryByID(qid)
			if len(query.CustomDependencies) > 0 {
				deps := []string{}
				for _, dep := range query.CustomDependencies {
					qq := qc.GetQueryByID(dep)
					if qq.OwningGroup.PackageType != CxSASTClientGo.CORP_QUERY || !*NoCorp {
						prefix := "NEW"
						if qq.BaseQueryID != qq.QueryID {
							prefix = "Override"
						}
						deps = append(deps, fmt.Sprintf("%v: %v", prefix, qq.StringDetailed()))
					}
				}
				if len(deps) > 0 {
					logger.Warningf("%v depends on:\n\t- %v", query.StringDetailed(), strings.Join(deps, "\n\t- "))
				}
			}

			if len(query.Dependencies) > 0 && !*NoProd {
				deps := []string{}
				for _, dep := range query.Dependencies {
					qq := qc.GetQueryByID(dep)
					prefix := "NEW"
					if qq.BaseQueryID != qq.QueryID {
						prefix = "Override"
					}
					deps = append(deps, fmt.Sprintf("%v: %v", prefix, qq.StringDetailed()))
				}
				logger.Warningf("%v depends on:\n\t- %v", query.StringDetailed(), strings.Join(deps, "\n\t- "))
			}
		}

		for _, qid := range teamQueries {
			list := qc.OverrideList(qid)
			logger.Infof("Team query:\n\t%v", strings.Join(list, "\n\t"))
			query := qc.GetQueryByID(qid)
			if len(query.CustomDependencies) > 0 {
				deps := []string{}
				for _, dep := range query.CustomDependencies {
					qq := qc.GetQueryByID(dep)
					if qq.OwningGroup.PackageType != CxSASTClientGo.CORP_QUERY || !*NoCorp {
						prefix := "NEW"
						if qq.BaseQueryID != qq.QueryID {
							prefix = "Override"
						}
						deps = append(deps, fmt.Sprintf("%v: %v", prefix, qq.StringDetailed()))
					}
				}
				if len(deps) > 0 {
					logger.Warningf("%v depends on:\n\t- %v", query.StringDetailed(), strings.Join(deps, "\n\t- "))
				}
			}

			if len(query.Dependencies) > 0 && !*NoProd {
				deps := []string{}
				for _, dep := range query.Dependencies {
					qq := qc.GetQueryByID(dep)
					prefix := "NEW"
					if qq.BaseQueryID != qq.QueryID {
						prefix = "Override"
					}
					deps = append(deps, fmt.Sprintf("%v: %v", prefix, qq.StringDetailed()))
				}
				logger.Warningf("%v depends on:\n\t- %v", query.StringDetailed(), strings.Join(deps, "\n\t- "))
			}
		}

		if !*NoCorp {
			for _, qid := range corpQueries {
				list := qc.OverrideList(qid)
				logger.Infof("Corp query:\n\t%v", strings.Join(list, "\n\t"))
				query := qc.GetQueryByID(qid)
				if len(query.CustomDependencies) > 0 {
					deps := []string{}
					for _, dep := range query.CustomDependencies {
						qq := qc.GetQueryByID(dep)
						if qq.OwningGroup.PackageType != CxSASTClientGo.CORP_QUERY || !*NoCorp {
							prefix := "NEW"
							if qq.BaseQueryID != qq.QueryID {
								prefix = "Override"
							}
							deps = append(deps, fmt.Sprintf("%v: %v", prefix, qq.StringDetailed()))
						}
					}
					if len(deps) > 0 {
						logger.Warningf("%v depends on:\n\t- %v", query.StringDetailed(), strings.Join(deps, "\n\t- "))
					}
				}

				if len(query.Dependencies) > 0 && !*NoProd {
					deps := []string{}
					for _, dep := range query.Dependencies {
						qq := qc.GetQueryByID(dep)
						deps = append(deps, qq.StringDetailed())
					}
					logger.Warningf("%v depends on:\n\t- %v", query.StringDetailed(), strings.Join(deps, "\n\t- "))
				}
			}
		}
	} else {
		for _, lang := range qc.QueryLanguages {
			for _, group := range lang.QueryGroups {
				if (group.PackageType == CxSASTClientGo.CORP_QUERY && !*NoCorp) || (group.PackageType == CxSASTClientGo.TEAM_QUERY || group.PackageType == CxSASTClientGo.PROJECT_QUERY) {
					for _, query := range group.Queries {
						if query.IsValid && query.IsCustom() {
							if len(query.CustomDependencies) > 0 {
								deps := []string{}
								for _, dep := range query.CustomDependencies {
									qq := qc.GetQueryByID(dep)
									if qq.OwningGroup.PackageType != CxSASTClientGo.CORP_QUERY || !*NoCorp {
										prefix := "NEW"
										if qq.BaseQueryID != qq.QueryID {
											prefix = "Override"
										}
										deps = append(deps, fmt.Sprintf("%v: %v", prefix, qq.StringDetailed()))
									}
								}
								if len(deps) > 0 {
									logger.Infof("%v depends on:\n\t- %v", query.StringDetailed(), strings.Join(deps, "\n\t- "))
								}
							}

							if len(query.Dependencies) > 0 && !*NoProd {
								deps := []string{}
								for _, dep := range query.Dependencies {
									qq := qc.GetQueryByID(dep)
									prefix := "NEW"
									if qq.BaseQueryID != qq.QueryID {
										prefix = "Override"
									}
									deps = append(deps, fmt.Sprintf("%v: %v", prefix, qq.StringDetailed()))
								}
								if len(deps) > 0 {
									logger.Infof("%v depends on:\n\t- %v", query.StringDetailed(), strings.Join(deps, "\n\t- "))
								}
							}
						}
					}
				}
			}
		}
	}
}
