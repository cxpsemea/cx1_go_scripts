package main

// from https://github.com/cxpsemea/go-querymerger/blob/main/cxqlcodemerger/cxqlcodemerger.go

// import "fmt"
import (
	"bufio"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cxpsemea/CxSASTClientGo"
)

const STATUS_OK = 0      // All good
const STATUS_REMERGE = 1 // Detected query code merged before, this is dangerous and must be reported or logged
// const STATUS_BROKEN_CHAIN = 1		// Detected broken chains (base.<x> not called), fixed
// const STATUS_ASSIGNMENT = 2			// Detected base assignments not to result (xvar = base.<x>), fixed
const STATUS_EMPTY = 8   // No queries to process
const STATUS_INVALID = 9 // Invalid content for processing

type CxQuery struct {
	SourceCode     string
	QueryId        uint64
	QueryName      string
	Language       string
	PackageId      uint64
	PackageName    string
	Severity       int
	Level          string
	TeamOrProjId   uint64
	TeamOrProjName string
	// For internal use
	tag       string
	callsbase bool
	issafe    bool
}

type QueryMerger []CxQuery

// Gets a list of queries
func Merger() *QueryMerger {
	return &QueryMerger{}
}

// Append a new query code to the end of the queries list to be processed
func (q *QueryMerger) Add(query *CxSASTClientGo.Query, owner string) {
	// Compose the query element
	var owningId uint64
	if query.OwningGroup.OwningProjectID > 0 {
		owningId = query.OwningGroup.OwningProjectID
	} else {
		owningId = query.OwningGroup.OwningTeamID
	}
	var qqry = q.constructqueryrecord(query.Source, query.QueryID, query.Name, query.Language, query.OwningGroup.PackageID, query.Group, query.Severity, query.OwningGroup.PackageType, owningId, owner)

	// Add the query to the list
	*q = append(*q, qqry)
}

// Insert a new query code at the top of the  queries list to be processed
func (q *QueryMerger) Insert(query *CxSASTClientGo.Query, owner string) {
	// Compose the query element
	var owningId uint64
	if query.OwningGroup.OwningProjectID > 0 {
		owningId = query.OwningGroup.OwningProjectID
	} else {
		owningId = query.OwningGroup.OwningTeamID
	}
	var qqry = q.constructqueryrecord(query.Source, query.QueryID, query.Name, query.Language, query.OwningGroup.PackageID, query.Group, query.Severity, query.OwningGroup.PackageType, owningId, owner)

	// Insert at the top od the list
	*q = append([]CxQuery{qqry}, *q...)
}

// Clears the queries list (slice), to be ready for the next processing
func (q *QueryMerger) Clear() {
	if len(*q) > 0 {
		*q = nil
	}
}

// Count the number of queries in queries list
func (q *QueryMerger) Count() int {
	return len(*q)
}

// Removes the last query on the list
func (q *QueryMerger) Delete() {
	if q.Count() > 0 {
		*q = QueryMerger(*q)[:q.Count()-1]
	}
}

// Retrieves a query object from list given its index
func (q *QueryMerger) Query(index int) (xquery CxQuery, err error) {
	if index < 0 || index > q.Count()-1 {
		return CxQuery{}, errors.New("Query index out of bounds")
	}
	return QueryMerger(*q)[index], nil
}

// Gets the last ververity or the highest severity from the queries list
func (q *QueryMerger) Severity(highest bool) int {
	var severity = 0
	var querieslist QueryMerger = *q
	for i := range querieslist {
		qry := querieslist[i]
		if (!highest) || (qry.Severity > severity) {
			severity = qry.Severity
		}
	}
	return severity
}

