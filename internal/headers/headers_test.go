package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaders(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected map[string]string
		n        int
		done     bool
		hasError bool
	}{
		{
			name:     "Valid single header",
			data:     []byte("Host: localhost:42069\r\n\r\n"),
			expected: map[string]string{"host": "localhost:42069"},
			n:        23,
			done:     false,
			hasError: false,
		},
		{
			name:     "Invalid spacing headers",
			data:     []byte("       Host : localhost:42069       \r\n\r\n"),
			expected: nil,
			n:        0,
			done:     false,
			hasError: true,
		},
		{
			name:     "Valid 2 headers with existing headers",
			data:     []byte("User-Agent: curl/7.81.0\r\n\r\n"),
			expected: map[string]string{"accept": "*/*", "user-agent": "curl/7.81.0"},
			n:        25,
			done:     false,
			hasError: false,
		},
		{
			name:     "Valid done",
			data:     []byte("\r\n"),
			expected: nil,
			n:        2,
			done:     true,
			hasError: false,
		},
		{
			name:     "Invalid spacing header",
			data:     []byte("Host : localhost:42069\r\n\r\n"),
			expected: nil,
			n:        0,
			done:     false,
			hasError: true,
		},
		{
			name:     "Valid single header with mixed case",
			data:     []byte("ConTent-LeNgth: 42\r\n\r\n"),
			expected: map[string]string{"content-length": "42"},
			n:        20,
			done:     false,
			hasError: false,
		},
		{
			name:     "Invalid character in header key",
			data:     []byte("H©st: localhost:42069\r\n\r\n"),
			expected: nil,
			n:        0,
			done:     false,
			hasError: true,
		},
		{
			name:     "Empty key",
			data:     []byte(": localhost:42069\r\n\r\n"),
			expected: nil,
			n:        0,
			done:     false,
			hasError: true,
		},
		{
			name:     "Multiple headers with mixed case",
			data:     []byte("CoNtEnT-TyPe: text/plain\r\n\r\n"),
			expected: map[string]string{"content-type": "text/plain"},
			n:        26,
			done:     false,
			hasError: false,
		},
		{
			name:     "Multiple values",
			data:     []byte("CoNtEnT-TyPe: text/plain\r\n\r\n"),
			expected: map[string]string{"content-type": "application/json, text/plain"},
			n:        26,
			done:     false,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := NewHeaders()
			if tt.name == "Valid 2 headers with existing headers" {
				headers["accept"] = "*/*"
			}
			if tt.name == "Multiple values" {
				headers["content-type"] = "application/json"
			}
			n, done, err := headers.Parse(tt.data)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.expected != nil {
				for k, v := range tt.expected {
					assert.Equal(t, v, headers[k])
				}
			}
			assert.Equal(t, tt.n, n)
			assert.Equal(t, tt.done, done)
		})
	}
}
