package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host            string   `yaml:"host"`
	ProjectID       string   `yaml:"project_id"`
	Environment     string   `yaml:"environment"`
	PollingInterval string   `yaml:"polling_interval"`
	Services        []string `yaml:"services"`
}

func main() {
	var (
		servicesFile string
		templateFile string
		outputFile   string
	)

	flag.StringVar(&servicesFile, "services", "services.yaml", "服务配置文件路径")
	flag.StringVar(&templateFile, "template", "config.yaml.tmpl", "模板文件路径")
	flag.StringVar(&outputFile, "output", "config.yaml", "输出文件路径")
	flag.Parse()

	// 读取服务配置
	config, err := loadConfig(servicesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取配置失败: %v\n", err)
		os.Exit(1)
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "配置验证失败: %v\n", err)
		os.Exit(1)
	}

	// 加载模板
	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载模板失败: %v\n", err)
		os.Exit(1)
	}

	// 生成输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建输出文件失败: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	if err := tmpl.Execute(outFile, config); err != nil {
		fmt.Fprintf(os.Stderr, "渲染模板失败: %v\n", err)
		os.Exit(1)
	}

	absOutput, _ := filepath.Abs(outputFile)
	fmt.Printf("✓ 已生成配置文件: %s\n", absOutput)
	fmt.Printf("  - 项目 ID: %s\n", config.ProjectID)
	fmt.Printf("  - 环境: %s\n", config.Environment)
	fmt.Printf("  - 服务数量: %d\n", len(config.Services))
	for _, svc := range config.Services {
		fmt.Printf("    • %s\n", svc)
	}

	// 打印符号链接命令供复制
	fmt.Println("\n📋 在各服务目录下创建符号链接:")
	for _, svc := range config.Services {
		fmt.Printf("    cd ../%s && ln -sf ../infisical-agent/secrets/%s.env .env\n", svc, svc)
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
		return fmt.Errorf("请在 services.yaml 中设置有效的 project_id")
	}
	if config.Environment == "" {
		return fmt.Errorf("请在 services.yaml 中设置 environment")
	}
	if len(config.Services) == 0 {
		return fmt.Errorf("请在 services.yaml 中至少添加一个服务")
	}
	if config.PollingInterval == "" {
		config.PollingInterval = "300s"
	}
	if config.Host == "" {
		config.Host = "https://app.infisical.com"
	}
	return nil
}
