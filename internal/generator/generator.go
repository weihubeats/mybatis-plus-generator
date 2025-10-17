package generator

import (
	"embed"
	"fmt"
	"mybatis-plus-generator/internal/model"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

// PrepareTemplateData 准备用于渲染模板的所有数据
func PrepareTemplateData(tableInfo model.TableInfo, paths model.PathConfig) model.TemplateData {
	// 提取包名
	doPackage := extractPackageName(paths.DOPath)
	mapperPackage := extractPackageName(paths.MapperPath)
	daoPackage := extractPackageName(paths.DAOPath)
	daoImplPackage := extractPackageName(paths.DAOImplPath)

	// 准备模板数据
	data := model.TemplateData{
		ORM:              paths.ORM,
		DOPackage:        doPackage,
		MapperPackage:    mapperPackage,
		DAOPackage:       daoPackage,
		DAOImplPackage:   daoImplPackage,
		DOClassName:      strcase.ToCamel(tableInfo.TableName) + "DO",
		MapperClassName:  strcase.ToCamel(tableInfo.TableName) + "Mapper",
		MapperVarName:    strcase.ToLowerCamel(tableInfo.TableName) + "Mapper",
		DAOClassName:     strcase.ToCamel(tableInfo.TableName) + "DAO",
		DAOImplClassName: strcase.ToCamel(tableInfo.TableName) + "DAOImpl",
		TableName:        tableInfo.TableName,
		Fields:           tableInfo.ToTemplateFields(),
		MapperNamespace:  mapperPackage + "." + strcase.ToCamel(tableInfo.TableName) + "Mapper",
	}

	// 处理 Imports
	data.Imports = collectImports(tableInfo.Fields)
	data.MybatisPlusImports = getMybatisPlusImports(tableInfo.Fields)

	return data
}

// GenerateFiles 根据模板和数据生成所有代码文件
func GenerateFiles(data model.TemplateData, paths model.PathConfig, templatesFS embed.FS) error {
	// templates/mybatis-flex templates/mybatis-plus
	pathPrefix := model.Path(paths.ORM)
	templateMappings := map[string]string{
		pathPrefix + "/do.tmpl":         filepath.Join(paths.DOPath, data.DOClassName+".java"),
		pathPrefix + "/mapper.tmpl":     filepath.Join(paths.MapperPath, data.MapperClassName+".java"),
		pathPrefix + "/dao.tmpl":        filepath.Join(paths.DAOPath, data.DAOClassName+".java"),
		pathPrefix + "/dao_impl.tmpl":   filepath.Join(paths.DAOImplPath, data.DAOImplClassName+".java"),
		pathPrefix + "/mapper.xml.tmpl": filepath.Join(paths.XMLPath, data.MapperClassName+".xml"),
	}

	for templateName, outputPath := range templateMappings {
		tmplFile, err := template.New(filepath.Base(templateName)).ParseFS(templatesFS, templateName)
		if err != nil {
			return fmt.Errorf("解析嵌入的模板文件 %s 失败: %w", templateName, err)
		}

		if err := generateFromTemplate(tmplFile, data, outputPath); err != nil {
			return fmt.Errorf("failed to generate file from template %s: %w", tmplFile, err)
		}

	}

	return nil
}

func generateFromTemplate(tmpl *template.Template, data interface{}, outputFile string) error {
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("creating directory %s failed: %w", filepath.Dir(outputFile), err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("creating file %s failed: %w", outputFile, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("executing template to file %s failed: %w", outputFile, err)
	}
	return nil
}

func getMybatisPlusImports(fields []model.Field) []string {
	imports := []string{
		"com.baomidou.mybatisplus.annotation.TableName",
	}
	hasId := false
	for _, field := range fields {
		if field.IsId {
			hasId = true
			break
		}
	}
	if hasId {
		imports = append(imports, "com.baomidou.mybatisplus.annotation.IdType")
		imports = append(imports, "com.baomidou.mybatisplus.annotation.TableId")
	}
	sort.Strings(imports)
	return imports
}

func collectImports(fields []model.Field) []string {
	importMap := make(map[string]bool)
	for _, field := range fields {
		if importPath := getJavaTypeImport(field.JavaType); importPath != "" {
			importMap[importPath] = true
		}
	}

	imports := make([]string, 0, len(importMap))
	for imp := range importMap {
		imports = append(imports, imp)
	}
	sort.Strings(imports)
	return imports
}

// --- 辅助函数 ---
// (此处省略了 sqlTypeToJavaType, extractPackageName, getJavaTypeImport 等辅助函数，它们可以原样或稍作修改后放在这个文件或一个独立的 util.go 文件中)
// 比如:
func getJavaTypeImport(javaType string) string {
	typeImportMap := map[string]string{
		"BigDecimal":    "java.math.BigDecimal",
		"LocalDate":     "java.time.LocalDate",
		"LocalTime":     "java.time.LocalTime",
		"LocalDateTime": "java.time.LocalDateTime",
		"Date":          "java.util.Date",
		"UUID":          "java.util.UUID",
		"List":          "java.util.List",
		"ArrayList":     "java.util.ArrayList",
		"Map":           "java.util.Map",
		"HashMap":       "java.util.HashMap",
		"Set":           "java.util.Set",
		"HashSet":       "java.util.HashSet",
		"Duration":      "java.time.Duration",
	}

	// 泛型处理
	if strings.Contains(javaType, "<") {
		baseType := javaType[:strings.Index(javaType, "<")]
		if imp, ok := typeImportMap[baseType]; ok {
			return imp
		}
	}
	return typeImportMap[javaType]

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

func isPackageRoot(part string) bool {
	return part == "com" || part == "org" || part == "net" ||
		part == "io" || part == "cn" || part == "edu"
}

func findLastPackageRoot(parts []string) int {
	for i := len(parts) - 1; i >= 0; i-- {
		if isPackageRoot(parts[i]) {
			return i
		}
	}
	return -1
}

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
