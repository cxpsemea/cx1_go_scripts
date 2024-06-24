package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/cxpsemea/CxSASTClientGo"
	"github.com/sirupsen/logrus"
)

/*
does the actual process of identifying the projects/teams with queries to migrate and tracking the result
*/

type MigratedQuery struct {
	Cxsq *CxSASTClientGo.Query
	Cx1q *Cx1ClientGo.Query
	Err  error
}

var QueryMigrationStatus map[uint64]MigratedQuery = make(map[uint64]MigratedQuery)

func DoProcess(cx1client *Cx1ClientGo.Cx1Client, qc *CxSASTClientGo.QueryCollection, teamsById *map[uint64]*CxSASTClientGo.Team, projectsById *map[uint64]*CxSASTClientGo.Project) {
	InitializeQueryMigration(cx1client)

	corpQueriesToMigrate := []uint64{}
	corpQueriesToCreate := []uint64{}
	teamQueriesToMigrate := make(map[uint64][]uint64)
	projectQueriesToMigrate := make(map[uint64][]uint64)
	projectsPerTeam := make(map[uint64][]uint64)

	for _, project := range *projectsById {
		if _, ok := projectsPerTeam[project.TeamID]; !ok {
			projectsPerTeam[project.TeamID] = []uint64{}
		}
		projectsPerTeam[project.TeamID] = append(projectsPerTeam[project.TeamID], project.ProjectID)
	}

	for lid := range qc.QueryLanguages {
		for gid := range qc.QueryLanguages[lid].QueryGroups {
			switch qc.QueryLanguages[lid].QueryGroups[gid].PackageType {
			case CxSASTClientGo.CORP_QUERY:
				for _, query := range qc.QueryLanguages[lid].QueryGroups[gid].Queries {
					logger.Infof("Appending corp query to migrate: %v", query.StringDetailed())
					corpQueriesToMigrate = appendQueryToList(&query, qc, corpQueriesToMigrate, logger)

				}
			case CxSASTClientGo.TEAM_QUERY:
				for _, query := range qc.QueryLanguages[lid].QueryGroups[gid].Queries {
					if query.IsValid {
						if query.BaseQueryID == query.QueryID { // doesn't inherit from anything
							logger.Infof("Appending team-override corp-base query to create: %v", query.StringDetailed())
							corpQueriesToCreate = appendQueryToList(&query, qc, corpQueriesToCreate, logger)

						} else {
							teamId := qc.QueryLanguages[lid].QueryGroups[gid].OwningTeamID
							if _, ok := teamQueriesToMigrate[teamId]; !ok {
								teamQueriesToMigrate[teamId] = make([]uint64, 0)
							}
							logger.Infof("Appending team-override query to migrate: %v", query.StringDetailed())
							teamQueriesToMigrate[teamId] = appendQueryToList(&query, qc, teamQueriesToMigrate[teamId], logger)
						}
					}
				}
			case CxSASTClientGo.PROJECT_QUERY:
				for _, query := range qc.QueryLanguages[lid].QueryGroups[gid].Queries {
					if query.IsValid {
						if query.BaseQueryID == query.QueryID { // doesn't inherit from anything
							logger.Infof("Appending project-override corp-base query to create: %v", query.StringDetailed())
							corpQueriesToCreate = appendQueryToList(&query, qc, corpQueriesToCreate, logger)
						} else {
							projectId := qc.QueryLanguages[lid].QueryGroups[gid].OwningProjectID
							if _, ok := projectQueriesToMigrate[projectId]; !ok {
								projectQueriesToMigrate[projectId] = make([]uint64, 0)
							}
							logger.Infof("Appending project-override query to migrate: %v", query.StringDetailed())
							projectQueriesToMigrate[projectId] = appendQueryToList(&query, qc, projectQueriesToMigrate[projectId], logger)
						}
					}
				}
			}
		}
	}

	// now migrate

	for _, queryId := range corpQueriesToCreate {
		sastq := qc.GetQueryByID(queryId)
		cx1q, err := MigrateCorpQuery(cx1client, qc, sastq, logger)
		if err != nil {
			logger.Errorf("Failed to create empty new corp query prerequisite for %v", sastq.StringDetailed())
		} else {
			logger.Infof("Successfully migrated %v to %v", sastq.StringDetailed(), cx1q.StringDetailed())
		}
		QueryMigrationStatus[sastq.QueryID] = MigratedQuery{sastq, cx1q, err}
	}

	for _, queryId := range corpQueriesToMigrate {
		sastq := qc.GetQueryByID(queryId)
		cx1q, err := MigrateCorpQuery(cx1client, qc, sastq, logger)
		if err != nil {
			logger.Errorf("Failed to migrate corp query %v", sastq.StringDetailed())
		} else {
			logger.Infof("Successfully migrated %v to %v", sastq.StringDetailed(), cx1q.StringDetailed())
		}
		QueryMigrationStatus[sastq.QueryID] = MigratedQuery{sastq, cx1q, err}
	}

	for teamId := range teamQueriesToMigrate {
		if _, ok := projectsPerTeam[teamId]; ok && len(projectsPerTeam[teamId]) > 0 {
			// team has projects, so needs to have queries migrated
			for _, qid := range teamQueriesToMigrate[teamId] {
				query := qc.GetQueryByID(qid)
				baseQuery, err := getCx1RootQuery(cx1client, qc, query)
				var cx1q *Cx1ClientGo.Query
				if err != nil {
					logger.Errorf("Unable to migrate query %v: failed to get cx1 root query: %s", query.StringDetailed(), err)
				} else {
					var newMergedQuery CxSASTClientGo.Query
					newMergedQuery, err = makeMergedQuery(qc, query, baseQuery, teamsById, logger)
					if err != nil {
						logger.Errorf("Unable to migrate query %v: failed to make merged query: %s", query.StringDetailed(), err)
					} else {
						cx1q, err = MigrateTeamQuery(cx1client, qc, &newMergedQuery, logger)
						if err != nil {
							logger.Errorf("Failed to migrate query %v: %s", query.StringDetailed(), err)
						} else {
							logger.Infof("Successfully migrated %v to %v", query.StringDetailed(), cx1q.StringDetailed())
						}
					}
				}
				QueryMigrationStatus[qid] = MigratedQuery{
					Cxsq: query,
					Cx1q: cx1q,
					Err:  err,
				}
			}
		}
	}

	/// afterwards output the status

	keys := make([]string, 0, len(QueryMigrationStatus))
	keymap := make(map[string]uint64)
	for id, qms := range QueryMigrationStatus {
		nn := fmt.Sprintf("%v.%v.%v #%d", qms.Cxsq.Language, qms.Cxsq.Group, qms.Cxsq.Name, qms.Cxsq.QueryID)
		keys = append(keys, nn)
		keymap[nn] = id
	}

	slices.Sort(keys)

	logger.Info("Migration result:")
	for _, key := range keys {
		qms := QueryMigrationStatus[keymap[key]]
		if qms.Err != nil {
			logger.Errorf("ERR: %v -> %s", qms.Cxsq.StringDetailed(), qms.Err)
		} else {
			logger.Infof("OK: %v -> %v", qms.Cxsq.StringDetailed(), qms.Cx1q.StringDetailed())
		}
	}
}

