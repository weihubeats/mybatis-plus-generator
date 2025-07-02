package parser

import "strings"

var DefaultTypeMapper = NewTypeMapper()

type TypeMapper struct {

	// map[dbType]map[sqlType]javaType
	mapping map[string]map[string]string
}

func NewTypeMapper() *TypeMapper {

	tm := &TypeMapper{
		mapping: make(map[string]map[string]string),
	}
	tm.registerDefaultMappings()
	return tm

}

func (tm *TypeMapper) registerDefaultMappings() {

	mysqlMappings := map[string]string{
		"INT":        "Integer",
		"INTEGER":    "Integer",
		"SMALLINT":   "Integer",
		"MEDIUMINT":  "Integer",
		"TINYINT":    "Integer", // 默认映射为Integer, 特殊情况在Map函数中处理
		"BIGINT":     "Long",
		"DECIMAL":    "BigDecimal",
		"NUMERIC":    "BigDecimal",
		"FLOAT":      "Float",
		"DOUBLE":     "Double",
		"REAL":       "Double",
		"BOOLEAN":    "Boolean",
		"BOOL":       "Boolean",
		"DATE":       "LocalDate",
		"TIME":       "LocalTime",
		"TIMESTAMP":  "LocalDateTime",
		"DATETIME":   "LocalDateTime",
		"CHAR":       "String",
		"VARCHAR":    "String",
		"TEXT":       "String",
		"TINYTEXT":   "String",
		"MEDIUMTEXT": "String",
		"LONGTEXT":   "String",
		"BLOB":       "byte[]",
		"MEDIUMBLOB": "byte[]",
		"LONGBLOB":   "byte[]",
		"BINARY":     "byte[]",
		"VARBINARY":  "byte[]",
	}

	// PostgreSQL Mappings
	postgresMappings := map[string]string{
		"INT":                         "Integer",
		"INTEGER":                     "Integer",
		"SMALLINT":                    "Integer",
		"INT4":                        "Integer",
		"BIGINT":                      "Long",
		"BIGSERIAL":                   "Long",
		"INT8":                        "Long",
		"DECIMAL":                     "BigDecimal",
		"NUMERIC":                     "BigDecimal",
		"REAL":                        "Float",
		"FLOAT4":                      "Float",
		"DOUBLE PRECISION":            "Double",
		"FLOAT8":                      "Double",
		"BOOLEAN":                     "Boolean",
		"BOOL":                        "Boolean",
		"DATE":                        "LocalDate",
		"TIME":                        "LocalTime",
		"TIME WITHOUT TIME ZONE":      "LocalTime",
		"TIME WITH TIME ZONE":         "LocalTime",
		"TIMESTAMP":                   "LocalDateTime",
		"TIMESTAMP WITHOUT TIME ZONE": "LocalDateTime",
		"TIMESTAMP WITH TIME ZONE":    "LocalDateTime",
		"CHAR":                        "String",
		"CHARACTER":                   "String",
		"VARCHAR":                     "String",
		"CHARACTER VARYING":           "String",
		"TEXT":                        "String",
		"BYTEA":                       "byte[]",
		"UUID":                        "UUID",
		"JSON":                        "String",
		"JSONB":                       "String",
		"ARRAY":                       "List",
		"INTERVAL":                    "Duration",
	}

	tm.mapping["mysql"] = mysqlMappings
	tm.mapping["postgresql"] = postgresMappings

}

func (tm *TypeMapper) Map(sqlType, dbType string) string {
	dbType = strings.ToLower(dbType)
	originalSQLType := strings.ToUpper(strings.TrimSpace(sqlType))

	if dbType == "mysql" && strings.HasPrefix(originalSQLType, "TINYINT(1)") {
		return "Boolean"
	}

	baseType := strings.Split(originalSQLType, "(")[0]

	if dbMappings, ok := tm.mapping[dbType]; ok {
		if javaType, ok := dbMappings[baseType]; ok {
			return javaType
		}
	}

	// 如果找不到任何映射，返回默认值
	return "String"

}