// Initialize CxQuery
func (q *QueryMerger) constructqueryrecord(sourcecode string, queryid uint64, queryname string, language string, packageid uint64, packagename string, severity int, level string, teamorprojid uint64, teamorprojname string) CxQuery {
	// Ensure same query name and language are the same
	var qtimestamp time.Time = time.Now()
	// Trim strings (never know what's comming)
	var qqueryname = strings.TrimSpace(queryname)
	var qlanguage = strings.TrimSpace(language)
	var qpackagename = strings.TrimSpace(packagename)
	var qteamorprojname = strings.TrimSpace(teamorprojname)
	// Severity
	var qseverity = ""
	switch severity {
	case 0:
		qseverity = "0 - Info"
	case 1:
		qseverity = "1 - Low"
	case 2:
		qseverity = "2 - Medium"
	case 3:
		qseverity = "3 - High"
	default:
		qseverity = "Invalid (" + strconv.Itoa(severity) + ")"
	}
	// Compose the tag
	var qtag = "// ======================================================\n"
	switch level {
	case CxSASTClientGo.CORP_QUERY:
		qtag = qtag + "// MERGED - CORPORATE LEVEL\n"
	case CxSASTClientGo.TEAM_QUERY:
		qtag = qtag + "// MERGED - TEAM LEVEL\n"
		qtag = qtag + "// TEAM: " + strconv.FormatUint(teamorprojid, 10) + " - " + qteamorprojname + "\n"
	case CxSASTClientGo.PROJECT_QUERY:
		qtag = qtag + "// MERGED - PROJECT LEVEL\n"
		qtag = qtag + "// PROJECT: " + strconv.FormatUint(teamorprojid, 10) + " - " + qteamorprojname + "\n"
	}
	qtag = qtag + "// QUERY: " + strconv.FormatUint(queryid, 10) + " - " + qqueryname + "\n"
	qtag = qtag + "// LANGUAGE: " + qlanguage + "\n"
	qtag = qtag + "// PACKAGE: " + strconv.FormatUint(packageid, 10) + " - " + qpackagename + "\n"
	qtag = qtag + "// SEVERITY: " + qseverity + "\n"
	qtag = qtag + "// TIMESTAMP: " + qtimestamp.Format("01-02-2006 15:04:05.000000") + "\n"
	qtag = qtag + "// ======================================================\n"
	var qqry = CxQuery{sourcecode, queryid, qqueryname, qlanguage, packageid, qpackagename, severity, level, teamorprojid, qteamorprojname, qtag, false, true}
	return qqry
}

// Find word in an array of words, private
func wordindex(vs []string, t string, prefixed bool) int {
	for i, v := range vs {
		if prefixed {
			if strings.HasPrefix(v, t) {
				return i
			}
		} else {
			if v == t {
				return i
			}
		}
	}
	return -1
}

// Indent the code, and comment it if needed, private
func arrangecode(sourcecode string, commented bool) string {
	var xcode string = ""
	var xline string
	var xorig string
	xorig = strings.TrimSpace(sourcecode)
	scanner := bufio.NewScanner(strings.NewReader(xorig))
	for scanner.Scan() {
		xline = scanner.Text()
		if commented {
			xcode = xcode + "\t//NO-BASE//\t " + xline + "\n"
		} else {
			xcode = xcode + "\t" + xline + "\n"
		}
	}
	return xcode
}

// Get code without comments, private
func uncommentedcode(sourcecode string) string {
	// C style comments
	var ccmt1 = regexp.MustCompile(`/\*([^*]|[\r\n]|(\*+([^*/]|[\r\n])))*\*+/`)
	// C++ style comments
	var ccmt2 = regexp.MustCompile(`//.*`)
	// Holder
	var rawbytes []byte
	rawbytes = ccmt1.ReplaceAll([]byte(sourcecode), []byte(""))
	rawbytes = ccmt2.ReplaceAll(rawbytes, []byte(""))
	return string(rawbytes)
}

// Check if the code calls base.<queryname> in a recongized way, private
func codecallsbase(sourcecode string, qname string) (qcallsbase bool, qissafe bool) {
	var callsbase bool = false
	var issafe bool = true
	var uncommented = uncommentedcode(sourcecode)
	// Check base invocation
	var thebase = "base." + qname + "()"
	callsbase = strings.Contains(uncommented, thebase)
	// Check for assignments that will indicate danger suggesting code needs fix or manual review
	// Such as assignement to a variable that is not "result"
	// The patterns recognized as VALID are:
	//   result = base.<queryname>
	//   result= base.<queryname>
	//   result =base.<queryname>
	//   result=base.<queryname>
	if callsbase {
		var found bool = strings.Contains(uncommented, "result="+thebase)
		if !found {
			var pos = strings.Index(uncommented, thebase) + len(thebase) + 10
			if pos >= len(uncommented) {
				pos = len(uncommented) - 1
			}
			var xtext = uncommented[:pos]
			// Remove unwanted escaped chars
			xtext = strings.ReplaceAll(xtext, "\n", " ")
			xtext = strings.ReplaceAll(xtext, "\r", " ")
			xtext = strings.ReplaceAll(xtext, "\t", " ")
			xtext = strings.ReplaceAll(xtext, "\b", " ")
			var words = strings.Split(xtext, " ")
			pos = wordindex(words, thebase, true)
			// Check for direct assignment
			found = ((pos >= 2) && (words[pos-2] == "result") && (words[pos-1] == "=")) || ((pos >= 1) && (words[pos-1] == "result="))
			// Last chance
			if !found {
				pos = 0
				for {
					if pos > len(words)-2 {
						break
					}
					if words[pos] == "result" && strings.HasPrefix(words[pos-1], "="+thebase) {
						found = true
						break
					}
					pos++
				}
			}
		}
		if !found {
			issafe = false
		}
	}
	return callsbase, issafe
}

