package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/dolthub/go-mysql-server/sql"
	"golang.org/x/exp/slices"
	"os"
	"sort"
)

type AnalysisOutput struct {
	queriesOutputPath  string
	analysisOutputPath string
}

type TestRun struct {
	Queries       QueryCollection
	FailedTestIds []string
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

	failedTests, err := getFailedTests(settings)
	if err != nil {
		return result, err
	}
	result.FailedTestIds = failedTests

	queries, err := parseQueries(settings, failedTests)
	if err != nil {
		return result, err
	}
	result.Queries = queries

	return result, nil
}

func getFailedTests(settings Settings) ([]string, error) {
	failedTests := make([]string, 0)
	if settings.pytestReportPath != "" {
		input, err := os.Open(settings.pytestReportPath)
		if err != nil {
			return failedTests, err
		}
		defer input.Close()

		separatorSeen := false
		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			line := scanner.Text()
			if line == pytestReportSeparator {
				separatorSeen = true
				continue
			}

			if !separatorSeen {
				continue
			}

			// check if the test failed
			failedTestParts := RegexSplit(line, pyTestFailedRegex)
			if len(failedTestParts) == 0 {
				// check if the test errored instead
				failedTestParts = RegexSplit(line, pyTestErrorRegex)
			}

			if len(failedTestParts) == 2 {
				testId := fmt.Sprintf("%s.%s", failedTestParts[1], failedTestParts[0])
				failedTests = append(failedTests, testId)
			}
		}
	}
	return failedTests, nil
}

func parseQueries(settings Settings, failedTestIds []string) (QueryCollection, error) {
	result := NewQueryCollection()
	ctx := sql.NewEmptyContext()
	logger := settings.logger

	input, err := os.Open(settings.doltLogFilePath)
	if err != nil {
		return result, err
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
			// this was a "notification" query, no need to analyze it
			continue
		case testFinishedParse != nil:
			finishedTestId := testFinishedParse[0]
			if finishedTestId != testId {
				logger.Logf("Line %d, test id mismatch: %s (this test) != %s (last started test)",
					lineNumber, finishedTestId, testId)
			}
			testId = ""
			// this was a "notification" query, no need to analyze it
			continue
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
		parsedNode, err := ParseQuery(ctx, query)
		if err != nil {
			logger.Logf("Error parsing query '%s': %s", query, err)
		} else {
			node = parsedNode
		}

		queryObj := Query{
			TestId:     testId,
			PyTestName: pyTestName,
			TestFailed: testFailed,
			Text:       query,
			Node:       node,
			LineNumber: lineNumber,
			Error:      queryError,
		}
		result.Add(queryObj)
	}

	return result, nil
}

func AnalyzeTestRun(settings Settings) (AnalysisOutput, error) {
	result := AnalysisOutput{}

	testRun, err := parseTestRun(settings)
	if err != nil {
		return result, err
	}
	queryCollection := testRun.Queries
	//failedTestIds := testRun.FailedTestIds

	// write the queries to a file
	queriesOutputPath := settings.GetOutputFilePath(".queries")
	queriesOutput, err := os.Create(queriesOutputPath)
	if err != nil {
		return result, err
	}
	defer queriesOutput.Close()
	queriesLogger := NewProxyLogger(NewFileLogger(queriesOutput), settings.logger)
	for _, query := range queryCollection.All {
		queriesLogger.Logf(query.String(settings.logQueryText))
		queriesLogger.Log(analysisReportSeparator)
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

	// fill in the results
	result.queriesOutputPath = queriesOutputPath
	result.analysisOutputPath = analysisOutputPath
	return result, nil
}
