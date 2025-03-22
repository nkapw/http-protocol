package headers

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

type Headers map[string]string

func NewHeaders() Headers {
	return make(Headers)
}
func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	end := bytes.Index(data, []byte("\r\n"))
	if end == -1 {
		return 0, false, nil
	}

	if end == 0 {
		return 2, true, nil
	}

	line := string(data[:end])

	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return 0, false, errors.New("invalid header format")
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if strings.Contains(parts[0], " ") {
		return 0, false, errors.New("invalid spacing in header key")
	}

	if !isValidKey(key) {
		return 0, false, errors.New("invalid character in header key")
	}

	lowerKey := strings.ToLower(key)
	if val, ok := h[lowerKey]; ok {
		value = fmt.Sprintf("%s, %s", val, value)
	}
	h[lowerKey] = value

	return end + 2, false, nil
}

func isValidKey(key string) bool {
	if len(key) == 0 {
		return false
	}

	for _, char := range key {
		if !isAllowedChar(char) {
			return false
		}
	}

	return true
}

func isAllowedChar(char rune) bool {
	switch {
	case 'A' <= char && char <= 'Z':
		return true
	case 'a' <= char && char <= 'z':
		return true
	case '0' <= char && char <= '9':
		return true
	case strings.ContainsRune("!#$%&'*+-.^_`|~", char):
		return true
	default:
		return false
	}
}
