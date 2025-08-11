package parser

import (
	"fmt"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"mybatis-plus-generator/internal/model"
	"strings"
)

// PostgreSQLParser 实现了 Parser 接口，用于解析 PostgreSQL DDL
type PostgreSQLParser struct{}

func (p *PostgreSQLParser) Parse(sql string) (model.TableInfo, error) {
	result, err := pg_query.Parse(sql)
	if err != nil {
		return model.TableInfo{}, fmt.Errorf("failed to parse PostgreSQL SQL: %w", err)
	}

	var tableInfo model.TableInfo
	// 按表存储列注释: map[tableName]map[columnName]comment
	columnComments := make(map[string]map[string]string)
	// 按表存储表注释: map[tableName]string
	tableComments := make(map[string]string)
	// 按表存储主键: map[tableName]map[columnName]bool
	primaryKeys := make(map[string]map[string]bool)

	// --- 第一遍遍历：收集所有注释和主键信息，并按表名归类 ---
	for _, stmt := range result.GetStmts() {
		// 1. 解析注释语句 (COMMENT ON)
		if commentStmt := stmt.GetStmt().GetCommentStmt(); commentStmt != nil {
			comment := commentStmt.GetComment()
			objtype := commentStmt.GetObjtype()

			switch objtype {
			case pg_query.ObjectType_OBJECT_COLUMN:
				if listNode := commentStmt.GetObject().GetList(); listNode != nil {
					objNameParts := listNode.GetItems()
					numParts := len(objNameParts)
					if numParts >= 2 {
						colName := objNameParts[numParts-1].GetString_().GetSval()
						tableName := objNameParts[numParts-2].GetString_().GetSval()

						if _, ok := columnComments[tableName]; !ok {
							columnComments[tableName] = make(map[string]string)
						}
						columnComments[tableName][colName] = comment
					}
				}
			case pg_query.ObjectType_OBJECT_TABLE:
				if rangeVar := commentStmt.GetObject().GetRangeVar(); rangeVar != nil {
					tableName := rangeVar.GetRelname()
					tableComments[tableName] = comment
				}
			}
		}

		// 2. 解析建表语句 (CREATE TABLE) 以收集主键信息
		if createStmt := stmt.GetStmt().GetCreateStmt(); createStmt != nil {
			tableName := createStmt.GetRelation().GetRelname()
			if _, ok := primaryKeys[tableName]; !ok {
				primaryKeys[tableName] = make(map[string]bool)
			}

			for _, constraint := range createStmt.GetConstraints() {
				if c := constraint.GetConstraint(); c != nil && c.GetContype() == pg_query.ConstrType_CONSTR_PRIMARY {
					for _, key := range c.GetKeys() {
						primaryKeys[tableName][key.GetString_().GetSval()] = true
					}
				}
			}
			for _, elt := range createStmt.GetTableElts() {
				if c := elt.GetColumnDef(); c != nil {
					for _, constraint := range c.GetConstraints() {
						if cons := constraint.GetConstraint(); cons != nil && cons.GetContype() == pg_query.ConstrType_CONSTR_PRIMARY {
							primaryKeys[tableName][c.GetColname()] = true
						}
					}
				}
			}
		}
	}

	// --- 第二遍遍历：构建 TableInfo ---
	for _, stmt := range result.GetStmts() {
		if createStmt := stmt.GetStmt().GetCreateStmt(); createStmt != nil {
			tableName := createStmt.GetRelation().GetRelname()
			tableInfo.TableName = tableName

			for _, elt := range createStmt.GetTableElts() {
				if colDef := elt.GetColumnDef(); colDef != nil {
					colName := colDef.GetColname()
					typeName := formatPostgresTypeName(colDef.GetTypeName())

					isId := primaryKeys[tableName][colName]
					if !isId && strings.ToLower(colName) == "id" {
						isId = true
					}

					var comment string
					if tableColumnComments, ok := columnComments[tableName]; ok {
						comment = tableColumnComments[colName]
					}

					field := model.Field{
						Name: colName,
						Type: typeName,
						// **【优化】** 使用新的TypeMapper进行类型转换
						JavaType: DefaultTypeMapper.Map(typeName, "postgresql"),
						Comment:  comment,
						IsId:     isId,
					}
					tableInfo.Fields = append(tableInfo.Fields, field)
				}
			}
			break
		}
	}

	if tableInfo.TableName == "" {
		return model.TableInfo{}, fmt.Errorf("no CREATE TABLE statement found in SQL")
	}

	return tableInfo, nil
}

func formatPostgresTypeName(typeName *pg_query.TypeName) string {
	var parts []string
	for _, name := range typeName.GetNames() {
		parts = append(parts, name.GetString_().GetSval())
	}

	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "" // Or some other default
}