func appendQueryToList(query *CxSASTClientGo.Query, qc *CxSASTClientGo.QueryCollection, list []uint64, logger *logrus.Logger) []uint64 {
	newList := list
	if query.IsValid && query.IsCustom() && !slices.Contains(newList, query.QueryID) {
		logger.Debug(" - appended query")
		newList = append(newList, query.QueryID)
		for _, qid := range query.Dependencies {
			qq := qc.GetQueryByID(qid)
			if ((qq.OwningGroup.OwningProjectID > 0 && qq.OwningGroup.OwningProjectID == query.OwningGroup.OwningProjectID) || (qq.OwningGroup.OwningTeamID > 0 && qq.OwningGroup.OwningTeamID == query.OwningGroup.OwningTeamID)) && !slices.Contains(newList, qid) {
				newList = prependQueryToList(qq, qc, newList, logger)
			}
		}
	} else {
		logger.Debug(" - already in list")
	}

	return newList
}

func prependQueryToList(query *CxSASTClientGo.Query, qc *CxSASTClientGo.QueryCollection, list []uint64, logger *logrus.Logger) []uint64 {
	newList := list
	if query.IsValid && query.IsCustom() {
		logger.Debugf(" - inserted query for dependency: %v", query.StringDetailed())

		newList = append([]uint64{query.QueryID}, newList...)
		for _, qid := range query.Dependencies {
			qq := qc.GetQueryByID(qid)
			if ((qq.OwningGroup.OwningProjectID == query.OwningGroup.OwningProjectID) || (qq.OwningGroup.OwningTeamID > 0)) && !slices.Contains(newList, qid) {
				newList = prependQueryToList(qq, qc, newList, logger)
			}
		}
	}
	return newList
}

func makeMergedQuery(qc *CxSASTClientGo.QueryCollection, source *CxSASTClientGo.Query, dest *Cx1ClientGo.Query, teamsById *map[uint64]*CxSASTClientGo.Team, logger *logrus.Logger) (CxSASTClientGo.Query, error) {
	merger := QueryMerger{}

	var owner string
	q := source

	mergeString := []string{}

	for {
		team, ok := (*teamsById)[q.OwningGroup.OwningTeamID]
		if !ok {
			break
		}
		owner = team.String()
		merger.Insert(q, owner)
		mergeString = append(mergeString, fmt.Sprintf(" - %v", q.StringDetailed()))

		if q.BaseQueryID != q.QueryID {
			q = qc.GetQueryByID(q.BaseQueryID)
			if q.OwningGroup.PackageType != CxSASTClientGo.TEAM_QUERY {
				break
			}
		}
	}

	code, err := merger.Merge(dest.Name)
	if err != nil {
		return *source, err
	}
	newQuery := *source
	newQuery.Source = code

	logger.Debugf("Created merged query for %v:\n%v", q.StringDetailed(), strings.Join(mergeString, "\n"))
	logger.Debugf("Generated code:\n%v", code)

	return newQuery, nil

}
