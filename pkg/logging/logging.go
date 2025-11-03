package logging

import (
	"log/slog"
	"os"
	"regexp"
	"strings"
)

var (
	// Default logger instance
	logger *slog.Logger

	// Patterns for detecting sensitive data
	sensitivePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|secret|token|key|auth)[\s]*[:=][\s]*[^\s]+`),
		regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-._~+/]+=*`),
		regexp.MustCompile(`(?i)Basic\s+[A-Za-z0-9+/]+=*`),
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),   // AWS Access Key
		regexp.MustCompile(`[0-9a-zA-Z/+=]{40}`), // AWS Secret Key pattern
	}
)

func init() {
	// Initialize with JSON handler by default
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// Check for debug mode from environment
	if os.Getenv("CLOUD_DEPLOY_DEBUG") == "true" {
		opts.Level = slog.LevelDebug
	}

	logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// SetLogger allows overriding the default logger
func SetLogger(l *slog.Logger) {
	logger = l
}

// GetLogger returns the current logger instance
func GetLogger() *slog.Logger {
	return logger
}

// SanitizeString removes or masks sensitive data from strings
func SanitizeString(s string) string {
	sanitized := s
	for _, pattern := range sensitivePatterns {
		sanitized = pattern.ReplaceAllStringFunc(sanitized, func(match string) string {
			// Extract the key part before the value
			parts := strings.SplitN(match, ":", 2)
			if len(parts) == 2 {
				return parts[0] + ": [REDACTED]"
			}
			parts = strings.SplitN(match, "=", 2)
			if len(parts) == 2 {
				return parts[0] + "=[REDACTED]"
			}
			return "[REDACTED]"
		})
	}
	return sanitized
}

// SanitizeMap creates a sanitized copy of a map, redacting sensitive keys
func SanitizeMap(m map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	sensitiveKeys := map[string]bool{
		"password":          true,
		"secret":            true,
		"token":             true,
		"key":               true,
		"auth":              true,
		"credential":        true,
		"access_key":        true,
		"secret_key":        true,
		"access_key_id":     true,
		"secret_access_key": true,
		"client_secret":     true,
		"api_key":           true,
	}

	for k, v := range m {
		lowerKey := strings.ToLower(k)
		if sensitiveKeys[lowerKey] {
			sanitized[k] = "[REDACTED]"
		} else if strVal, ok := v.(string); ok {
			sanitized[k] = SanitizeString(strVal)
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// Info logs an informational message
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

// InfoContext logs with additional context fields
func InfoContext(msg string, contextFields map[string]interface{}, args ...any) {
	sanitized := SanitizeMap(contextFields)
	allArgs := make([]any, 0, len(args)+len(sanitized)*2)
	allArgs = append(allArgs, args...)
	for k, v := range sanitized {
		allArgs = append(allArgs, k, v)
	}
	logger.Info(msg, allArgs...)
}

// ErrorContext logs an error with additional context fields
func ErrorContext(msg string, contextFields map[string]interface{}, args ...any) {
	sanitized := SanitizeMap(contextFields)
	allArgs := make([]any, 0, len(args)+len(sanitized)*2)
	allArgs = append(allArgs, args...)
	for k, v := range sanitized {
		allArgs = append(allArgs, k, v)
	}
	logger.Error(msg, allArgs...)
}
