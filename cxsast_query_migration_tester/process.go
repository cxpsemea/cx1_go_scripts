package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cxpsemea/Cx1ClientGo"
	"github.com/cxpsemea/CxSASTClientGo"
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

func GenerateMigrationList(cx1client *Cx1ClientGo.Cx1Client, qc *CxSASTClientGo.QueryCollection, teamsById *map[uint64]*CxSASTClientGo.Team, projectsById *map[uint64]*CxSASTClientGo.Project) QueriesList {
	queriesList := NewQueriesList()

	/*corpQueriesToMigrate := []uint64{}
	corpQueriesToCreate := []uint64{}
	teamQueriesToMigrate := make(map[uint64][]uint64)
	projectQueriesToMigrate := make(map[uint64][]uint64)*/

	for lid := range qc.QueryLanguages {
		for gid := range qc.QueryLanguages[lid].QueryGroups {
			logger.Tracef("Processing query group %v", qc.QueryLanguages[lid].QueryGroups[gid].String())
			switch qc.QueryLanguages[lid].QueryGroups[gid].PackageType {
			case CxSASTClientGo.CORP_QUERY:
				for _, query := range qc.QueryLanguages[lid].QueryGroups[gid].Queries {
					queriesList.AppendCorp(&query, qc)
					//queriesList.CorpQueriesToMigrate = appendQueryToList(&query, qc, corpQueriesToMigrate)

				}
			case CxSASTClientGo.TEAM_QUERY:
				//qg := &qc.QueryLanguages[lid].QueryGroups[gid]

				for _, query := range qc.QueryLanguages[lid].QueryGroups[gid].Queries {
					if query.IsValid {
						destName := ""
						if query.BaseQueryID == query.QueryID { // team override doesn't inherit from anything, need to create a corp-level base in Cx1
							queriesList.AppendNewCorp(&query, qc)
						} else {
							baseQuery, err := getCx1RootQuery(cx1client, qc, &query)
							if err != nil {
								logger.Errorf("Unable to migrate query %v: failed to get cx1 root query: %s", query.StringDetailed(), err)
								continue
							}
							destName = baseQuery.Name
						}

						teamId := qc.QueryLanguages[lid].QueryGroups[gid].OwningTeamID
						newMergedQuery, err := makeMergedQuery(qc, &query, destName, teamsById)
						if err != nil {
							logger.Errorf("Unable to migrate query %v: failed to make merged query: %s", query.StringDetailed(), err)
						} else {
							queriesList.AppendTeam(&newMergedQuery, teamId, qc)
						}
					}
				}

			case CxSASTClientGo.PROJECT_QUERY:
				for _, query := range qc.QueryLanguages[lid].QueryGroups[gid].Queries {
					if query.IsValid {
						if query.BaseQueryID == query.QueryID { // project override doesn't inherit from anything, need to create a corp-level base in Cx1
							queriesList.AppendNewCorp(&query, qc)
						}
						projectId := qc.QueryLanguages[lid].QueryGroups[gid].OwningProjectID
						queriesList.AppendProject(&query, projectId, qc)
					}
				}
			}
		}
	}

	logger.Trace("Checking in-scope queries from parent teams")
	// For team-query-migrations, need to add the parent-team queries which are outside of the current-team-query dependencies
	// eg: team1 has Stored_XSS
	//     team1\team2 has sql_injection
	// Migrating team2 should include the team1 Stored_XSS
	for teamId := range queriesList.TeamQueriesToMigrate {
		team := (*teamsById)[teamId]
		logger.Debugf("Checking team %v parent teams for in-scope queries", team.String())
		dependencies := []uint64{}

		// first list out the queries we already need to satisfy this team's dependencies
		for _, query := range queriesList.TeamQueriesToMigrate[teamId] {
			for qq := query; qq != nil && qq.BaseQueryID != qq.QueryID; qq = qc.GetQueryByID(qq.BaseQueryID) {
				if qq.OwningGroup.PackageType != CxSASTClientGo.TEAM_QUERY {
					break
				}
				if !slices.Contains(dependencies, qq.QueryID) {
					dependencies = append(dependencies, qq.QueryID)

					for _, depId := range qq.CustomDependencies {
						if !slices.Contains(dependencies, depId) {
							dependencies = append(dependencies, depId)
						}
					}
				}
			}
		}

		// next find the queries in the parent team(s) that are not already counted as a dependency
		teamPath := makeTeamHierarchy(teamId, teamsById)
		for _, parentTeamId := range teamPath {
			pteam := (*teamsById)[parentTeamId]
			logger.Debugf("Checking team %v's parent team %v queries", team.String(), pteam.String())
			for _, query := range queriesList.TeamQueriesToMigrate[parentTeamId] {
				for qq := query; qq != nil && qq.BaseQueryID != qq.QueryID; qq = qc.GetQueryByID(qq.BaseQueryID) {
					if qq.OwningGroup.PackageType != CxSASTClientGo.TEAM_QUERY {
						break
					}
					if !slices.Contains(dependencies, qq.QueryID) {
						dependencies = append(dependencies, qq.QueryID)
						queriesList.InsertTeam(qq, teamId, qc)

						for _, depId := range qq.CustomDependencies {
							if !slices.Contains(dependencies, depId) {
								dependencies = append(dependencies, depId)
							}
						}
					}
				}
			}
		}
	}

	return queriesList
}

