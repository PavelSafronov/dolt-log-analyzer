package main

import (
	"fmt"
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/expression"
	"github.com/dolthub/go-mysql-server/sql/parse"
	"github.com/dolthub/go-mysql-server/sql/plan"
	"github.com/dolthub/go-mysql-server/sql/transform"
)

func ParseQuery(ctx *sql.Context, query string) (sql.Node, error) {
	shouldDebug := query == "UPDATE `extras_job` SET `created` = '2023-02-03', `last_updated` = '2023-03-23 21:55:59.590578', `_custom_field_data` = '{}', `source` = 'local', `module_name` = 'api_test_job', `job_class_name` = 'APITestJob', `slug` = 'local-api_test_job-apitestjob', `grouping` = 'api_test_job', `name` = 'Job for API Tests', `description` = '', `installed` = 1, `enabled` = 0, `commit_default` = 1, `hidden` = 0, `read_only` = 0, `approval_required` = 0, `soft_time_limit` = 0.0e0, `time_limit` = 0.0e0, `grouping_override` = 0, `name_override` = 0, `description_override` = 0, `commit_default_override` = 0, `hidden_override` = 0, `read_only_override` = 0, `approval_required_override` = 0, `soft_time_limit_override` = 0, `time_limit_override` = 0, `git_repository_id` = NULL, `has_sensitive_variables` = 0, `has_sensitive_variables_override` = 0, `is_job_hook_receiver` = 0, `task_queues` = '[\\\"default\\\", \\\"nonexistent\\\"]', `task_queues_override` = 0 WHERE `extras_job`.`id` = '99de80426576477b9654223abc588108'"
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
		case *plan.UnresolvedTable:
			newNode = plan.NewResolvedDualTable()
		case *plan.Project:
			newNode = plan.NewProject([]sql.Expression{expression.NewStar()}, node.Child)
		default:
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
				switch e.Type().(type) {
				case sql.StringType:
					newExpr = expression.NewLiteral("placeholder", sql.Text)
				case sql.NumberType:
					newExpr = expression.NewLiteral(1, sql.Int64)
				case sql.DecimalType:
					newExpr = expression.NewLiteral(1, sql.Int64)
				case sql.NullType:
					// no-op
					return e, transform.SameTree, nil
				default:
					panic(fmt.Sprintf("unhandled type: %s", e.Type()))
				}
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
