package main

import (
	"encoding/json"
	"fmt"
	"github.com/blastrain/vitess-sqlparser/tidbparser/ast"
	"github.com/blastrain/vitess-sqlparser/tidbparser/dependency/mysql"
	"github.com/blastrain/vitess-sqlparser/tidbparser/parser"
	"github.com/iancoleman/strcase"
	pg_query "github.com/pganalyze/pg_query_go/v4"
	"log"
	"mybatis-plus-generator/configs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type Field struct {
	Name     string // 字段名
	Type     string // SQL 类型
	JavaType string // 对应的 Java 类型
	Comment  string // 字段注释
}

// TableInfo 表示表的信息
type TableInfo struct {
	TableName string  // 表名
	Fields    []Field // 字段列表
}

func (tableInfo *TableInfo) toTemplateFields() []Field {
	newField := make([]Field, len(tableInfo.Fields))
	for i, field := range tableInfo.Fields {
		newField[i] = Field{Name: strcase.ToLowerCamel(field.Name), Type: field.Type, JavaType: field.JavaType, Comment: field.Comment}

	}
	return newField

}

type TemplateData struct {
	DOClassName      string
	MapperClassName  string
	DAOClassName     string
	DAOImplClassName string
	MapperVarName    string
	TableName        string
	Fields           []Field
	DOPackage        string
	MapperPackage    string
	DAOPackage       string
	DAOImplPackage   string
}

type Comment struct {
	Type      string // "line" 或 "block"
	Content   string // 注释内容
	Position  int    // 注释在SQL中的位置
	ColumnRef string // 如果是字段注释，引用的字段名
	TableRef  string // 如果是表注释，引用的表名

}

const (
	defaultPackage = "com.example.default"
)

var (
	// 更灵活的正则表达式，支持多行和不同格式的 CREATE TABLE 语句
	tableNameRegex = regexp.MustCompile(`(?i)create\s+table\s+(?:if\s+not\s+exists\s+)?(?:\w+\.)?['"]?(\w+)['"]?`)
	fieldRegex     = regexp.MustCompile(`(?im)^\s*['"]?(\w+)['"]?\s+([^,\n]+?)(?:\s+constraint|\s+primary|\s+unique|\s+default|\s+references|\s+not null|,|\n|$)`)
	commentRegex   = regexp.MustCompile(`(?i)comment\s+on\s+column\s+\w+\.(\w+)\s+is\s+'([^']*)'`)
)

