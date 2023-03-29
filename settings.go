package main

import (
	"flag"
	"path/filepath"
)

type Settings struct {
	// Path to the dolt log file
	doltLogFilePath string
	// Path to the pytest report file
	pytestReportPath string
	// Path to the output directory
	outputDirPath string
	// Base name of the output files, taken from the dolt log file name
	outputFileBaseName string
	// Logger to use for logging
	logger Logger
	// Whether to hide queries that are not associated with a test
	hideNonTestQueries bool

	// Whether to log query text
	logQueryText bool
	// The extension to use for the output files, taken from the dolt log file name
	logFileExtension string
}

func NewSettings(logPath string, pytestReportPath string) Settings {
	logFileName := filepath.Base(logPath)
	logFileExt := filepath.Ext(logFileName)
	outputFileBaseName := logFileName[:len(logFileName)-len(logFileExt)]

	settings := Settings{
		doltLogFilePath:    logPath,
		pytestReportPath:   pytestReportPath,
		outputDirPath:      filepath.Dir(logPath),
		hideNonTestQueries: false,
		logQueryText:       true,
		outputFileBaseName: outputFileBaseName,
		logFileExtension:   logFileExt,
		logger:             NewConsoleLogger(),
	}
	return settings
}

func readInputs() Settings {
	var verbose bool
	var logPath string
	var pytestReportPath string
	var hideNonTestQueries bool
	var showQueryText bool

	flag.StringVar(&logPath, "log", "", "Path to the dolt log file")
	flag.StringVar(&pytestReportPath, "pytest-report", "", "Path to the pytest report file")
	flag.BoolVar(&hideNonTestQueries, "hide-non-test-queries", false, "Whether to hide queries that are not associated with a test")
	flag.BoolVar(&showQueryText, "show-query-text", false, "Whether to log query text")

	flag.BoolVar(&verbose, "verbose", false, "Whether to log to stdout")
	flag.BoolVar(&verbose, "v", false, "Whether to log to stdout")
	flag.Parse()

	settings := NewSettings(logPath, pytestReportPath)
	settings.hideNonTestQueries = hideNonTestQueries
	settings.logQueryText = showQueryText
	if !verbose {
		settings.logger = nil
	}
	return settings
}

func (s *Settings) GetOutputFilePath(suffix string) string {
	return filepath.Join(s.outputDirPath, s.outputFileBaseName+suffix+s.logFileExtension)
}
