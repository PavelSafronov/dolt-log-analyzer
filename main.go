package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/dolthub/go-mysql-server/sql"
	"os"
	"strings"
)

func main() {
	settings := readInputs()
	result, err := mainLogic(settings)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Queries output: %s", result.queriesOutputPath)
	fmt.Printf("Analysis output: %s", result.analysisOutputPath)
}

var separator = "--------------------------------------------------\n"

type Pair[T, U any] struct {
	First  T
	Second U
}

func mainLogic(settings Settings) (AnalysisOutput, error) {
	// parse the input file
	queryCollection, err := parseInput(settings)
	if err != nil {
		return AnalysisOutput{}, err
	}

	// analyze the queries
	analysis, err := Analyze(queryCollection, settings)
	return analysis, err
}

// parseInput parses the input file and returns a QueryCollection of all the input queries.
func parseInput(settings Settings) (QueryCollection, error) {
	collection := NewQueryCollection()
	ctx := sql.NewEmptyContext()
	logger := settings.logger

	input, err := os.Open(settings.doltLogFilePath)
	if err != nil {
		return collection, err
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

		var pyTestName string
		if testId != "" {
			// test_napalm_args (nautobot.dcim.tests.test_filters.PlatformTestCase)
			lastPeriodIndex := strings.LastIndex(testId, ".")
			testName := testId[lastPeriodIndex+1:]
			testSuite := testId[:lastPeriodIndex]
			pyTestName = fmt.Sprintf("%s (%s)", testName, testSuite)
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
			Text:       query,
			Node:       node,
			LineNumber: lineNumber,
			PyTestName: pyTestName,
			Error:      queryError,
		}
		collection.Add(queryObj)
	}

	return collection, nil
}
