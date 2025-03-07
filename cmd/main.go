package main

import (
	"fmt"
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
	doPath := r.FormValue("do_path")
	mapperPath := r.FormValue("mapper_path")
	daoPath := r.FormValue("dao_path")
	daoImplPath := r.FormValue("dao_impl_path")
	xmlPath := r.FormValue("xml_path")

	// 验证必填字段
	if sql == "" || doPath == "" || mapperPath == "" || daoPath == "" || daoImplPath == "" || xmlPath == "" {
		http.Error(w, "SQL 或路径不能为空", http.StatusBadRequest)
		return
	}

	// 解析SQL获取表信息
	tableInfo, err := parseSQL(sql)
	if err != nil {
		http.Error(w, fmt.Sprintf("解析SQL失败: %v", err), http.StatusBadRequest)
		return
	}

	// 提取包名
	doPackage := extractPackageName(doPath)
	mapperPackage := extractPackageName(mapperPath)
	daoPackage := extractPackageName(daoPath)

	// 准备模板数据
	data := TemplateData{
		DOPackage:        doPackage,
		MapperPackage:    mapperPackage,
		DAOPackage:       daoPackage,
		DOClassName:      toCamelCase(tableInfo.TableName) + "DO",
		MapperClassName:  toCamelCase(tableInfo.TableName) + "Mapper",
		DAOClassName:     toCamelCase(tableInfo.TableName) + "DAO",
		DAOImplClassName: toCamelCase(tableInfo.TableName) + "DAOImpl",
		MapperVarName:    strings.ToLower(string(tableInfo.TableName[0])) + tableInfo.TableName[1:] + "Mapper",
		TableName:        tableInfo.TableName,
		Fields:           tableInfo.Fields,
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

func parseSQL(sql string) (TableInfo, error) {
	// 提取表名
	tableNameMatch := tableNameRegex.FindStringSubmatch(sql)
	if len(tableNameMatch) < 2 {
		return TableInfo{}, fmt.Errorf("无法提取表名")
	}
	tableName := tableNameMatch[1]

	// 提取字段定义
	fieldMatches := fieldRegex.FindAllStringSubmatch(sql, -1)
	fieldsMap := make(map[string]Field)
	for _, match := range fieldMatches {
		if len(match) >= 3 {
			fieldName := match[1]
			fieldType := match[2]
			fieldsMap[fieldName] = Field{
				Name:     fieldName,
				Type:     fieldType,
				JavaType: sqlTypeToJavaType(fieldType),
				Comment:  "",
			}
		}
	}

	// 提取字段注释
	commentMatches := commentRegex.FindAllStringSubmatch(sql, -1)
	for _, match := range commentMatches {
		if len(match) >= 3 {
			fieldName := match[1]
			comment := match[2]
			if field, ok := fieldsMap[fieldName]; ok {
				field.Comment = comment
				fieldsMap[fieldName] = field
			}
		}
	}

	// 将 map 转换为 slice
	fields := make([]Field, 0, len(fieldsMap))
	for _, field := range fieldsMap {
		fields = append(fields, field)
	}

	return TableInfo{TableName: tableName, Fields: fields}, nil
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

// 转换为驼峰命名
func toCamelCase(s string) string {
	data := make([]byte, 0, len(s))
	j := false
	k := false
	num := len(s) - 1
	for i := 0; i <= num; i++ {
		d := s[i]
		if k == false && d >= 'A' && d <= 'Z' {
			k = true
		}
		if d >= 'a' && d <= 'z' && (j || k == false) {
			d = d - 32
			j = false
			k = true
		}
		if k && d == '_' && num > i && s[i+1] >= 'a' && s[i+1] <= 'z' {
			j = true
			continue
		}
		data = append(data, d)
	}
	return string(data[:])
}

func sqlTypeToJavaType(sqlType string) string {
	sqlType = strings.ToUpper(sqlType)

	switch {
	case strings.HasPrefix(sqlType, "INT"):
		return "Integer"
	case strings.HasPrefix(sqlType, "BIGINT"):
		return "Long"
	case strings.HasPrefix(sqlType, "DECIMAL"):
		return "BigDecimal"
	case strings.HasPrefix(sqlType, "FLOAT"), strings.HasPrefix(sqlType, "DOUBLE"):
		return "Double"
	case strings.HasPrefix(sqlType, "BOOL"), strings.HasPrefix(sqlType, "TINYINT(1)"):
		return "Boolean"
	case strings.HasPrefix(sqlType, "DATE"):
		return "Date"
	case strings.HasPrefix(sqlType, "TIMESTAMP"), strings.HasPrefix(sqlType, "DATETIME"):
		return "Date"
	default:
		return "String"
	}
}

func extractPackageName(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "java" && i < len(parts)-1 {
			return strings.Join(parts[i+1:], ".")
		}
	}
	return defaultPackage
}
