package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/parse"
	"github.com/dolthub/go-mysql-server/sql/transform"
	"os"
	"regexp"
)

func Noop(data any) {

}

var (
	doltLogFilePath string
	outputFilePath  string
)

var (
	// 2023-03-22T18:55:23Z DEBUG [conn 2] Query finished in 1 ms {connectTime=2023-03-22T18:55:23Z, connectionDb=, query=SET NAMES utf8mb4}
	finishedQueryRegex = ".*] Query finished in .*{.*, query=(.*)}"
	// 2023-03-22T18:55:23Z WARN [conn 2] error running query {connectTime=2023-03-22T18:55:23Z, connectionDb=, error=can't create database test_nautobot; database exists, query=CREATE DATABASE `test_nautobot`}
	errorQueryRegex = ".*] error running query {.*, error=(.*), query=(.*)}"
	// select 'dolt: setUp, test id = nautobot.dcim.tests.test_filters.CableTestCase.test_color'
	testStartingRegex = "select 'dolt: setUp, test id = (.*)'"
	// select 'dolt: _post_teardown, test id = nautobot.dcim.tests.test_filters.CableTestCase.test_id'
	testFinishedRegex = "select 'dolt: _post_teardown, test id = (.*)'"
)

func main() {
	readInputs()
	err := mainLogic(doltLogFilePath, outputFilePath)
	if err != nil {
		panic(err)
	}
}

func readInputs() {
	flag.StringVar(&doltLogFilePath, "log", "", "Path to the dolt log file")
	flag.StringVar(&outputFilePath, "out", "", "Path to the output file")
	flag.Parse()
}

func mainLogic(log string, out string) error {
	input, err := os.Open(log)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(out)
	if err != nil {
		return err
	}
	defer output.Close()

	logAndWriteLn := func(format string, a ...any) {
		message := fmt.Sprintf(format, a...) + "\n"
		output.WriteString(message)
		fmt.Print(message)
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

		node, err := parse.Parse(ctx, query)
		if err != nil {
			logAndWriteLn("line %d, error parsing query '%s': %s\n", lineNumber, query, err)
			continue
		}
		node, err = DropExtraneousData(node)
		if err != nil {
			logAndWriteLn("line %d, error dropping extraneous data from query: %s\n", lineNumber, err)
			continue
		}
		logAndWriteLn("line %d, query tree: \n%s\n",
			lineNumber, sql.DebugString(node))
		if queryError != "" {
			logAndWriteLn("ERROR: %s\n", queryError)
		}

		logAndWriteLn("--------------------------------------------------")
	}
}

func DropExtraneousData(node sql.Node) (sql.Node, error) {
	newNode, _, err := transform.Node(node, func(node sql.Node) (sql.Node, transform.TreeIdentity, error) {
		var newNode sql.Node
		switch node := node.(type) {
		//case *plan.Project:
		//	newNode = plan.NewProject([]sql.Expression{node.Projections[0]}, node.Child)
		//case *plan.Filter:
		//	newNode = plan.NewFilter(node.Expression, node.Child)
		default:
			return node, transform.SameTree, nil
		}
		return newNode, transform.NewTree, nil
	})
	return newNode, err
}

func RegexSplit(text string, exp string) []string {
	regex := regexp.MustCompile(exp)
	matches := regex.FindAllSubmatch([]byte(text), -1)
	if matches == nil {
		return nil
	}
	result := make([]string, len(matches[0])-1)
	for i := 1; i < len(matches[0]); i++ {
		result[i-1] = string(matches[0][i])
	}
	return result
}
