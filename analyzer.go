package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/parse"
	"github.com/dolthub/go-mysql-server/sql/plan"
	"github.com/dolthub/go-mysql-server/sql/transform"
	"golang.org/x/exp/slices"
	"os"
	"sort"
	"strings"
)

type AnalysisOutput struct {
	analysisOutputPath     string
	patchQueriesOutputPath string
	queriesOutputPath      string
	testsOutputPath        string
}

type TestRun struct {
	Queries       QueryCollection
	FailedTestIds []string
	Tests         []Test
	PatchQueries  []PatchQuery
}

type PatchQuery struct {
	LineNumber int
	TableName  string
	Queries    []string
}

type Test struct {
	Id         string
	Failed     bool
	Queries    []Query
	TablesUsed []string
}

func (t *Test) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Test %s / %s\n", t.Id, PyTestNameFromTestId(t.Id)))
	if t.Failed {
		sb.WriteString(fmt.Sprintf("Failed: %t\n", t.Failed))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("Tables used: %s\n", strings.Join(t.TablesUsed, ", ")))
	sb.WriteString("Queries: \n")
	for _, query := range t.Queries {
		sb.WriteString(fmt.Sprintf("%s;\n", query.Text))
	}
	sb.WriteString("\n")

	sb.WriteString("send_query(DOLT_PATCH) calls for used tables:\n")
	sb.WriteString("if is_db_dolt():\n")
	for _, table := range t.TablesUsed {
		sb.WriteString(fmt.Sprintf(`    send_query("SELECT statement_order, TO_BASE64(statement) FROM DOLT_PATCH('HEAD', 'WORKING', '%s');",True)`, table))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

var pytestReportSeparator = "======================================================================"

var analysisReportSeparator = "--------------------------------------------------\n"

type Pair[T, U any] struct {
	First  T
	Second U
}

// parseInput parses the input file and returns a QueryCollection of all the input queries.
func parseTestRun(settings Settings) (TestRun, error) {
	result := TestRun{}

	failedTestIds, patchQueries, err := parsePytestReport(settings)
	if err != nil {
		return result, err
	}
	result.FailedTestIds = failedTestIds
	result.PatchQueries = patchQueries

	queries, tests, err := parseQueries(settings, failedTestIds)
	if err != nil {
		return result, err
	}
	result.Queries = queries
	result.Tests = tests

	return result, nil
}

func parsePytestReport(settings Settings) (failedTestIds []string, patchQueries []PatchQuery, err error) {
	patchQueries = make([]PatchQuery, 0)
	failedTestIds = make([]string, 0)
	if settings.pytestReportPath != "" {
		input, err := os.Open(settings.pytestReportPath)
		if err != nil {
			return failedTestIds, patchQueries, err
		}
		defer input.Close()

		lineNumber := 0
		nextLineHasPatchQuery := false
		patchQueryTargetTable := ""
		separatorSeen := false
		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			line := scanner.Text()
			lineNumber++
			if line == pytestReportSeparator {
				separatorSeen = true
			}

			if separatorSeen {
				// check if the test failed
				failedTestParts := RegexSplit(line, pyTestFailedRegex)
				if len(failedTestParts) == 0 {
					// check if the test errored instead
					failedTestParts = RegexSplit(line, pyTestErrorRegex)
				}

				if len(failedTestParts) == 2 {
					testId := fmt.Sprintf("%s.%s", failedTestParts[1], failedTestParts[0])
					failedTestIds = append(failedTestIds, testId)
				}
			} else {
				if nextLineHasPatchQuery {
					pq := PatchQuery{
						TableName:  patchQueryTargetTable,
						LineNumber: lineNumber,
					}
					resultMatch := RegexSplit(line, pyTestPatchResultRegex)
					result := resultMatch[0]
					items := strings.Split(result, "), (")
					for _, item := range items {
						item = strings.Replace(item, "\\n", "", -1)
						itemMatch := RegexSplit(item, `.*, '(.*)'.*`)
						b64query := itemMatch[0]
						queryBytes, _ := base64.StdEncoding.DecodeString(b64query)
						query := string(queryBytes)
						pq.Queries = append(pq.Queries, query)
					}

					patchQueries = append(patchQueries, pq)
					nextLineHasPatchQuery = false
				} else {
					shouldDebug := strings.Contains(line, "FROM DOLT_PATCH")
					if shouldDebug {
						fmt.Println(line)
					}
					// check if this is a patch query result
					patchQuery := RegexSplit(line, pyTestPatchRegex)
					if len(patchQuery) == 1 {
						nextLineHasPatchQuery = true
						patchQueryTargetTable = patchQuery[0]
					}
				}
			}
		}
	}
	return failedTestIds, patchQueries, nil
}