func main() {
	http.HandleFunc("/", generateHandler)
	address := ":8080"
	fmt.Printf("服务器启动在 http://localhost%s\n", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	// 不是POST请求直接返回静态页面
	if r.Method != http.MethodPost {
		http.ServeFile(w, r, "web/static/index.html")
		return
	}

	// 获取配置
	config := configs.NewConfig()

	// 验证表单数据
	sql := r.FormValue("sql")
	dbType := r.FormValue("dbType")
	doPath := r.FormValue("do_path")
	mapperPath := r.FormValue("mapper_path")
	daoPath := r.FormValue("dao_path")
	daoImplPath := r.FormValue("dao_impl_path")
	xmlPath := r.FormValue("xml_path")

	// 验证必填字段
	if sql == "" || doPath == "" || mapperPath == "" || daoPath == "" || daoImplPath == "" || xmlPath == "" || dbType == "" {
		http.Error(w, "SQL 或路径不能为空", http.StatusBadRequest)
		return
	}

	// 解析SQL获取表信息
	tableInfo, err := parseSQL(sql, "postgresql")
	if err != nil {
		http.Error(w, fmt.Sprintf("解析SQL失败: %v", err), http.StatusBadRequest)
		return
	}

	// 提取包名
	doPackage := extractPackageName(doPath)
	mapperPackage := extractPackageName(mapperPath)
	daoPackage := extractPackageName(daoPath)
	daoImplPackage := extractPackageName(daoImplPath)

	// 准备模板数据
	data := TemplateData{
		DOPackage:        doPackage,
		MapperPackage:    mapperPackage,
		DAOPackage:       daoPackage,
		DAOImplPackage:   daoImplPackage,
		DOClassName:      strcase.ToCamel(tableInfo.TableName) + "DO",
		MapperClassName:  strcase.ToCamel(tableInfo.TableName) + "Mapper",
		DAOClassName:     strcase.ToCamel(tableInfo.TableName) + "DAO",
		DAOImplClassName: strcase.ToCamel(tableInfo.TableName) + "DAOImpl",
		MapperVarName:    strings.ToLower(string(tableInfo.TableName[0])) + tableInfo.TableName[1:] + "Mapper",
		TableName:        tableInfo.TableName,
		Fields:           tableInfo.toTemplateFields(),
	}

	// 定义模板和输出文件的映射
	tmplPath := config.TmplPath
	templates := map[string]string{
		filepath.Join(tmplPath, "do.tmpl"):         filepath.Join(doPath, data.DOClassName+".java"),
		filepath.Join(tmplPath, "mapper.tmpl"):     filepath.Join(mapperPath, data.MapperClassName+".java"),
		filepath.Join(tmplPath, "dao.tmpl"):        filepath.Join(daoPath, data.DAOClassName+".java"),
		filepath.Join(tmplPath, "dao_impl.tmpl"):   filepath.Join(daoImplPath, data.DAOImplClassName+".java"),
		filepath.Join(tmplPath, "mapper.xml.tmpl"): filepath.Join(xmlPath, data.MapperClassName+".xml"),
	}

	// 处理所有模板
	for tmplFile, outputFile := range templates {
		if err := generateFromTemplate(tmplFile, data, outputFile); err != nil {
			http.Error(w, fmt.Sprintf("生成文件失败: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Write([]byte("代码生成成功"))
}

func parseSQL(sql string, dbType string) (TableInfo, error) {
	dbType = strings.ToLower(dbType)

	if dbType == "mysql" {
		return parseMySQLDDL(sql)
	} else if dbType == "postgresql" || dbType == "postgres" {
		return parsePostgreSQLDDL(sql)
	}

	return TableInfo{}, fmt.Errorf("不支持的数据库类型: %s", dbType)
}

func parseMySQLDDL(sql string) (TableInfo, error) {
	p := parser.New()
	stmtNodes, err := p.Parse(sql, mysql.DefaultCharset, "")
	if err != nil {
		return TableInfo{}, fmt.Errorf("解析MySQL SQL失败: %w", err)
	}

	if len(stmtNodes) == 0 {
		return TableInfo{}, fmt.Errorf("没有找到SQL语句")
	}

	// 查找CREATE TABLE语句
	var createTableStmt *ast.CreateTableStmt
	for _, stmtNode := range stmtNodes {
		if stmt, ok := stmtNode.(*ast.CreateTableStmt); ok {
			createTableStmt = stmt
			break
		}
	}

	if createTableStmt == nil {
		return TableInfo{}, fmt.Errorf("没有找到CREATE TABLE语句")
	}

	// 提取表名
	tableName := createTableStmt.Table.Name.String()

	// 提取字段
	fields := make([]Field, 0, len(createTableStmt.Cols))
	for _, col := range createTableStmt.Cols {
		fieldName := col.Name.Name.String()
		fieldType := col.Tp.InfoSchemaStr()
		comment := ""

		// 提取注释
		for _, opt := range col.Options {
			if opt.Tp == ast.ColumnOptionComment {
				comment = opt.Expr.GetDatum().GetString()
				break
			}
		}

		fields = append(fields, Field{
			Name:     fieldName,
			Type:     fieldType,
			JavaType: sqlTypeToJavaType(fieldType, "mysql"),
			Comment:  comment,
		})
	}

	return TableInfo{TableName: tableName, Fields: fields}, nil
}

func parsePostgreSQLDDL(sql string) (TableInfo, error) {

	// 解析 SQL
	result, err := pg_query.Parse(sql)
	if err != nil {
		return TableInfo{}, fmt.Errorf("解析 PostgreSQL SQL 失败: %w", err)
	}

	// 转换为 JSON 以便更容易处理
	jsonResult, err := json.Marshal(result)
	if err != nil {
		return TableInfo{}, fmt.Errorf("JSON 转换失败: %w", err)
	}
	// 解析 JSON
	var parsedJSON map[string]interface{}

	if err := json.Unmarshal(jsonResult, &parsedJSON); err != nil {
		return TableInfo{}, fmt.Errorf("JSON 解析失败: %w", err)
	}
	// 提取表名和字段
	tableName := ""

	fields := []Field{}
	stmts, ok := parsedJSON["stmts"].([]interface{})
	if !ok || len(stmts) == 0 {
		return TableInfo{}, fmt.Errorf("未找到语句")
	}

	// 遍历每个语句
	for _, stmt := range stmts {
		stmtMap, ok := stmt.(map[string]interface{})
		if !ok {
			continue
		}
		stmtObj, ok := stmtMap["stmt"].(map[string]interface{})
		if !ok {
			continue
		}
		// 查找 CreateStmt
		node, _ := stmtObj["Node"].(map[string]interface{})

		createStmt, ok := node["CreateStmt"].(map[string]interface{})

		if !ok {
			continue
		}
		// 提取表名
		relation, ok := createStmt["relation"].(map[string]interface{})
		if ok {
			relname, ok := relation["relname"].(string)
			if ok {
				tableName = relname
			}
		}

		// 提取字段
		tableElts, ok := createStmt["table_elts"].([]interface{})

		if !ok {
			continue
		}

		for _, elt := range tableElts {
			eltMap, ok := elt.(map[string]interface{})
			if !ok {
				continue
			}

			// 提取 ColumnDef
			m := eltMap["Node"].(map[string]interface{})
			columnDef, ok := m["ColumnDef"].(map[string]interface{})
			if !ok {
				continue
			}
			// 提取列名
			colname, ok := columnDef["colname"].(string)
			if !ok {
				continue
			}

			// 提取类型
			typeName := "unknown"
			typeNameObj, ok := columnDef["type_name"].(map[string]interface{})
			if ok {
				names, ok := typeNameObj["names"].([]interface{})
				if ok && len(names) > 0 {
					lastName := names[len(names)-1]
					lastNameMap, ok := lastName.(map[string]interface{})
					if ok {
						m2 := lastNameMap["Node"].(map[string]interface{})
						stringObj, ok := m2["String_"].(map[string]interface{})
						if ok {
							typeName, _ = stringObj["sval"].(string)
						}

					}

				}

			}
			// 尝试提取注释 - 在 PostgreSQL 中，注释通常是分开的语句

			// 这里我们先提取不到，需要另外处理

			comment := ""
			// 添加字段
			fields = append(fields, Field{
				// todo 不应该在此处进行驼峰转换
				Name:     colname,
				Type:     typeName,
				JavaType: sqlTypeToJavaType(typeName, "postgresql"),
				Comment:  comment,
			})

		}

	}

	if tableName == "" {
		return TableInfo{}, fmt.Errorf("未找到表名")

	}

	comments := parsePostgreSQLComments(sql)
	columnComments := make(map[string]string)
	for _, comment := range comments {
		if comment.ColumnRef != "" && comment.TableRef == tableName {

			columnComments[comment.ColumnRef] = comment.Content

		}

	}

	for i, field := range fields {

		// 将字段名转回原始的SQL字段名以匹配注释

		originalFieldName := field.Name

		if comment, ok := columnComments[originalFieldName]; ok {

			fields[i].Comment = comment

		}

	}

	return TableInfo{TableName: tableName, Fields: fields}, nil

}

func parsePostgreSQLComments(sql string) []Comment {
	comments := []Comment{}
	// 1. 解析单行注释 (--开始的注释)
	lineCommentRegex := regexp.MustCompile(`--(.*)`)
	lineMatches := lineCommentRegex.FindAllStringSubmatchIndex(sql, -1)
	for _, match := range lineMatches {
		if len(match) >= 4 {
			start, end := match[2], match[3]
			commentContent := strings.TrimSpace(sql[start:end])
			comments = append(comments, Comment{
				Type:     "line",
				Content:  commentContent,
				Position: match[0],
			})
		}
	}

	// 2. 解析块注释 (/* */包围的注释)
	blockCommentRegex := regexp.MustCompile(`/\*([\s\S]*?)\*/`)
	blockMatches := blockCommentRegex.FindAllStringSubmatchIndex(sql, -1)
	for _, match := range blockMatches {
		if len(match) >= 4 {
			start, end := match[2], match[3]
			commentContent := strings.TrimSpace(sql[start:end])
			comments = append(comments, Comment{
				Type:     "block",
				Content:  commentContent,
				Position: match[0],
			})
		}
	}

	// 3. 解析COMMENT ON语句
	commentOnRegex := regexp.MustCompile(`(?i)COMMENT\s+ON\s+COLUMN\s+(\w+)\.(\w+)\s+IS\s+'(.*?)'`)
	onMatches := commentOnRegex.FindAllStringSubmatch(sql, -1)
	for _, match := range onMatches {
		if len(match) >= 4 {
			tableRef := match[1]
			columnRef := match[2]
			commentContent := match[3]
			comments = append(comments, Comment{
				Type:      "comment_on",
				Content:   commentContent,
				TableRef:  tableRef,
				ColumnRef: columnRef,
			})

		}

	}

	// 解析表注释

	tableCommentRegex := regexp.MustCompile(`(?i)COMMENT\s+ON\s+TABLE\s+(\w+)\s+IS\s+'(.*?)'`)
	tableMatches := tableCommentRegex.FindAllStringSubmatch(sql, -1)
	for _, match := range tableMatches {
		if len(match) >= 3 {
			tableRef := match[1]
			commentContent := match[2]
			comments = append(comments, Comment{
				Type:     "table_comment",
				Content:  commentContent,
				TableRef: tableRef,
			})
		}

	}

	return comments

}

func generateFromTemplate(tmplFile string, data interface{}, outputFile string) error {

	// 解析模板
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		return fmt.Errorf("解析模板 %s 失败: %w", tmplFile, err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("创建目录 %s 失败: %w", filepath.Dir(outputFile), err)
	}

	// 创建输出文件
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建文件 %s 失败: %w", outputFile, err)
	}
	defer file.Close()

	// 执行模板
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("执行模板到文件 %s 失败: %w", outputFile, err)
	}

	return nil
}

func sqlTypeToJavaType(sqlType string, dbType string) string {
	sqlType = strings.ToUpper(strings.TrimSpace(sqlType))
	dbType = strings.ToLower(dbType)

	// 处理带长度的类型，如 VARCHAR(255)
	typeParts := strings.Split(sqlType, "(")
	baseType := typeParts[0]

	if dbType == "mysql" {
		switch baseType {
		case "INT", "INTEGER", "SMALLINT", "MEDIUMINT", "TINYINT":
			if baseType == "TINYINT" && len(typeParts) > 1 && strings.TrimRight(typeParts[1], ")") == "1" {
				return "Boolean"
			}
			return "Integer"
		case "BIGINT":
			return "Long"
		case "DECIMAL", "NUMERIC":
			return "BigDecimal"
		case "FLOAT", "DOUBLE", "REAL":
			return "Double"
		case "BOOLEAN", "BOOL":
			return "Boolean"
		case "DATE":
			return "LocalDate"
		case "TIME":
			return "LocalTime"
		case "TIMESTAMP", "DATETIME":
			return "LocalDateTime"
		case "CHAR", "VARCHAR", "TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT":
			return "String"
		case "BLOB", "MEDIUMBLOB", "LONGBLOB", "BINARY", "VARBINARY":
			return "byte[]"
		default:
			return "String"
		}
	} else if dbType == "postgresql" || dbType == "postgres" {
		switch baseType {
		case "INT", "INTEGER", "SMALLINT":
			return "Integer"
		case "BIGINT":
			return "Long"
		case "DECIMAL", "NUMERIC":
			return "BigDecimal"
		case "REAL", "DOUBLE PRECISION":
			return "Double"
		case "BOOLEAN":
			return "Boolean"
		case "DATE":
			return "LocalDate"
		case "TIME", "TIME WITHOUT TIME ZONE", "TIME WITH TIME ZONE":
			return "LocalTime"
		case "TIMESTAMP", "TIMESTAMP WITHOUT TIME ZONE", "TIMESTAMP WITH TIME ZONE":
			return "LocalDateTime"
		case "CHAR", "CHARACTER", "VARCHAR", "CHARACTER VARYING", "TEXT":
			return "String"
		case "BYTEA":
			return "byte[]"
		case "UUID":
			return "UUID"
		case "JSON", "JSONB":
			return "String" // 或者使用特定的JSON库
		case "ARRAY":
			return "List<Object>" // 或者更具体的类型
		case "INTERVAL":
			return "Duration"
		default:
			return "String"
		}
	}

	return "String" // 默认
}

func extractPackageName(path string) string {
	path = filepath.ToSlash(path) // 统一路径分隔符
	parts := strings.Split(path, "/")
	defaultPackage := "default.package"

	// 1. 尝试匹配标准Maven/Gradle源码目录结构
	for i := 0; i < len(parts)-2; i++ {
		if parts[i] == "src" {
			if parts[i+1] == "main" && parts[i+2] == "java" {
				return joinValidParts(parts[i+3:])
			}
			if parts[i+1] == "test" && parts[i+2] == "java" {
				return joinValidParts(parts[i+3:])
			}
		}
	}

	// 2. 尝试匹配Java目录后的包根
	for i, part := range parts {
		if part == "java" && i < len(parts)-1 {
			if isPackageRoot(parts[i+1]) {
				return joinValidParts(parts[i+1:])
			}
		}
	}

	// 3. 直接查找包根目录（com/org等）
	for i, part := range parts {
		if isPackageRoot(part) {
			return joinValidParts(parts[i:])
		}
	}

	// 4. 回退策略：找到最后一个存在的包根目录
	if lastIndex := findLastPackageRoot(parts); lastIndex != -1 {
		return joinValidParts(parts[lastIndex:])
	}

	return defaultPackage
}

// 判断是否是包根目录
func isPackageRoot(part string) bool {
	return part == "com" || part == "org" || part == "net" ||
		part == "io" || part == "cn" || part == "edu"
}

// 查找最后一个包根目录位置
func findLastPackageRoot(parts []string) int {
	for i := len(parts) - 1; i >= 0; i-- {
		if isPackageRoot(parts[i]) {
			return i
		}
	}
	return -1
}

// 拼接有效路径并过滤非法字符
func joinValidParts(parts []string) string {
	validParts := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || strings.ContainsAny(p, " -$") {
			continue
		}
		validParts = append(validParts, p)
	}
	if len(validParts) == 0 {
		return "default.package"
	}
	return strings.Join(validParts, ".")
}
