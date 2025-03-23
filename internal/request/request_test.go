package request

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

func (cr *chunkReader) Read(p []byte) (n int, err error) {
	if cr.pos >= len(cr.data) {
		return 0, io.EOF
	}
	endIndex := min(cr.pos+cr.numBytesPerRead, len(cr.data))
	n = copy(p, cr.data[cr.pos:endIndex])
	cr.pos += n

	if n > cr.numBytesPerRead {
		n = cr.numBytesPerRead
		cr.pos -= n - cr.numBytesPerRead
	}
	return n, nil
}

func TestRequestLineParser(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected *RequestLine
		hasError bool
	}{
		{
			name: "GoodGetRequestLine",
			data: "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			expected: &RequestLine{
				Method:        "GET",
				RequestTarget: "/",
				HttpVersion:   "HTTP/1.1",
			},
			hasError: false,
		},
		{
			name: "GoodGetRequestLineWithPath",
			data: "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			expected: &RequestLine{
				Method:        "GET",
				RequestTarget: "/coffee",
				HttpVersion:   "HTTP/1.1",
			},
			hasError: false,
		},
		{
			name:     "InvalidNumberOfPartsInRequestLine",
			data:     "/coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &chunkReader{
				data:            tt.data,
				numBytesPerRead: 3,
			}
			r, err := RequestFromReader(reader)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, r)
				assert.Equal(t, tt.expected.Method, r.RequestLine.Method)
				assert.Equal(t, tt.expected.RequestTarget, r.RequestLine.RequestTarget)
				assert.Equal(t, tt.expected.HttpVersion, r.RequestLine.HttpVersion)
			}
		})
	}
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected map[string]string
		hasError bool
	}{
		{
			name: "Standard Headers",
			data: "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
			expected: map[string]string{
				"host":       "localhost:42069",
				"user-agent": "curl/7.81.0",
				"accept":     "*/*",
			},
			hasError: false,
		},
		{
			name:     "Malformed Header",
			data:     "GET / HTTP/1.1\r\nHost localhost:42069\r\n\r\n",
			expected: nil,
			hasError: true,
		},
		{
			name:     "Empty Headers",
			data:     "GET / HTTP/1.1\r\n\r\n",
			expected: map[string]string{},
			hasError: false,
		},
		{
			name: "Duplicate Headers",
			data: "GET / HTTP/1.1\r\nSet-Person: lane-loves-go\r\nSet-Person: prime-loves-zig\r\n\r\n",
			expected: map[string]string{
				"set-person": "lane-loves-go, prime-loves-zig",
			},
			hasError: false,
		},
		{
			name: "Case Insensitive Headers",
			data: "GET / HTTP/1.1\r\nHOST: localhost:42069\r\nuser-agent: curl/7.81.0\r\n\r\n",
			expected: map[string]string{
				"host":       "localhost:42069",
				"user-agent": "curl/7.81.0",
			},
			hasError: false,
		},
		{
			name:     "Missing End of Headers",
			data:     "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &chunkReader{
				data:            tt.data,
				numBytesPerRead: 3,
			}
			r, err := RequestFromReader(reader)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, r)
				for k, v := range tt.expected {
					assert.Equal(t, v, r.Headers[k])
				}
			}
		})
	}
}

func TestRequestParsingBody(t *testing.T) {
	// Test: Standard Body
	reader := &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 13\r\n" +
			"\r\n" +
			"hello world!\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "hello world!\n", string(r.Body))

	// Test: Empty Body, 0 reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "", string(r.Body))

	// Test: Empty Body, no reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"\r\n",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "", string(r.Body))

	// Test: Body shorter than reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 20\r\n" +
			"\r\n" +
			"partial content",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: No Content-Length but Body Exists
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"\r\n" +
			"hello world!\n",
		numBytesPerRead: 3,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "", string(r.Body))
}
