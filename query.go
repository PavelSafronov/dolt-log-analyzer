package main

import (
	"fmt"
	"github.com/dolthub/go-mysql-server/sql"
	"strings"
)

type Query struct {
	LineNumber int
	Text       string
	Node       sql.Node
	NodeDebug  string
	TestId     string
	TestFailed bool
	PyTestName string
	Error      string
}

type QueryCollection struct {
	All           []Query
	TestQueries   []Query
	ByTestId      map[string][]Query
	ByDebugString map[string][]Query
}

func NewQueryCollection() QueryCollection {
	return QueryCollection{
		All:           make([]Query, 0),
		ByTestId:      make(map[string][]Query),
		ByDebugString: make(map[string][]Query),
	}
}

func (c *QueryCollection) Add(query Query) {
	c.All = append(c.All, query)
	if query.TestId != "" {
		c.TestQueries = append(c.TestQueries, query)
	}
	c.ByTestId[query.TestId] = append(c.ByTestId[query.TestId], query)
	nodeDebugString := sql.DebugString(query.Node)
	c.ByDebugString[nodeDebugString] = append(c.ByDebugString[nodeDebugString], query)
}

func (q *Query) String(logQueryText bool) string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("Line %d\n", q.LineNumber))
	if q.TestId != "" {
		if q.TestFailed {
			sb.WriteString(fmt.Sprintf("FAILED TEST: %s / %s\n", q.TestId, q.PyTestName))
		} else {
			sb.WriteString(fmt.Sprintf("Passing test: %s / %s\n", q.TestId, q.PyTestName))
		}
	}
	if logQueryText {
		sb.WriteString(fmt.Sprintf("Query:\n%s\n", q.Text))
	}
	if q.Node == nil {
		sb.WriteString("Query tree: nil\n")
	} else {
		sb.WriteString(fmt.Sprintf("Query tree:\n%s\n", sql.DebugString(q.Node)))
	}
	if q.Error != "" {
		sb.WriteString(fmt.Sprintf("Query error: %s\n", q.Error))
	}

	return sb.String()
}
