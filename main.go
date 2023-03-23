package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/dolthub/go-mysql-server/sql"
	"os"
)

type Settings struct {
	// Path to the dolt log file
	doltLogFilePath string
	// Path to the output file
	outputFilePath string
	// Logger to use for logging
	logger Logger
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

	flag.StringVar(&log, "log", "", "Path to the dolt log file")
	flag.StringVar(&out, "out", "", "Path to the output file")
	flag.BoolVar(&verbose, "verbose", false, "Whether to log to stdout")
	flag.BoolVar(&verbose, "v", false, "Whether to log to stdout")
	flag.Parse()

	settings := Settings{
		doltLogFilePath: log,
		outputFilePath:  out,
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

	parseInput(input, logAndWriteLn)

	return nil
}

// parseInput parses the input file and writes the results to the output file
// this function does not return errors, instead it logs them to the output file, along
// with the line number where the error occurred and the query that caused the error
func parseInput(input *os.File, logAndWriteLn func(format string, a ...any)) {
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
				logAndWriteLn("line %d, test id mismatch: %s != %s\n", lineNumber, finishedTestId, testId)
			}
			testId = ""
			// this was a "notification" query, no need to analyze it
			continue
		}

		if testId != "" {
			logAndWriteLn("Test: %s\n", testId)
		}

		node, err := ParseQuery(ctx, query)
		if err != nil {
			logAndWriteLn("line %d, error parsing query '%s': %s\n", lineNumber, query, err)
			continue
		}

		logAndWriteLn("line %d, query tree: \n%s\n",
			lineNumber, sql.DebugString(node))
		if queryError != "" {
			logAndWriteLn("Query error: %s\n", queryError)
		}

		logAndWriteLn("--------------------------------------------------")
	}
}
