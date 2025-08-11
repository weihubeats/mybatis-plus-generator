package handler

import (
	"embed"
	"fmt"
	"mybatis-plus-generator/internal/generator"
	"mybatis-plus-generator/internal/model"
	"mybatis-plus-generator/internal/parser"
	"net/http"
)

//go:embed web/static/index.html
var staticFiles embed.FS

//go:embed all:templates
var templateFiles embed.FS // 我们将使用这个变量

const templateDir = "templates" // 假设模板放在项目根目录的 templates 文件夹下

// GenerateHandler 处理代码生成请求
func GenerateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		content, err := staticFiles.ReadFile("web/static/index.html")
		if err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
		return
	}

	// 1. 获取和验证输入
	sql := r.FormValue("sql")
	dbType := r.FormValue("dbType")
	paths := model.PathConfig{
		DOPath:      r.FormValue("do_path"),
		MapperPath:  r.FormValue("mapper_path"),
		DAOPath:     r.FormValue("dao_path"),
		DAOImplPath: r.FormValue("dao_impl_path"),
		XMLPath:     r.FormValue("xml_path"),
	}

	if sql == "" || dbType == "" || paths.DOPath == "" || paths.MapperPath == "" || paths.DAOPath == "" || paths.DAOImplPath == "" || paths.XMLPath == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// 2. 解析 SQL
	p, err := parser.NewParser(dbType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tableInfo, err := p.Parse(sql)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse SQL: %v", err), http.StatusBadRequest)
		return
	}

	// 3. 准备模板数据
	templateData := generator.PrepareTemplateData(tableInfo, paths)

	// 4. 生成文件
	if err := generator.GenerateFiles(templateData, paths, templateFiles); err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate code: %v", err), http.StatusInternalServerError)
		return
	}

	// 5. 返回成功响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Code generated successfully!"))
}
