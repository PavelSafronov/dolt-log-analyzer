package main

import (
	"fmt"
)

func main() {
	settings := readInputs()
	result, err := mainLogic(settings)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Queries output: %s\n", result.queriesOutputPath)
	fmt.Printf("Analysis output: %s\n", result.analysisOutputPath)
}

func mainLogic(settings Settings) (AnalysisOutput, error) {
	// parse and analyze the input file
	analysis, err := AnalyzeTestRun(settings)
	return analysis, err
}
