package main

import (
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/parse"
	"github.com/dolthub/go-mysql-server/sql/transform"
)

func ParseQuery(ctx *sql.Context, query string) (sql.Node, error) {
	node, err := parse.Parse(ctx, query)
	if err != nil {
		return nil, err
	}
	node, err = DropExtraneousData(node)
	if err != nil {
		return nil, err
	}
	return node, nil
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
