package parser

import (
	"fmt"
	"github.com/blastrain/vitess-sqlparser/tidbparser/ast"
	"github.com/blastrain/vitess-sqlparser/tidbparser/dependency/mysql"
	"github.com/blastrain/vitess-sqlparser/tidbparser/parser"
	"mybatis-plus-generator/internal/model"
	"strings"
)

// MySQLParser 实现了 Parser 接口，用于解析 MySQL DDL
type MySQLParser struct{}

func (p *MySQLParser) Parse(sql string) (model.TableInfo, error) {
	stmtNodes, err := parser.New().Parse(sql, mysql.DefaultCharset, "")
	if err != nil {
		return model.TableInfo{}, fmt.Errorf("failed to parse MySQL SQL: %w", err)
	}

	if len(stmtNodes) == 0 {
		return model.TableInfo{}, fmt.Errorf("no SQL statement found")
	}

	var createTableStmt *ast.CreateTableStmt
	for _, stmtNode := range stmtNodes {
		if stmt, ok := stmtNode.(*ast.CreateTableStmt); ok {
			createTableStmt = stmt
			break
		}
	}

	if createTableStmt == nil {
		return model.TableInfo{}, fmt.Errorf("no CREATE TABLE statement found")
	}

	tableName := createTableStmt.Table.Name.String()
	fields := make([]model.Field, 0, len(createTableStmt.Cols))

	// 查找主键列
	primaryKeys := make(map[string]bool)
	for _, cons := range createTableStmt.Constraints {
		if cons.Tp == ast.ConstraintPrimaryKey {
			for _, key := range cons.Keys {
				primaryKeys[key.Column.Name.L] = true
			}
		}
	}

	for _, col := range createTableStmt.Cols {
		fieldName := col.Name.Name.String()
		fieldType := col.Tp.InfoSchemaStr()
		comment := ""

		for _, opt := range col.Options {
			if opt.Tp == ast.ColumnOptionComment {
				comment = opt.Expr.GetDatum().GetString()
				break
			}
		}

		isId := primaryKeys[strings.ToLower(fieldName)]
		// 兼容旧的逻辑，如果有名为id的列也视为主键
		if !isId && strings.ToLower(fieldName) == "id" {
			isId = true
		}

		fields = append(fields, model.Field{
			Name:     fieldName,
			Type:     fieldType,
			JavaType: DefaultTypeMapper.Map(fieldName, "mysql"),
			Comment:  comment,
			IsId:     isId,
		})
	}

	return model.TableInfo{TableName: tableName, Fields: fields}, nil
}
