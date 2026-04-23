package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/yulintan/k8s-mcp-server/internal/k8s"
	"github.com/yulintan/k8s-mcp-server/internal/tools"
	"gopkg.in/yaml.v3"
)

const version = "0.1.0"

type logLevel int

const (
	levelDebug logLevel = iota
	levelInfo
	levelWarn
	levelError
)

func parseLogLevel(raw string) logLevel {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return levelDebug
	case "warn", "warning":
		return levelWarn
	case "error":
		return levelError
	default:
		return levelInfo
	}
}

func setupLogger() (*os.File, logLevel, error) {
	level := parseLogLevel(os.Getenv("K8S_MCP_LOG_LEVEL"))
	writers := []io.Writer{os.Stderr}
	logFilePath := strings.TrimSpace(os.Getenv("K8S_MCP_LOG_FILE"))

	if logFilePath != "" {
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, level, fmt.Errorf("open log file %q: %w", logFilePath, err)
		}
		writers = append(writers, f)
		log.SetOutput(io.MultiWriter(writers...))
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
		return f, level, nil
	}

	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	return nil, level, nil
}

func logf(current logLevel, target logLevel, format string, args ...any) {
	if target < current {
		return
	}

	label := "INFO"
	switch target {
	case levelDebug:
		label = "DEBUG"
	case levelWarn:
		label = "WARN"
	case levelError:
		label = "ERROR"
	}

	log.Printf("[%s] %s", label, fmt.Sprintf(format, args...))
}

func effectiveKubeconfigPath(flagValue string) string {
	if strings.TrimSpace(flagValue) != "" {
		return flagValue
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.kube/config"
	}
	return filepath.Join(home, ".kube", "config")
}

func loadConfigYAML(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}

	var cfg map[string]string
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	for key, value := range cfg {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from %s: %w", key, path, err)
		}
	}
	return nil
}

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("%s:%d: expected KEY=VALUE", path, lineNum)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("%s:%d: empty key", path, lineNum)
		}

		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("%s:%d: set %s: %w", path, lineNum, key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

func main() {
	port := flag.Int("port", 0, "HTTP/SSE port. 0 = stdio mode (default, for Claude Desktop/Cursor/VS Code).")
	kubeconfigPath := flag.String("kubeconfig", "", "Path to kubeconfig file. Default: ~/.kube/config.")
	flag.Parse()

	if err := loadConfigYAML("config.yml"); err != nil {
		fmt.Fprintf(os.Stderr, "config load error: %v\n", err)
		os.Exit(1)
	}
	if err := loadDotEnv(".env"); err != nil {
		fmt.Fprintf(os.Stderr, "dotenv load error: %v\n", err)
		os.Exit(1)
	}

	logFile, currentLevel, err := setupLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger setup error: %v\n", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	effectiveKubeconfig := effectiveKubeconfigPath(*kubeconfigPath)
	logf(currentLevel, levelInfo, "starting k8s-mcp-server v%s", version)
	logf(currentLevel, levelDebug, "log level=%q log_file=%q", strings.TrimSpace(os.Getenv("K8S_MCP_LOG_LEVEL")), strings.TrimSpace(os.Getenv("K8S_MCP_LOG_FILE")))
	logf(currentLevel, levelDebug, "kubeconfig=%s", effectiveKubeconfig)

	cm := k8s.NewClientManager(*kubeconfigPath)

	s := server.NewMCPServer(
		"k8s-mcp-server",
		version,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	tools.RegisterAllTools(s, cm)

	if *port == 0 {
		// Stdio mode: used by Claude Desktop, Cursor, VS Code Continue, etc.
		logf(currentLevel, levelInfo, "transport=stdio")
		logf(currentLevel, levelDebug, "stdio mode uses stdout for MCP messages and stderr for logs")
		if logFile != nil {
			logf(currentLevel, levelInfo, "writing logs to %s", logFile.Name())
		}
		if err := server.ServeStdio(s); err != nil {
			logf(currentLevel, levelError, "stdio server error: %v", err)
			fmt.Fprintf(os.Stderr, "stdio server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// HTTP/SSE mode: for web-based clients.
		addr := fmt.Sprintf(":%d", *port)
		baseURL := fmt.Sprintf("http://localhost:%d", *port)
		sseServer := server.NewSSEServer(s, server.WithBaseURL(baseURL))
		logf(currentLevel, levelInfo, "transport=sse addr=%s base_url=%s", addr, baseURL)
		logf(currentLevel, levelDebug, "sse debug: server_name=%q version=%s kubeconfig=%s", "k8s-mcp-server", version, effectiveKubeconfig)
		if logFile != nil {
			logf(currentLevel, levelInfo, "writing logs to %s", logFile.Name())
		}
		if err := sseServer.Start(addr); err != nil {
			logf(currentLevel, levelError, "SSE server error: %v", err)
			fmt.Fprintf(os.Stderr, "SSE server error: %v\n", err)
			os.Exit(1)
		}
	}
}
