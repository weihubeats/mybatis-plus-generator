package parser

import (
	"fmt"
	"mybatis-plus-generator/internal/model"
	"strings"
)

// Parser 是一个可以解析SQL DDL的接口
type Parser interface {
	Parse(sql string) (model.TableInfo, error)
}

func NewParser(dbType string) (Parser, error) {
	switch strings.ToLower(dbType) {
	case "mysql":
		return &MySQLParser{}, nil
	case "postgresql", "postgres":
		return &PostgreSQLParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
