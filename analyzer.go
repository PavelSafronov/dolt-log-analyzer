package main

import (
	"os"
	"sort"
)

type AnalysisOutput struct {
	queriesOutputPath  string
	analysisOutputPath string
}

func Analyze(queryCollection QueryCollection, settings Settings) (AnalysisOutput, error) {
	result := AnalysisOutput{}

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
		queriesLogger.Log(separator)
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

	analysisLogger.Logf(separator)

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

	sort.SliceStable(sortedListOfTestQueries, func(i, j int) bool {
		left := sortedListOfTestQueries[i].Second
		right := sortedListOfTestQueries[j].Second

		if len(left) == len(right) {
			for i := 0; i < len(left); i++ {
				l := left[i]
				r := right[i]
				return l.LineNumber <= r.LineNumber
			}
			return false
		} else {
			return len(left) > len(right)
		}
	})
	for _, pair := range sortedListOfTestQueries {
		dbg := pair.First
		queries := pair.Second

		analysisLogger.Logf("Debug string: %s\n", dbg)
		analysisLogger.Logf("Number of queries: %d\n", len(queries))
		testQueries := [][]Query{}
		for _, query := range queries {
			if query.TestId != "" {
				testQueries = append(testQueries, []Query{query})
			}
		}
		analysisLogger.Logf("Number of test queries: %d\n", len(testQueries))
		analysisLogger.Logf(separator)
	}

	// fill in the results
	result.queriesOutputPath = queriesOutputPath
	result.analysisOutputPath = analysisOutputPath
	return result, nil
}
