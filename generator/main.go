package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host            string   `yaml:"host"`
	ProjectID       string   `yaml:"project_id"`
	Environment     string   `yaml:"environment"`
	PollingInterval string   `yaml:"polling_interval"`
	RootFolder      string   `yaml:"root_folder"`
	Services        []string `yaml:"services"`
}

const projectHomepage = "https://github.com/YewFence/infisical-agent"

func main() {
	var (
		servicesFile string
		templateFile string
		outputFile   string
	)

	flag.StringVar(&servicesFile, "services", "config.yaml", "服务配置文件路径")
	flag.StringVar(&templateFile, "template", "config.yaml.tmpl", "模板文件路径")
	flag.StringVar(&outputFile, "output", "config-no-manually-edit.yaml", "输出文件路径")
	flag.Parse()

	// 读取服务配置
	config, err := loadConfig(servicesFile)
	if err != nil {
		exitWithError("读取配置失败", err)
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		exitWithError("配置验证失败", err)
	}

	// 加载模板
	tmpl, err := template.New(filepath.Base(templateFile)).Funcs(template.FuncMap{
		"secretPath": buildSecretPath,
	}).ParseFiles(templateFile)
	if err != nil {
		exitWithError("加载模板失败", err)
	}

	// 生成输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		exitWithError("创建输出文件失败", err)
	}
	defer outFile.Close()

	if err := tmpl.Execute(outFile, config); err != nil {
		exitWithError("渲染模板失败", err)
	}

	absOutput, _ := filepath.Abs(outputFile)
	fmt.Printf("✓ 已生成配置文件: %s\n", absOutput)
	fmt.Printf("  - 项目 ID: %s\n", config.ProjectID)
	fmt.Printf("  - 环境: %s\n", config.Environment)
	if config.RootFolder != "" {
		fmt.Printf("  - 根文件夹: %s\n", config.RootFolder)
	} else {
		fmt.Printf("  - 根文件夹: (无)\n")
	}
	fmt.Printf("  - 服务数量: %d\n", len(config.Services))
	for _, svc := range config.Services {
		fmt.Printf("    • %s\n", svc)
	}

	// 打印符号链接命令供复制
	agentDirName := getExecutableDirName()
	fmt.Println("\n📋 在各服务目录下创建符号链接:")
	for _, svc := range config.Services {
		fmt.Printf("    cd ../%s && ln -sf ../%s/secrets/%s.env .env\n", svc, agentDirName, svc)
	}

	// 打印 env_file 路径供复制
	fmt.Println("\n📋 同时在 docker-compose.yml 中添加 env_file:")
	fmt.Println("    env_file: .env")

	// 打印备份建议
	fmt.Printf("\n💡 建议先备份原 .env 文件（如果有）\n")
	for _, svc := range config.Services {
		fmt.Printf("    mv ../%s/.env ../%s/.env.bak\n", svc, svc)
	}

}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件 %s: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析 YAML: %w", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if config.ProjectID == "" || config.ProjectID == "<your-project-id>" {
		return fmt.Errorf("请在 config.yaml 中设置有效的 project_id")
	}
	if config.Environment == "" {
		return fmt.Errorf("请在 config.yaml 中设置 environment")
	}
	if len(config.Services) == 0 {
		return fmt.Errorf("请在 config.yaml 中至少添加一个服务")
	}
	if config.PollingInterval == "" {
		config.PollingInterval = "300s"
	}
	if config.Host == "" {
		config.Host = "https://app.infisical.com"
	}
	config.RootFolder = normalizeRootFolder(config.RootFolder)
	return nil
}

func normalizeRootFolder(root string) string {
	root = strings.TrimSpace(root)
	root = strings.Trim(root, "/")
	if root == "" {
		return ""
	}
	return "/" + root
}

func buildSecretPath(root, service string) string {
	service = strings.TrimSpace(service)
	service = strings.Trim(service, "/")
	if service == "" {
		return root
	}
	if root == "" {
		return "/" + service
	}
	return root + "/" + service
}

func getExecutableDirName() string {
	exe, err := os.Executable()
	if err != nil {
		return getWorkingDirName()
	}
	dir := filepath.Dir(exe)
	if dir == "." || dir == "" {
		return getWorkingDirName()
	}
	return filepath.Base(dir)
}

func getWorkingDirName() string {
	cwd, err := os.Getwd()
	if err != nil || cwd == "" || cwd == "." {
		return "infisical-agent"
	}
	return filepath.Base(cwd)
}

func exitWithError(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	fmt.Fprintf(os.Stderr, "项目主页: %s\n", projectHomepage)
	os.Exit(1)
}
