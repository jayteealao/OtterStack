package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDetectType tests type detection from variable names
func TestDetectType(t *testing.T) {
	tests := []struct {
		name         string
		varName      string
		defaultValue string
		expected     VarType
	}{
		// URL patterns
		{
			name:     "DATABASE_URL",
			varName:  "DATABASE_URL",
			expected: TypeURL,
		},
		{
			name:     "API_URI",
			varName:  "API_URI",
			expected: TypeURL,
		},
		{
			name:     "REDIS_ENDPOINT",
			varName:  "REDIS_ENDPOINT",
			expected: TypeURL,
		},
		{
			name:     "DB_HOST",
			varName:  "DB_HOST",
			expected: TypeURL,
		},

		// Email patterns
		{
			name:     "ADMIN_EMAIL",
			varName:  "ADMIN_EMAIL",
			expected: TypeEmail,
		},
		{
			name:     "SUPPORT_EMAIL",
			varName:  "SUPPORT_EMAIL",
			expected: TypeEmail,
		},

		// Port patterns
		{
			name:     "HTTP_PORT",
			varName:  "HTTP_PORT",
			expected: TypePort,
		},
		{
			name:     "DB_PORTS",
			varName:  "DB_PORTS",
			expected: TypePort,
		},

		// Integer patterns
		{
			name:     "WORKER_COUNT",
			varName:  "WORKER_COUNT",
			expected: TypeInteger,
		},
		{
			name:     "MAX_CONNECTIONS",
			varName:  "MAX_CONNECTIONS",
			expected: TypeInteger,
		},
		{
			name:     "TIMEOUT",
			varName:  "TIMEOUT",
			expected: TypeInteger,
		},
		{
			name:     "WORKERS",
			varName:  "WORKERS",
			expected: TypeInteger,
		},
		{
			name:     "THREAD_LIMIT",
			varName:  "THREAD_LIMIT",
			expected: TypeInteger,
		},

		// Boolean patterns
		{
			name:     "ENABLE_CACHE",
			varName:  "ENABLE_CACHE",
			expected: TypeBoolean,
		},
		{
			name:     "DEBUG_ENABLED",
			varName:  "DEBUG_ENABLED",
			expected: TypeBoolean,
		},
		{
			name:     "FEATURE_FLAG",
			varName:  "FEATURE_FLAG",
			expected: TypeBoolean,
		},
		{
			name:     "USE_SSL",
			varName:  "USE_SSL",
			expected: TypeBoolean,
		},

		// Detect from default value (URL)
		{
			name:         "CUSTOM_VAR with URL default",
			varName:      "CUSTOM_VAR",
			defaultValue: "https://example.com",
			expected:     TypeURL,
		},

		// Detect from default value (Email)
		{
			name:         "CUSTOM_VAR with email default",
			varName:      "CUSTOM_VAR",
			defaultValue: "user@example.com",
			expected:     TypeEmail,
		},

		// Detect from default value (Port)
		{
			name:         "CUSTOM_VAR with port default",
			varName:      "CUSTOM_VAR",
			defaultValue: "3000",
			expected:     TypePort,
		},

		// Detect from default value (Boolean)
		{
			name:         "CUSTOM_VAR with boolean default",
			varName:      "CUSTOM_VAR",
			defaultValue: "true",
			expected:     TypeBoolean,
		},

		// Default to string
		{
			name:     "SECRET_KEY",
			varName:  "SECRET_KEY",
			expected: TypeString,
		},
		{
			name:         "RANDOM_VAR",
			varName:      "RANDOM_VAR",
			defaultValue: "some value",
			expected:     TypeString,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectType(tt.varName, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidateURL tests URL validation
func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid https URL",
			value:   "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			value:   "http://localhost:3000",
			wantErr: false,
		},
		{
			name:    "valid postgres URL",
			value:   "postgres://user:pass@localhost/db",
			wantErr: false,
		},
		{
			name:    "empty (allowed)",
			value:   "",
			wantErr: false,
		},
		{
			name:    "no scheme",
			value:   "example.com",
			wantErr: true,
		},
		{
			name:    "invalid format",
			value:   "not a url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateEmail tests email validation
func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid email",
			value:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			value:   "admin@mail.example.com",
			wantErr: false,
		},
		{
			name:    "empty (allowed)",
			value:   "",
			wantErr: false,
		},
		{
			name:    "missing @",
			value:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "missing domain",
			value:   "user@",
			wantErr: true,
		},
		{
			name:    "invalid format",
			value:   "not an email",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePort tests port number validation
func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid port",
			value:   "3000",
			wantErr: false,
		},
		{
			name:    "port 80",
			value:   "80",
			wantErr: false,
		},
		{
			name:    "port 65535 (max)",
			value:   "65535",
			wantErr: false,
		},
		{
			name:    "empty (allowed)",
			value:   "",
			wantErr: false,
		},
		{
			name:    "port 0 (invalid)",
			value:   "0",
			wantErr: true,
		},
		{
			name:    "port above 65535",
			value:   "99999",
			wantErr: true,
		},
		{
			name:    "not a number",
			value:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateInteger tests integer validation
func TestValidateInteger(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid positive integer",
			value:   "42",
			wantErr: false,
		},
		{
			name:    "zero",
			value:   "0",
			wantErr: false,
		},
		{
			name:    "negative integer",
			value:   "-10",
			wantErr: false,
		},
		{
			name:    "empty (allowed)",
			value:   "",
			wantErr: false,
		},
		{
			name:    "not a number",
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "float",
			value:   "3.14",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInteger(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateBoolean tests boolean validation
func TestValidateBoolean(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "true",
			value:   "true",
			wantErr: false,
		},
		{
			name:    "false",
			value:   "false",
			wantErr: false,
		},
		{
			name:    "yes",
			value:   "yes",
			wantErr: false,
		},
		{
			name:    "no",
			value:   "no",
			wantErr: false,
		},
		{
			name:    "1",
			value:   "1",
			wantErr: false,
		},
		{
			name:    "0",
			value:   "0",
			wantErr: false,
		},
		{
			name:    "uppercase TRUE",
			value:   "TRUE",
			wantErr: false,
		},
		{
			name:    "mixed case Yes",
			value:   "Yes",
			wantErr: false,
		},
		{
			name:    "empty (allowed)",
			value:   "",
			wantErr: false,
		},
		{
			name:    "invalid value",
			value:   "maybe",
			wantErr: true,
		},
		{
			name:    "number other than 0/1",
			value:   "2",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBoolean(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetPlaceholder tests placeholder text generation
func TestGetPlaceholder(t *testing.T) {
	tests := []struct {
		name         string
		varType      VarType
		defaultValue string
		expected     string
	}{
		{
			name:         "URL with default",
			varType:      TypeURL,
			defaultValue: "https://example.com",
			expected:     "https://example.com",
		},
		{
			name:     "URL without default",
			varType:  TypeURL,
			expected: "https://example.com",
		},
		{
			name:     "Email without default",
			varType:  TypeEmail,
			expected: "user@example.com",
		},
		{
			name:     "Port without default",
			varType:  TypePort,
			expected: "3000",
		},
		{
			name:     "Integer without default",
			varType:  TypeInteger,
			expected: "10",
		},
		{
			name:     "Boolean without default",
			varType:  TypeBoolean,
			expected: "true",
		},
		{
			name:         "String with default",
			varType:      TypeString,
			defaultValue: "my default",
			expected:     "my default",
		},
		{
			name:     "String without default",
			varType:  TypeString,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPlaceholder(tt.varType, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidateByType tests that correct validators are returned
func TestValidateByType(t *testing.T) {
	tests := []struct {
		name    string
		varType VarType
		isNil   bool
	}{
		{
			name:    "URL validator",
			varType: TypeURL,
			isNil:   false,
		},
		{
			name:    "Email validator",
			varType: TypeEmail,
			isNil:   false,
		},
		{
			name:    "Port validator",
			varType: TypePort,
			isNil:   false,
		},
		{
			name:    "Integer validator",
			varType: TypeInteger,
			isNil:   false,
		},
		{
			name:    "Boolean validator",
			varType: TypeBoolean,
			isNil:   false,
		},
		{
			name:    "String validator (nil)",
			varType: TypeString,
			isNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := ValidateByType(tt.varType)
			if tt.isNil {
				assert.Nil(t, validator)
			} else {
				assert.NotNil(t, validator)
			}
		})
	}
}