func MigrateQueries(cx1client *Cx1ClientGo.Cx1Client, qc *CxSASTClientGo.QueryCollection, teamsById *map[uint64]*CxSASTClientGo.Team, projectsById *map[uint64]*CxSASTClientGo.Project, projectsPerTeam *map[uint64][]uint64, queriesList QueriesList) {
	for _, sastq := range *queriesList.CorpQueriesToCreate {
		//sastq := qc.GetQueryByID(queryId)
		cx1q, err := MigrateEmptyCorpQuery(cx1client, qc, sastq)
		if err != nil {
			logger.Errorf("Failed to create new corp query prerequisite for %v", sastq.StringDetailed())
		} else {
			logger.Infof("Successfully migrated corp query %v to %v", sastq.StringDetailed(), cx1q.StringDetailed())
		}
		QueryMigrationStatus[sastq.QueryID] = MigratedQuery{sastq, cx1q, err}
	}

	for _, sastq := range *queriesList.CorpQueriesToMigrate {
		cx1q, err := MigrateCorpQuery(cx1client, qc, sastq)
		if err != nil {
			logger.Errorf("Failed to migrate corp query %v", sastq.StringDetailed())
		} else {
			logger.Infof("Successfully migrated corp query %v to %v", sastq.StringDetailed(), cx1q.StringDetailed())
		}
		QueryMigrationStatus[sastq.QueryID] = MigratedQuery{sastq, cx1q, err}
	}

	for teamId := range queriesList.TeamQueriesToMigrate {
		if _, ok := (*projectsPerTeam)[teamId]; ok && len((*projectsPerTeam)[teamId]) > 0 {
			for _, query := range queriesList.TeamQueriesToMigrate[teamId] {
				cx1q, err := MigrateTeamQuery(cx1client, qc, query)
				if err != nil {
					logger.Errorf("Failed to migrate team query %v: %s", query.StringDetailed(), err)
				} else {
					logger.Infof("Successfully migrated team query %v to %v", query.StringDetailed(), cx1q.StringDetailed())
				}
				QueryMigrationStatus[query.QueryID] = MigratedQuery{query, cx1q, err}
			}
		}
	}

	for projectId := range queriesList.ProjectQueriesToMigrate {
		for _, query := range queriesList.ProjectQueriesToMigrate[projectId] {
			cx1q, err := MigrateProjectQuery(cx1client, qc, query)
			if err != nil {
				logger.Errorf("Failed to migrate project query %v: %s", query.StringDetailed(), err)
			} else {
				logger.Infof("Successfully migrated project query %v to %v", query.StringDetailed(), cx1q.StringDetailed())
			}
			QueryMigrationStatus[query.QueryID] = MigratedQuery{query, cx1q, err}
		}
	}
}

func MigrateTeamQueries(cx1client *Cx1ClientGo.Cx1Client, qc *CxSASTClientGo.QueryCollection, teamsById *map[uint64]*CxSASTClientGo.Team, projectsById *map[uint64]*CxSASTClientGo.Project, queriesList QueriesList, teamId uint64) {
	for _, query := range queriesList.TeamQueriesToMigrate[teamId] {
		cx1q, err := MigrateTeamQuery(cx1client, qc, query)
		if err != nil {
			logger.Errorf("Failed to migrate team query %v: %s", query.StringDetailed(), err)
		} else {
			logger.Infof("Successfully migrated team query %v to %v", query.StringDetailed(), cx1q.StringDetailed())
		}
		QueryMigrationStatus[query.QueryID] = MigratedQuery{query, cx1q, err}
	}
}

