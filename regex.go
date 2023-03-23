package main

import "regexp"

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
