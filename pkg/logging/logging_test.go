package logging

import (
	"strings"
	"testing"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "AWS access key",
			input:    "Using access key AKIAIOSFODNN7EXAMPLE",
			expected: "Using access key [REDACTED]",
		},
		{
			name:     "password in config",
			input:    "password: secretpassword123",
			expected: "password: [REDACTED]",
		},
		{
			name:     "token with equals",
			input:    "token=abc123def456",
			expected: "token=[REDACTED]",
		},
		{
			name:     "bearer token",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Authorization: [REDACTED]",
		},
		{
			name:     "basic auth",
			input:    "Authorization: Basic dXNlcjpwYXNzd29yZA==",
			expected: "Authorization: [REDACTED]",
		},
		{
			name:     "safe string",
			input:    "Deploying application to production",
			expected: "Deploying application to production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if !strings.Contains(result, "[REDACTED]") && strings.Contains(tt.expected, "[REDACTED]") {
				t.Errorf("SanitizeString() failed to redact sensitive data\nInput:    %s\nGot:      %s\nExpected: %s",
					tt.input, result, tt.expected)
			}
			// For safe strings, should be unchanged
			if !strings.Contains(tt.expected, "[REDACTED]") && result != tt.expected {
				t.Errorf("SanitizeString() modified safe string\nInput:    %s\nGot:      %s\nExpected: %s",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeMap(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]interface{}
		checkKey     string
		wantRedacted bool
	}{
		{
			name: "password key",
			input: map[string]interface{}{
				"username": "admin",
				"password": "secret123",
			},
			checkKey:     "password",
			wantRedacted: true,
		},
		{
			name: "access_key_id",
			input: map[string]interface{}{
				"region":        "us-east-1",
				"access_key_id": "AKIAIOSFODNN7EXAMPLE",
			},
			checkKey:     "access_key_id",
			wantRedacted: true,
		},
		{
			name: "secret in value",
			input: map[string]interface{}{
				"config": "password=secret123",
			},
			checkKey:     "config",
			wantRedacted: true,
		},
		{
			name: "safe values",
			input: map[string]interface{}{
				"region": "us-west-2",
				"count":  5,
			},
			checkKey:     "region",
			wantRedacted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeMap(tt.input)
			value, ok := result[tt.checkKey]
			if !ok {
				t.Errorf("SanitizeMap() lost key %s", tt.checkKey)
				return
			}

			strValue, isString := value.(string)
			if tt.wantRedacted {
				if !isString {
					t.Errorf("SanitizeMap() didn't convert sensitive value to string")
					return
				}
				if !strings.Contains(strValue, "[REDACTED]") {
					t.Errorf("SanitizeMap() didn't redact sensitive key %s, got: %v", tt.checkKey, value)
				}
			} else {
				// Safe values should be unchanged
				if strValue == "[REDACTED]" {
					t.Errorf("SanitizeMap() incorrectly redacted safe key %s", tt.checkKey)
				}
			}
		})
	}
}

func TestSanitizeMapNestedStructures(t *testing.T) {
	input := map[string]interface{}{
		"app_name":   "my-app",
		"secret_key": "super-secret",
		"token":      "abc123",
		"config": map[string]interface{}{
			"db_host": "localhost",
		},
	}

	result := SanitizeMap(input)

	// Check that secret_key and token are redacted
	if result["secret_key"] != "[REDACTED]" {
		t.Errorf("secret_key not redacted: %v", result["secret_key"])
	}
	if result["token"] != "[REDACTED]" {
		t.Errorf("token not redacted: %v", result["token"])
	}

	// Check that safe values are preserved
	if result["app_name"] != "my-app" {
		t.Errorf("app_name was modified: %v", result["app_name"])
	}
}
