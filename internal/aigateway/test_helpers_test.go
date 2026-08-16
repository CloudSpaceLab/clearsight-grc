package aigateway

import (
	"context"
	"io"
	"sync"
)

type fakeProvider struct {
	id            string
	response      Response
	completeErr   error
	streamFactory func() ProviderStream

	mu            sync.Mutex
	completeCalls int
	streamCalls   int
}

func (p *fakeProvider) ID() string { return p.id }

func (p *fakeProvider) Complete(context.Context, ProviderRequest) (Response, error) {
	p.mu.Lock()
	p.completeCalls++
	p.mu.Unlock()
	if p.completeErr != nil {
		return Response{}, p.completeErr
	}
	return p.response, nil
}

func (p *fakeProvider) Stream(context.Context, ProviderRequest) (ProviderStream, error) {
	p.mu.Lock()
	p.streamCalls++
	p.mu.Unlock()
	if p.streamFactory != nil {
		return p.streamFactory(), nil
	}
	return &fakeStream{errors: map[int]error{0: ErrUnavailable}}, nil
}

func (p *fakeProvider) counts() (int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.completeCalls, p.streamCalls
}

type fakeStream struct {
	events []StreamEvent
	errors map[int]error
	index  int
	closed bool
}

func (s *fakeStream) Recv() (StreamEvent, error) {
	if s.closed {
		return StreamEvent{}, io.EOF
	}
	if err := s.errors[s.index]; err != nil {
		s.index++
		return StreamEvent{}, err
	}
	if s.index >= len(s.events) {
		return StreamEvent{}, io.EOF
	}
	event := s.events[s.index]
	s.index++
	return event, nil
}

func (s *fakeStream) Close() error { s.closed = true; return nil }