func MigrateProjectQueries(cx1client *Cx1ClientGo.Cx1Client, qc *CxSASTClientGo.QueryCollection, teamsById *map[uint64]*CxSASTClientGo.Team, projectsById *map[uint64]*CxSASTClientGo.Project, queriesList QueriesList, projectId uint64) {
	for _, query := range queriesList.ProjectQueriesToMigrate[projectId] {
		cx1q, err := MigrateProjectQuery(cx1client, qc, query)
		if err != nil {
			logger.Errorf("Failed to migrate project query %v: %s", query.StringDetailed(), err)
		} else {
			logger.Infof("Successfully migrated project query %v to %v", query.StringDetailed(), cx1q.StringDetailed())
		}
		QueryMigrationStatus[query.QueryID] = MigratedQuery{query, cx1q, err}
	}
}

func MigrateQuery(cx1client *Cx1ClientGo.Cx1Client, qc *CxSASTClientGo.QueryCollection, teamsById *map[uint64]*CxSASTClientGo.Team, projectsById *map[uint64]*CxSASTClientGo.Project, queriesList QueriesList, queryId uint64) {

	for _, sastq := range *queriesList.CorpQueriesToCreate {
		if sastq.QueryID == queryId {
			cx1q, err := MigrateEmptyCorpQuery(cx1client, qc, sastq)
			if err != nil {
				logger.Errorf("Failed to create new corp query prerequisite for %v", sastq.StringDetailed())
			} else {
				logger.Infof("Successfully migrated corp query %v to %v", sastq.StringDetailed(), cx1q.StringDetailed())
			}
			QueryMigrationStatus[sastq.QueryID] = MigratedQuery{sastq, cx1q, err}
		}
	}

	for _, sastq := range *queriesList.CorpQueriesToMigrate {
		if sastq.QueryID == queryId {
			cx1q, err := MigrateCorpQuery(cx1client, qc, sastq)
			if err != nil {
				logger.Errorf("Failed to migrate corp query %v", sastq.StringDetailed())
			} else {
				logger.Infof("Successfully migrated corp query %v to %v", sastq.StringDetailed(), cx1q.StringDetailed())
			}
			QueryMigrationStatus[sastq.QueryID] = MigratedQuery{sastq, cx1q, err}
		}
	}

	for teamId := range queriesList.TeamQueriesToMigrate {
		for _, query := range queriesList.TeamQueriesToMigrate[teamId] {
			if query.QueryID == queryId {
				cx1q, err := MigrateTeamQuery(cx1client, qc, query)
				if err != nil {
					logger.Errorf("Failed to migrate team query %v: %s", query.StringDetailed(), err)
				} else {
					logger.Infof("Successfully migrated team query %v to %v", query.StringDetailed(), cx1q.StringDetailed())
				}
				QueryMigrationStatus[query.QueryID] = MigratedQuery{query, cx1q, err}
			}
		}
	}

	for projectId := range queriesList.ProjectQueriesToMigrate {
		for _, query := range queriesList.ProjectQueriesToMigrate[projectId] {
			if query.QueryID == queryId {
				cx1q, err := MigrateProjectQuery(cx1client, qc, query)
				if err != nil {
					logger.Errorf("Failed to migrate project query %v: %s", query.StringDetailed(), err)
				} else {
					logger.Infof("Successfully migrated project query %v to %v", query.StringDetailed(), cx1q.StringDetailed())
				}
				QueryMigrationStatus[query.QueryID] = MigratedQuery{query, cx1q, err}
			}
		}
	}
}

func Summary() {
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

func makeMergedQuery(qc *CxSASTClientGo.QueryCollection, source *CxSASTClientGo.Query, destName string, teamsById *map[uint64]*CxSASTClientGo.Team) (CxSASTClientGo.Query, error) {
	merger := QueryMerger{}
	logger.Tracef("Creating merged query for %v", source.StringDetailed())

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
		} else {
			break
		}
	}

	if len(mergeString) <= 1 {
		logger.Tracef("No queries to merge, returning original")
		return *source, nil
	}

	code, err := merger.Merge(destName)
	if err != nil {
		return *source, err
	}
	newQuery := *source
	newQuery.Source = code

	logger.Tracef("Created merged query for %v:\n%v", q.StringDetailed(), strings.Join(mergeString, "\n"))
	logger.Tracef("Generated code:\n%v", code)

	return newQuery, nil

}

func makeTeamHierarchy(teamId uint64, teamsById *map[uint64]*CxSASTClientGo.Team) []uint64 {
	ret := []uint64{}

	for team := (*teamsById)[teamId]; team != nil && team.ParentID > 0; team = (*teamsById)[team.ParentID] {
		ret = append(ret, team.ParentID)
	}
	return ret
}
