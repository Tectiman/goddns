package log

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
)

func Init(logOutput string) error {
	// 根据 logOutput 参数决定日志输出方式
	if logOutput != "" && logOutput != "shell" {
		// 尝试打开日志文件
		file, err := os.OpenFile(logOutput, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		log.SetOutput(file)
		isLogTerminal = false
	} else {
		// 输出到终端
		log.SetOutput(os.Stdout)
		isLogTerminal = isStdoutTerminal()
	}

	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	return nil
}

func SetupDefaultLogger() {
	// 判断标准输出是否为终端
	isLogTerminal = isStdoutTerminal()

	// Set log format with timestamp but no extra prefixes
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

// Info logs informational messages
var (
	isLogTerminal bool
	colorReset    = "\033[0m"
	colorRed      = "\033[31m"
	colorBlue     = "\033[34m"
	colorCyan     = "\033[36m"
	colorGreen    = "\033[32m"
	colorYellow   = "\033[33m"
)

// 敏感信息脱敏正则
var (
	// 匹配 API Token (至少 20 位字母数字)
	tokenRegex = regexp.MustCompile(`(?i)(?:token|api[_-]?key|secret)[\s:=]+['"]?([a-zA-Z0-9_-]{20,})['"]?`)
	// 匹配 AccessKey (如阿里云 LTAI 开头)
	accessKeyRegex = regexp.MustCompile(`(?i)(?:access[_-]?key[_-]?id)[\s:=]+['"]?([a-zA-Z0-9]{12,})['"]?`)
)

// sanitizeMessage 脱敏敏感信息
func sanitizeMessage(msg string) string {
	// 脱敏 API Token
	msg = tokenRegex.ReplaceAllStringFunc(msg, func(s string) string {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			parts = strings.SplitN(s, ":", 2)
		}
		if len(parts) == 2 {
			return parts[0] + "=***REDACTED***"
		}
		return "***REDACTED***"
	})

	// 脱敏 AccessKey
	msg = accessKeyRegex.ReplaceAllStringFunc(msg, func(s string) string {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			parts = strings.SplitN(s, ":", 2)
		}
		if len(parts) == 2 {
			return parts[0] + "=***REDACTED***"
		}
		return "***REDACTED***"
	})

	return msg
}

func isTerminal() bool {
	// 只在类 Unix 下简单判断
	return runtime.GOOS != "windows" && os.Getenv("TERM") != "" && os.Getenv("TERM") != "dumb"
}

func isStdoutTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func colorWrap(s, color string) string {
	if isLogTerminal {
		return color + s + colorReset
	}
	return s
}

func Info(format string, args ...interface{}) {
	msg := sanitizeMessage(fmt.Sprintf(format, args...))
	log.Printf("%s %s", colorWrap("[INFO]", colorCyan), msg)
}

func Error(format string, args ...interface{}) {
	msg := sanitizeMessage(fmt.Sprintf(format, args...))
	log.Printf("%s %s", colorWrap("[ERROR]", colorRed), msg)
}

func Success(format string, args ...interface{}) {
	msg := sanitizeMessage(fmt.Sprintf(format, args...))
	log.Printf("%s %s", colorWrap("[SUCCESS]", colorGreen), msg)
}

func Fatal(format string, args ...interface{}) {
	msg := sanitizeMessage(fmt.Sprintf(format, args...))
	log.Printf("%s %s", colorWrap("[FATAL]", colorRed), msg)
	os.Exit(1)
}

func Warning(format string, args ...interface{}) {
	msg := sanitizeMessage(fmt.Sprintf(format, args...))
	log.Printf("%s %s", colorWrap("[WARNING]", colorYellow), msg)
}
