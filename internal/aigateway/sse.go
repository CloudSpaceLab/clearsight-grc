package aigateway

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type sseEvent struct {
	Name string
	Data string
}

type sseReader struct {
	reader        *bufio.Reader
	maxEventBytes int64
}

func newSSEReader(reader io.Reader, maxEventBytes int64) *sseReader {
	return &sseReader{reader: bufio.NewReaderSize(reader, 32<<10), maxEventBytes: maxEventBytes}
}

func (s *sseReader) next() (sseEvent, error) {
	var event sseEvent
	var data strings.Builder
	var total int64
	for {
		line, err := s.readLine(s.maxEventBytes - total)
		total += int64(len(line))
		if total > s.maxEventBytes {
			return sseEvent{}, withCause(ErrProtocol, fmt.Errorf("SSE event exceeds configured limit"))
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return sseEvent{}, err
		}
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			if event.Name == "" && data.Len() == 0 {
				if errors.Is(err, io.EOF) {
					return sseEvent{}, io.EOF
				}
				continue
			}
			event.Data = strings.TrimSuffix(data.String(), "\n")
			return event, nil
		}
		if strings.HasPrefix(line, ":") {
			if errors.Is(err, io.EOF) {
				return sseEvent{}, io.EOF
			}
			continue
		}
		field, value, found := strings.Cut(line, ":")
		if found && strings.HasPrefix(value, " ") {
			value = value[1:]
		}
		switch field {
		case "event":
			event.Name = value
		case "data":
			data.WriteString(value)
			data.WriteByte('\n')
		}
		if errors.Is(err, io.EOF) {
			if event.Name == "" && data.Len() == 0 {
				return sseEvent{}, io.EOF
			}
			event.Data = strings.TrimSuffix(data.String(), "\n")
			return event, nil
		}
	}
}

func (s *sseReader) readLine(remaining int64) (string, error) {
	if remaining <= 0 {
		return "", withCause(ErrProtocol, fmt.Errorf("SSE event exceeds configured limit"))
	}
	var line strings.Builder
	for {
		fragment, err := s.reader.ReadSlice('\n')
		if int64(line.Len()+len(fragment)) > remaining {
			return "", withCause(ErrProtocol, fmt.Errorf("SSE event exceeds configured limit"))
		}
		line.Write(fragment)
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return line.String(), err
	}
}
