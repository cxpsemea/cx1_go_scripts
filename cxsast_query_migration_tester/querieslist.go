package main

import (
	"fmt"
	"slices"

	"github.com/cxpsemea/CxSASTClientGo"
)

type QueriesList struct {
	CorpQueriesToCreate     *[]*CxSASTClientGo.Query
	CorpQueriesToMigrate    *[]*CxSASTClientGo.Query
	TeamQueriesToMigrate    map[uint64][]*CxSASTClientGo.Query
	ProjectQueriesToMigrate map[uint64][]*CxSASTClientGo.Query
}

func NewQueriesList() QueriesList {
	return QueriesList{
		CorpQueriesToCreate:     &[]*CxSASTClientGo.Query{},
		CorpQueriesToMigrate:    &[]*CxSASTClientGo.Query{},
		TeamQueriesToMigrate:    make(map[uint64][]*CxSASTClientGo.Query),
		ProjectQueriesToMigrate: make(map[uint64][]*CxSASTClientGo.Query),
	}
}

func (ql *QueriesList) AppendCorp(query *CxSASTClientGo.Query, qc *CxSASTClientGo.QueryCollection) {
	if !slices.Contains(*ql.CorpQueriesToMigrate, query) {
		logger.Infof("Appending corp query to migrate: %v", query.StringDetailed())
		ql.CorpQueriesToMigrate = ql.appendQueryToList(query, qc, ql.CorpQueriesToMigrate)
	} else {
		logger.Tracef("Skip appending corp query to migrate: %v", query.StringDetailed())
	}
}
func (ql *QueriesList) AppendNewCorp(query *CxSASTClientGo.Query, qc *CxSASTClientGo.QueryCollection) {
	if !slices.Contains(*ql.CorpQueriesToCreate, query) {
		logger.Infof("Appending corp-base query to create: %v", query.StringDetailed())
		ql.CorpQueriesToCreate = ql.appendQueryToList(query, qc, ql.CorpQueriesToCreate)
	} else {
		logger.Tracef("Skip appending corp-base query to migrate: %v", query.StringDetailed())
	}
}
func (ql *QueriesList) AppendTeam(query *CxSASTClientGo.Query, teamId uint64, qc *CxSASTClientGo.QueryCollection) {
	logger.Infof("Appending team %d override query to migrate: %v", teamId, query.StringDetailed())

	if _, ok := ql.TeamQueriesToMigrate[teamId]; !ok {
		ql.TeamQueriesToMigrate[teamId] = make([]*CxSASTClientGo.Query, 0)
	}
	list := ql.TeamQueriesToMigrate[teamId]
	newList := ql.appendQueryToList(query, qc, &list)
	if &list != newList {
		ql.TeamQueriesToMigrate[teamId] = *newList
	}

	if query.BaseQueryID == query.QueryID {
		ql.AppendNewCorp(query, qc)
	}
}
func (ql *QueriesList) InsertTeam(query *CxSASTClientGo.Query, teamId uint64, qc *CxSASTClientGo.QueryCollection) {
	logger.Infof("Inserting team %d override query to migrate: %v", teamId, query.StringDetailed())

	if _, ok := ql.TeamQueriesToMigrate[teamId]; !ok {
		ql.TeamQueriesToMigrate[teamId] = make([]*CxSASTClientGo.Query, 0)
	}
	list := ql.TeamQueriesToMigrate[teamId]
	newList := insertQueryToList(query, qc, &list)
	if &list != newList {
		ql.TeamQueriesToMigrate[teamId] = *newList
	}

	if query.BaseQueryID == query.QueryID {
		ql.AppendNewCorp(query, qc)
	}
}

func (ql *QueriesList) AppendProject(query *CxSASTClientGo.Query, projectId uint64, qc *CxSASTClientGo.QueryCollection) {
	logger.Infof("Appending project %d override query to migrate: %v", projectId, query.StringDetailed())

	if _, ok := ql.ProjectQueriesToMigrate[projectId]; !ok {
		ql.ProjectQueriesToMigrate[projectId] = make([]*CxSASTClientGo.Query, 0)
	}
	list := ql.ProjectQueriesToMigrate[projectId]
	newList := ql.appendQueryToList(query, qc, &list)
	if &list != newList {
		ql.ProjectQueriesToMigrate[projectId] = *newList
	}

	if query.BaseQueryID == query.QueryID {
		ql.AppendNewCorp(query, qc)
	}
}

