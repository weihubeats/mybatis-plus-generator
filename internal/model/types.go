package model

import "github.com/iancoleman/strcase"

// Field 表示数据库表的字段信息
type Field struct {
	Name     string // 字段名 (原始名称)
	Type     string // SQL 类型
	JavaType string // 对应的 Java 类型
	Comment  string // 字段注释
	IsId     bool   // 是否为主键ID字段
}

// TableInfo 表示表的信息
type TableInfo struct {
	TableName string  // 表名
	Fields    []Field // 字段列表
}

// ToTemplateFields 将字段名转换为小驼峰命名法，用于模板渲染
func (ti *TableInfo) ToTemplateFields() []Field {

	newFields := make([]Field, len(ti.Fields))
	for i, field := range ti.Fields {
		newFields[i] = Field{
			Name:     strcase.ToLowerCamel(field.Name),
			Type:     field.Type,
			JavaType: field.JavaType,
			Comment:  field.Comment,
			IsId:     field.IsId,
		}
	}
	return newFields
}

// TemplateData 是传递给Go模板的最终数据结构
type TemplateData struct {
	DOClassName        string
	MapperClassName    string
	DAOClassName       string
	DAOImplClassName   string
	MapperVarName      string
	TableName          string
	Fields             []Field
	DOPackage          string
	MapperPackage      string
	DAOPackage         string
	DAOImplPackage     string
	Imports            []string
	MapperNamespace    string
	MybatisPlusImports []string
}

// PathConfig 存储用户提供的所有路径
type PathConfig struct {
	DOPath      string
	MapperPath  string
	DAOPath     string
	DAOImplPath string
	XMLPath     string
}