// Merge the all queries in list into a single query
func (q *QueryMerger) merge_queries(destqueryname string) (qquerycode string, qstatus int, qstatusmessage string) {
	var status int = STATUS_OK
	var statusmessage string = ""
	var result string = ""
	var querycount int = 0
	var querieslist QueryMerger = *q
	var index int
	var firstindex int = 0
	var extratag string
	var querycode string
	var queryinject string
	var sbase string
	var xbase string
	var xresult string
	var xcounter int
	var xqueryname string
	var xdestqueryname string

	// Precheck status
	status, statusmessage = q.validate_queries_struct()

	// Check we have some queries
	if len(*q) == 0 {
		result = ""
		status = STATUS_EMPTY
		return result, status, statusmessage
	}

	// Get query name from first query
	xqueryname = querieslist[0].QueryName
	xdestqueryname = destqueryname
	if destqueryname == "" {
		xdestqueryname = xqueryname
	}

	// If only one query, aggegation and identation are not needed
	if len(querieslist) == 1 {
		qry := &querieslist[0]
		extratag = ""
		querycode = qry.SourceCode
		// If query name changed, must ensure the right base.<newname> is being called/referred
		if xqueryname != xdestqueryname {
			extratag = extratag + "// QUERY RENAMED FROM " + xqueryname + " TO " + xdestqueryname + "\n"
			extratag = extratag + "// ======================================================= \n"
			querycode = strings.Replace(querycode, "base."+xqueryname+"()", "base."+xdestqueryname+"()", -1)
		}
		result = qry.tag + extratag + "\n" + querycode + "\n"
		status = STATUS_OK
		return result, status, statusmessage
	}

	// Analyze query code for:
	// - Broken chain, base.<queryname> not called
	// - Unhandled code, base.<queryname> is not assigned to "result" directly
	index = 0
	for i := range querieslist {
		qry := &querieslist[i]
		qry.callsbase, qry.issafe = codecallsbase(qry.SourceCode, qry.QueryName)
		if !(qry.callsbase) {
			firstindex = index
		}
		index++
	}

	// Inject code fixes into the queries
	for i := range querieslist {
		qry := &querieslist[i]
		extratag = ""
		querycode = qry.SourceCode
		if result != "" {
			result = result + "\n\n\n"
		}
		// If query name changed, must ensure the right base.<newname> is being called/referred
		if xqueryname != xdestqueryname {
			extratag = extratag + "// QUERY RENAMED FROM " + xqueryname + " TO " + xdestqueryname + "\n"
			querycode = strings.Replace(querycode, "base."+xqueryname+"()", "base."+xdestqueryname+"()", -1)
		}
		// Other injections are not relevant for the first query in the chain. Only for the next
		if querycount >= firstindex {
			if !(qry.callsbase) {
				extratag = extratag + "// BASE CALL CHAIN BROKEN - QUERY DOES NOT CALL BASE\n"
			} else if !(qry.issafe) {
				extratag = extratag + "// DIRECT RESULT ASSIGNMENT UNDETECTED - result = base.<x> \n"
			}
		}
		if extratag != "" {
			extratag = extratag + "// ======================================================= \n"
		}
		// Now check fixes
		if querycount > firstindex {
			if !(qry.callsbase) {
				queryinject = "\n"
				queryinject = queryinject + "// ---------- >> AUTO ADDED BY MERGE\n"
				queryinject = queryinject + "result.Clear();\n"
				queryinject = queryinject + "// << ---------- AUTO ADDED BY MERGE\n\n"
				querycode = queryinject + querycode
			} else if !(qry.issafe) {
				sbase = "base." + xdestqueryname + "()"
				xbase = "_merged_base_" + xdestqueryname
				xcounter = 0
				xresult = xbase
				for {
					if !(strings.Contains(querycode, xresult)) {
						break
					}
					xcounter++
					xresult = xbase + strconv.Itoa(xcounter)
				}
				// Inject the fix in the code ...
				querycode = strings.Replace(querycode, sbase, xresult, -1)
				queryinject = "\n"
				queryinject = queryinject + "// ---------- >> AUTO ADDED BY MERGE\n"
				queryinject = queryinject + "CxList " + xresult + " = result.Clone();\n"
				queryinject = queryinject + "result.Clear();\n"
				queryinject = queryinject + "// << ---------- AUTO ADDED BY MERGE\n\n"
				querycode = queryinject + querycode
			} else {
				sbase = "base." + xdestqueryname + "()"
				querycode = strings.Replace(querycode, sbase, "result", -1)
			}
		}
		result = result + qry.tag + extratag + "{\n" + arrangecode(querycode, (querycount < firstindex)) + "\n}"
		querycount++
	}

	return result, status, statusmessage
}

