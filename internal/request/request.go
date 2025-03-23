package request

import (
	"bytes"
	"errors"
	"fmt"
	"http-protocol/internal/headers"
	"io"
	"strconv"
	"strings"
)

const (
	stateInitialized = iota
	stateDone
	requestStateParsingHeaders
	stateParsingBody
)

const crlf = "\r\n"
const bufSize = 8

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	Body        []byte
	state       int
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	buf := make([]byte, bufSize)
	readToIndex := 0
	r := &Request{state: stateInitialized}

	for r.state != stateDone {
		if readToIndex == len(buf) {
			newSize := len(buf) * 2
			newBuf := make([]byte, newSize)
			copy(newBuf, buf)
			buf = newBuf
		}

		n, err := reader.Read(buf[readToIndex:])
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		readToIndex += n

		consumed, parseErr := r.parse(buf[:readToIndex])
		if parseErr != nil {
			return nil, parseErr
		}

		copy(buf, buf[consumed:readToIndex])
		readToIndex -= consumed

		if errors.Is(err, io.EOF) {
			if r.state != stateDone {
				return nil, fmt.Errorf("missing end of headers")
			}
			r.state = stateDone
		}
	}
	return r, nil
}

func parseRequestLine(data []byte) (int, *RequestLine, error) {
	end := bytes.Index(data, []byte(crlf))
	if end == -1 {
		return 0, nil, nil
	}

	parts := strings.Split(string(data[:end]), " ")
	if len(parts) != 3 {
		return 0, nil, fmt.Errorf("invalid request line")
	}

	return end + 2, &RequestLine{
		Method:        parts[0],
		RequestTarget: parts[1],
		HttpVersion:   parts[2],
	}, nil
}

func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0

	for r.state != stateDone {
		n, err := r.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return totalBytesParsed, err
		}
		totalBytesParsed += n

		// Jika tidak ada data lagi yang bisa diproses, keluar dari loop
		if n == 0 {
			break
		}
	}

	return totalBytesParsed, nil
}

func (r *Request) parseSingle(data []byte) (int, error) {
	switch r.state {
	case stateInitialized:
		consumed, reqLine, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if consumed == 0 {
			return 0, nil
		}
		r.RequestLine = *reqLine
		r.state = requestStateParsingHeaders
		return consumed, nil
	case requestStateParsingHeaders:
		if len(r.Headers) == 0 {
			r.Headers = headers.NewHeaders()
		}
		consumed, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			contentLengthStr := r.Headers.Get("Content-Length")
			if contentLengthStr == "" {
				r.state = stateDone
			} else {
				r.state = stateParsingBody
			}
		}
		return consumed, nil
	case stateParsingBody:
		contentLengthStr := r.Headers.Get("Content-Length")
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			return 0, fmt.Errorf("invalid content length")
		}

		remainingBytes := contentLength - len(r.Body)
		if remainingBytes <= 0 {
			r.state = stateDone
			return 0, nil
		}

		bytesToRead := min(remainingBytes, len(data))
		r.Body = append(r.Body, data[:bytesToRead]...)

		if len(r.Body) == contentLength {
			r.state = stateDone
		}
		return bytesToRead, nil
	case stateDone:
		return 0, fmt.Errorf("error: trying to read data in a done state")
	default:
		return 0, fmt.Errorf("error: unknown state")
	}
}