func (ql *QueriesList) FixGroups(qc *CxSASTClientGo.QueryCollection) error {
	for _, q := range *ql.CorpQueriesToCreate {
		qq := qc.GetQueryByID(q.QueryID)
		if qq == nil {
			return fmt.Errorf("failed to find query %v: query collection may have changed since queries.json was generated, please regenerate queries.json", q.String())
		}
		q.OwningGroup = qq.OwningGroup
	}
	for _, q := range *ql.CorpQueriesToMigrate {
		qq := qc.GetQueryByID(q.QueryID)
		if qq == nil {
			return fmt.Errorf("failed to find query %v: query collection may have changed since queries.json was generated, please regenerate queries.json", q.String())
		}
		q.OwningGroup = qq.OwningGroup
	}
	for teamId := range ql.TeamQueriesToMigrate {
		for _, q := range ql.TeamQueriesToMigrate[teamId] {
			qq := qc.GetQueryByID(q.QueryID)
			if qq == nil {
				return fmt.Errorf("failed to find query %v: query collection may have changed since queries.json was generated, please regenerate queries.json", q.String())
			}
			q.OwningGroup = qq.OwningGroup
		}
	}
	for projId := range ql.ProjectQueriesToMigrate {
		for _, q := range ql.ProjectQueriesToMigrate[projId] {
			qq := qc.GetQueryByID(q.QueryID)
			if qq == nil {
				return fmt.Errorf("failed to find query %v: query collection may have changed since queries.json was generated, please regenerate queries.json", q.String())
			}
			q.OwningGroup = qq.OwningGroup
		}
	}
	return nil
}

func (ql *QueriesList) appendQueryToList(query *CxSASTClientGo.Query, qc *CxSASTClientGo.QueryCollection, list *[]*CxSASTClientGo.Query) *[]*CxSASTClientGo.Query {
	newList := list
	if query.IsValid && query.IsCustom() {
		if !slices.Contains(*list, query) {
			logger.Debug(" - appended query")
			l := append(*list, query)

			newList = &l

			for _, qid := range query.Dependencies {
				qq := qc.GetQueryByID(qid)
				if !slices.Contains(*newList, qq) {
					if (qq.OwningGroup.OwningProjectID > 0 && qq.OwningGroup.OwningProjectID == query.OwningGroup.OwningProjectID) || (qq.OwningGroup.OwningTeamID > 0 && qq.OwningGroup.OwningTeamID == query.OwningGroup.OwningTeamID) {
						newList = insertQueryToList(qq, qc, newList)
					} else if qq.OwningGroup.PackageType == CxSASTClientGo.CORP_QUERY {
						ql.AppendCorp(qq, qc)
					}
				}
			}

		} else {
			logger.Debug(" - already in list")
		}
	}

	return newList
}

func insertQueryToList(query *CxSASTClientGo.Query, qc *CxSASTClientGo.QueryCollection, list *[]*CxSASTClientGo.Query) *[]*CxSASTClientGo.Query {
	newList := list
	if query.IsValid && query.IsCustom() {
		if !slices.Contains(*newList, query) {
			logger.Debugf(" - inserted query for dependency: %v", query.StringDetailed())

			list := append([]*CxSASTClientGo.Query{query}, *newList...)
			newList = &list

			for _, qid := range query.Dependencies {
				qq := qc.GetQueryByID(qid)
				if ((qq.OwningGroup.OwningProjectID == query.OwningGroup.OwningProjectID) || (qq.OwningGroup.OwningTeamID > 0)) && !slices.Contains(*newList, qq) {
					newList = insertQueryToList(qq, qc, newList)
				}
			}
		} else {
			logger.Debug(" - already in list")
		}
	}
	return newList
}