func parseQueries(settings Settings, failedTestIds []string) (QueryCollection, []Test, error) {
	queryCollection := NewQueryCollection()
	tests := []Test{}
	var currentTest *Test
	ctx := sql.NewEmptyContext()
	logger := settings.logger

	input, err := os.Open(settings.doltLogFilePath)
	if err != nil {
		return queryCollection, tests, err
	}
	defer input.Close()

	testId := ""
	lineNumber := 0
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		finishedParse := RegexSplit(line, finishedQueryRegex)
		errorParse := RegexSplit(line, errorQueryRegex)

		var query string
		var queryError string

		switch {
		case finishedParse != nil:
			query = finishedParse[0]
		case errorParse != nil:
			queryError = errorParse[0]
			query = errorParse[1]
		default:
			continue
		}

		if query == "" {
			continue
		}

		// recent dolt supports base64 encoding of queries, try to decode
		queryBytes, err := base64.StdEncoding.DecodeString(query)
		if err == nil {
			query = string(queryBytes)
		}

		testStartingParse := RegexSplit(query, testStartingRegex)
		testFinishedParse := RegexSplit(query, testFinishedRegex)
		switch {
		case testStartingParse != nil:
			testId = testStartingParse[0]
			//// this was a "notification" query, no need to analyze it
			//continue
		case testFinishedParse != nil:
			finishedTestId := testFinishedParse[0]
			if finishedTestId != testId {
				logger.Logf("Line %d, test id mismatch: %s (this test) != %s (last started test)",
					lineNumber, finishedTestId, testId)
			}
			testId = ""
			//// this was a "notification" query, no need to analyze it
			//continue
		}

		if settings.hideNonTestQueries && testId == "" {
			continue
		}

		var testFailed bool
		var pyTestName string
		if testId != "" {
			pyTestName = PyTestNameFromTestId(testId)
			testFailed = slices.Contains(failedTestIds, testId)
		}

		var node sql.Node
		parsedNode, err := parse.Parse(ctx, query)
		if err != nil {
			logger.Logf("Line %d, error parsing query '%s': %s", lineNumber, query, err)
			continue
		} else {
			node = parsedNode
		}

		//cleanNode, err := DropExtraneousData(node)
		//if err != nil {
		//	logger.Logf("Line %d, error cleaning query '%s': %s", lineNumber, query, err)
		//	continue
		//} else {
		//	node = cleanNode
		//}

		nodeDebugString := sql.DebugString(node)

		queryObj := Query{
			TestId:     testId,
			PyTestName: pyTestName,
			TestFailed: testFailed,
			Text:       query,
			Node:       node,
			LineNumber: lineNumber,
			Error:      queryError,
			NodeDebug:  nodeDebugString,
		}
		queryCollection.Add(queryObj)

		if testId != "" {
			tablesUsed := getTablesUsed(node)

			if currentTest == nil || currentTest.Id != testId {
				if currentTest != nil {
					tests = append(tests, *currentTest)
					currentTest = nil
				}
				currentTest = &Test{
					Id:         testId,
					Failed:     testFailed,
					Queries:    []Query{queryObj},
					TablesUsed: tablesUsed,
				}
			} else {
				currentTest.Queries = append(currentTest.Queries, queryObj)
				for _, table := range tablesUsed {
					if !slices.Contains(currentTest.TablesUsed, table) {
						currentTest.TablesUsed = append(currentTest.TablesUsed, table)
					}
				}
			}
		}
	}

	if currentTest != nil {
		tests = append(tests, *currentTest)
		currentTest = nil
	}

	return queryCollection, tests, nil
}

func getTablesUsed(node sql.Node) []string {
	tables := []string{}
	transform.Inspect(node, func(node sql.Node) bool {
		var tableName string
		switch node := node.(type) {
		case *plan.UnresolvedTable:
			tableName = node.Name()
		case *plan.ResolvedTable:
			tableName = node.Name()
		}
		if tableName != "" && !slices.Contains(tables, tableName) {
			tables = append(tables, tableName)
		}

		expressioner, ok := node.(sql.Expressioner)
		if ok {
			expressions := expressioner.Expressions()
			for _, expr := range expressions {
				transform.InspectExpr(expr, func(expr sql.Expression) bool {
					switch expr := expr.(type) {
					case *plan.Subquery:
						subTables := getTablesUsed(expr.Query)
						for _, subTable := range subTables {
							if !slices.Contains(tables, subTable) {
								tables = append(tables, subTable)
							}
						}
					}
					return true
				})
			}
		}

		return true
	})
	return tables
}

