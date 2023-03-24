package main

import (
	"fmt"
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/expression"
	"github.com/dolthub/go-mysql-server/sql/parse"
	"github.com/dolthub/go-mysql-server/sql/plan"
	"github.com/dolthub/go-mysql-server/sql/transform"
	"github.com/dolthub/go-mysql-server/sql/types"
	"strings"
)

var stringPlaceholder = "placeholder"

func getPlaceholder(dataType sql.Type) sql.Expression {
	switch dataType.(type) {
	case sql.StringType:
		return expression.NewLiteral(stringPlaceholder, types.Text)
	case sql.NumberType:
		return expression.NewLiteral(1, types.Int64)
	case sql.DecimalType:
		return expression.NewLiteral(1, types.Int64)
	case sql.NullType:
		return expression.NewLiteral(nil, types.Null)
	default:
		panic(fmt.Sprintf("unhandled type: %s", dataType))
	}
}

func ParseQuery(ctx *sql.Context, query string) (sql.Node, error) {
	shouldDebug := strings.Contains(query, "RELEASE SAVEPOINT")
	if shouldDebug {
		fmt.Println("here")
	}
	node, err := parse.Parse(ctx, query)
	if err != nil {
		return nil, err
	}
	debugString := sql.DebugString(node)
	if shouldDebug {
		fmt.Println(debugString)
	}
	node, err = DropExtraneousData(node)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func DropExtraneousData(node sql.Node) (sql.Node, error) {
	newNode, _, err := transform.NodeWithOpaque(node, func(node sql.Node) (sql.Node, transform.TreeIdentity, error) {
		var err error
		var newNode sql.Node
		switch node := node.(type) {
		case *plan.CreateSavepoint:
			newNode = plan.NewCreateSavepoint(stringPlaceholder)
		case *plan.ReleaseSavepoint:
			newNode = plan.NewReleaseSavepoint(stringPlaceholder)
		case *plan.RollbackSavepoint:
			newNode = plan.NewRollbackSavepoint(stringPlaceholder)
		case *plan.UnresolvedTable:
			newNode = plan.NewResolvedDualTable()
		case *plan.TableAlias:
			newNode = plan.NewTableAlias(stringPlaceholder, node.Child)
		case *plan.Project:
			newNode = plan.NewProject([]sql.Expression{expression.NewStar()}, node.Child)
		default:
			fmt.Printf("unhandled node type: %T", node)
			//panic(fmt.Sprintf("unhandled node type: %T", node))
		}

		sameAll := transform.SameTree
		if newNode != nil {
			sameAll = transform.NewTree
		} else {
			newNode = node
		}

		newNode, same, err := transform.NodeExprs(newNode, func(e sql.Expression) (sql.Expression, transform.TreeIdentity, error) {
			var newExpr sql.Expression
			switch e := e.(type) {
			case *plan.Subquery:
				query := e.Query
				newQuery, err := DropExtraneousData(query)
				if err != nil {
					return nil, transform.SameTree, err
				}
				newExpr = e.WithQuery(newQuery)
			case *expression.Literal:
				newExpr = getPlaceholder(e.Type())
			default:
				return e, transform.SameTree, nil
			}
			if newExpr == nil {
				panic("newExpr is nil")
			}
			return newExpr, transform.NewTree, nil
		})
		if err != nil {
			return nil, transform.SameTree, err
		}

		sameAll = same && sameAll
		return newNode, sameAll, nil
	})
	return newNode, err
}