// Validate if the
func (q *QueryMerger) validate_queries_struct() (status int, message string) {
	var querieslist QueryMerger = *q
	var queryremerge bool
	var queryname string
	var querylang string
	//var queryteam string

	// No queries to process
	if len(querieslist) < 1 {
		return STATUS_EMPTY, "Cannot process an empty set of queries"
	}

	queryname = querieslist[0].QueryName
	querylang = querieslist[0].Language
	//queryteam = querieslist[0].TeamOrProjName

	// Detect if this query is a remerge
	if strings.Contains(querieslist[0].SourceCode, "// MERGED - PROJECT LEVEL\n") || strings.Contains(querieslist[0].SourceCode, "// MERGED - TEAM LEVEL\n") || strings.Contains(querieslist[0].SourceCode, "// MERGED - CORPORATE LEVEL\n") {
		queryremerge = true
	} else {
		queryremerge = false
	}

	// If only one query, we can go
	if len(querieslist) == 1 {
		// Query severity must be between 0 and 3
		if querieslist[0].Severity < 0 || querieslist[0].Severity > 3 {
			return STATUS_INVALID, "Query severity out of range"
		} else {
			if queryremerge {
				return STATUS_REMERGE, ""
			} else {
				return STATUS_OK, ""
			}
		}
	}

	// Corp level queries can't be used in aggregation
	for i := range querieslist {
		// Query name must be the same
		if querieslist[i].QueryName != queryname {
			return STATUS_INVALID, "Query name must be the same"
		}
		// Query language must be the same
		if querieslist[i].Language != querylang {
			return STATUS_INVALID, "Query language must be the same"
		}
		// Query severity must be between 0 and 3
		if querieslist[i].Severity < 0 || querieslist[i].Severity > 3 {
			return STATUS_INVALID, "Query severity out of range"
		}
		// Corp level queries can't be used in aggregation
		if querieslist[i].Level == CxSASTClientGo.CORP_QUERY {
			return STATUS_INVALID, "Corp level queries cannot be merged"
		}
		// Project level query in aggregation must be unique and the last on the list
		if querieslist[i].Level == CxSASTClientGo.PROJECT_QUERY && i < len(querieslist)-1 {
			return STATUS_INVALID, "Project level query must be the last on the list"
		}
		// Check team sequence
		/*if querieslist[i].Level == CxSASTClientGo.TEAM_QUERY {
			if strings.HasPrefix(querieslist[i].TeamOrProjName, queryteam) {
				queryteam = querieslist[i].TeamOrProjName
			} else {
				return STATUS_INVALID, "Team level query not in same tree hierachy"
			}
		}*/
	}

	if queryremerge {
		return STATUS_REMERGE, ""
	} else {
		return STATUS_OK, ""
	}
}

// Merge the list of queries int a singe code
// Parameter:
//   - destqueryname	to detect and process query renames
//     use an empty string if this check is not needed
//     Example:
//     JS: Potentially_Vulnerable_To_Xsrf     	in v9.3
//     Found as Potentially_Vulnerable_To_CSRF 	in v9.5.5 and CXONE
//
// Returns:
// - qquerycode		the merged query code
// - qstatus		the status of the merged query
func (q *QueryMerger) Merge(destqueryname string) (qquerycode string, err error) {
	var xstatus int = STATUS_OK
	var xstatusmessage string = ""
	var xresult string
	xresult, xstatus, xstatusmessage = q.merge_queries(destqueryname)
	if xstatus > STATUS_REMERGE {
		return xresult, errors.New(xstatusmessage)

	} else {
		return xresult, nil
	}
}

// Helper funtion to check the contents
func (q *QueryMerger) CheckStatus() (status int, message string) {
	var xstatus int
	var xstatusmessage string
	xstatus, xstatusmessage = q.validate_queries_struct()
	return xstatus, xstatusmessage
}

// Helper funtion to get CxQL code without comments
// Returns:
// - quncommentedcode		the uncommented and merged query code
// It does not check for errors, just deliver the uncommented code
func (q *QueryMerger) UncommentedCode() (quncommentedcode string) {
	var xresult string
	xresult, _, _ = q.merge_queries("")
	if xresult != "" {
		xresult = uncommentedcode(xresult)
	}
	return xresult
}
