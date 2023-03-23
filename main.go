package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/dolthub/go-mysql-server/sql"
	"os"
	"strings"
)

type Settings struct {
	// Path to the dolt log file
	doltLogFilePath string
	// Path to the output file
	outputFilePath string
	// Logger to use for logging
	logger Logger
	// Whether to hide queries that are not associated with a test
	hideNonTestQueries bool
	// Whether to log query text
	logQueryText bool
}

func main() {
	settings := readInputs()
	err := mainLogic(settings)
	if err != nil {
		panic(err)
	}
}

func readInputs() Settings {
	var verbose bool
	var log string
	var out string
	var hideNonTestQueries bool
	var showQueryText bool

	flag.StringVar(&log, "log", "", "Path to the dolt log file")
	flag.StringVar(&out, "out", "", "Path to the output file")
	flag.BoolVar(&hideNonTestQueries, "hide-non-test-queries", false, "Whether to hide queries that are not associated with a test")
	flag.BoolVar(&showQueryText, "show-query-text", false, "Whether to log query text")

	flag.BoolVar(&verbose, "verbose", false, "Whether to log to stdout")
	flag.BoolVar(&verbose, "v", false, "Whether to log to stdout")
	flag.Parse()

	settings := Settings{
		doltLogFilePath:    log,
		outputFilePath:     out,
		hideNonTestQueries: hideNonTestQueries,
		logQueryText:       showQueryText,
	}
	if verbose {
		settings.logger = NewConsoleLogger()
	}
	return settings
}

func mainLogic(settings Settings) error {
	input, err := os.Open(settings.doltLogFilePath)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(settings.outputFilePath)
	if err != nil {
		return err
	}
	defer output.Close()

	logAndWriteLn := func(format string, a ...any) {
		message := fmt.Sprintf(format, a...) + "\n"
		output.WriteString(message)
		if settings.logger != nil {
			settings.logger.Log(message)
		}
	}

	parseInput(input, logAndWriteLn, settings)

	return nil
}

// parseInput parses the input file and writes the results to the output file
// this function does not return errors, instead it logs them to the output file, along
// with the line number where the error occurred and the query that caused the error
func parseInput(input *os.File, logAndWriteLn func(format string, a ...any), settings Settings) {
	ctx := sql.NewEmptyContext()
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
				logAndWriteLn("Line %d, test id mismatch: %s (this test) != %s (last started test)", lineNumber, finishedTestId, testId)
			}
			testId = ""
			// this was a "notification" query, no need to analyze it
			continue
		}

		if settings.hideNonTestQueries && testId == "" {
			continue
		}

		logAndWriteLn("Line %d", lineNumber)

		if testId != "" {
			pyTestName := getPyTestName(testId)
			logAndWriteLn("Test: %s", pyTestName)
		}

		if settings.logQueryText {
			logAndWriteLn("Query:\n%s", query)
		}

		node, err := ParseQuery(ctx, query)
		if err != nil {
			logAndWriteLn("Error parsing query '%s': %s", query, err)
			continue
		}

		logAndWriteLn("Query tree:\n%s", sql.DebugString(node))
		if queryError != "" {
			logAndWriteLn("Query error: %s", queryError)
		}

		logAndWriteLn("--------------------------------------------------")
	}
}

func getPyTestName(fullTestId string) string {
	// test_napalm_args (nautobot.dcim.tests.test_filters.PlatformTestCase)
	lastPeriodIndex := strings.LastIndex(fullTestId, ".")
	testName := fullTestId[lastPeriodIndex+1:]
	testSuite := fullTestId[:lastPeriodIndex]
	pyTestName := fmt.Sprintf("%s (%s)", testName, testSuite)
	return pyTestName
}