func AnalyzeTestRun(settings Settings) (AnalysisOutput, error) {
	result := AnalysisOutput{}

	testRun, err := parseTestRun(settings)
	if err != nil {
		return result, err
	}
	queryCollection := testRun.Queries

	// output queries
	if len(queryCollection.All) > 0 {
		// write the queries to a file
		queriesOutputPath := settings.GetOutputFilePath(".queries")
		queriesOutput, err := os.Create(queriesOutputPath)
		if err != nil {
			return result, err
		}
		defer queriesOutput.Close()
		queriesLogger := NewProxyLogger(NewFileLogger(queriesOutput), settings.logger)

		// write the queries to a file, one line per query
		flatQueriesOutputPath := settings.GetOutputFilePath(".queries_flat")
		flatQueriesOutput, err := os.Create(flatQueriesOutputPath)
		if err != nil {
			return result, err
		}
		defer queriesOutput.Close()
		flatQueriesLogger := NewFileLogger(flatQueriesOutput)

		for _, query := range queryCollection.All {
			queriesLogger.Logf(query.String(settings.logQueryText))
			queriesLogger.Log(analysisReportSeparator)
			flatQueriesLogger.Logf("%s;\n", query.Text)
		}
		result.queriesOutputPath = queriesOutputPath
	}

	// unencode 'from dolt_patch' queries and write them to a file
	if len(testRun.PatchQueries) > 0 {
		patchQueriesOutputPath := settings.GetOutputFilePath(".patch_queries")
		patchQueriesOutput, err := os.Create(patchQueriesOutputPath)
		if err != nil {
			return result, err
		}
		defer patchQueriesOutput.Close()
		patchQueriesLogger := NewFileLogger(patchQueriesOutput)

		for _, patchQuery := range testRun.PatchQueries {
			patchQueriesLogger.Logf("Report line: %d\n", patchQuery.LineNumber)
			patchQueriesLogger.Logf("Table: %s\n\n", patchQuery.TableName)

			for _, query := range patchQuery.Queries {
				patchQueriesLogger.Logf("%s\n", query)
			}
			patchQueriesLogger.Log(analysisReportSeparator)
		}
		result.patchQueriesOutputPath = patchQueriesOutputPath
	}

	// write tests to a file
	if len(testRun.Tests) > 0 {
		testsOutputPath := settings.GetOutputFilePath(".tests")
		testsOutput, err := os.Create(testsOutputPath)
		if err != nil {
			return result, err
		}
		defer testsOutput.Close()
		testsLogger := NewFileLogger(testsOutput)
		for _, test := range testRun.Tests {
			testsLogger.Logf(test.String())
			testsLogger.Log(analysisReportSeparator)
		}
		result.testsOutputPath = testsOutputPath
	}

	// write analysis to a file
	analysisOutputPath := settings.GetOutputFilePath(".analysis")
	analysisOutput, err := os.Create(analysisOutputPath)
	if err != nil {
		return result, err
	}
	defer analysisOutput.Close()
	analysisLogger := NewProxyLogger(NewFileLogger(analysisOutput), settings.logger)

	analysisLogger.Logf("Total queries: %d\n", len(queryCollection.All))
	analysisLogger.Logf("Number of tests: %d\n", len(queryCollection.ByTestId))
	analysisLogger.Logf("Number of test queries: %d\n", len(queryCollection.TestQueries))
	analysisLogger.Log(analysisReportSeparator)
	result.analysisOutputPath = analysisOutputPath

	sortedListOfTestQueries := sortQueries(queryCollection)

	for _, pair := range sortedListOfTestQueries {
		dbg := pair.First
		queries := pair.Second

		analysisLogger.Logf("Debug string: \n%s\n", dbg)
		analysisLogger.Logf("Number of queries: %d\n", len(queries))

		for index, query := range queries {
			analysisLogger.Logf("Query %d/%d:\n%s\n", index+1, len(queries), query.String(settings.logQueryText))
		}

		analysisLogger.Logf(analysisReportSeparator)
	}

	return result, nil
}

func sortQueries(queryCollection QueryCollection) []Pair[string, []Query] {
	sortedListOfTestQueries := []Pair[string, []Query]{}
	for dbg, queries := range queryCollection.ByDebugString {
		testQueries := make([]Query, 0)
		for _, query := range queries {
			if query.TestId != "" {
				testQueries = append(testQueries, query)
			}
		}
		if len(testQueries) > 0 {
			pair := Pair[string, []Query]{dbg, testQueries}
			sortedListOfTestQueries = append(sortedListOfTestQueries, pair)
		}
	}

	// sort list of queries by:
	// 1. number of failed tests
	// 2. number of queries
	// 3. debug string
	countFailed := func(queries Query) bool {
		return queries.TestFailed
	}
	sort.SliceStable(sortedListOfTestQueries, func(i, j int) bool {
		leftPair := sortedListOfTestQueries[i]
		leftQueries := leftPair.Second
		leftFailedTestsCount := Count(leftQueries, countFailed)

		rightPair := sortedListOfTestQueries[j]
		rightQueries := rightPair.Second
		rightFailedTestsCount := Count(rightQueries, countFailed)

		switch {
		case leftFailedTestsCount != rightFailedTestsCount:
			return leftFailedTestsCount > rightFailedTestsCount
		case len(leftQueries) != len(rightQueries):
			return len(leftQueries) > len(rightQueries)
		default:
			return leftPair.First < rightPair.First
		}
	})
	return sortedListOfTestQueries
}
