package mocks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

var (
	ErrEmptyMessages    = fmt.Errorf("empty messages")
	ErrorNoMoreMessages = fmt.Errorf("no more messages")
)

type (
	ElasticMessageMock struct {
		Request   string
		Response  string
		WithError bool
		Error     error
		Status    int
	}

	ElasticMock struct {
		client *elasticsearch.Client
	}

	ElasticMockTransport struct {
		searchMessages []ElasticMessageMock
		messageCursor  int
	}
)

func NewElasticMock(messages []ElasticMessageMock) (*ElasticMock, error) {
	if len(messages) == 0 {
		return nil, ErrEmptyMessages
	}

	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Transport: &ElasticMockTransport{
			searchMessages: messages,
			messageCursor:  0,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating elastic client: %w", err)
	}

	return &ElasticMock{client: client}, nil
}

func (em *ElasticMock) CallContext(
	ctx context.Context,
	_ string,
	callFunc func(ctx context.Context, client *elasticsearch.Client) error,
) error {
	return callFunc(ctx, em.client)
}

func (emt *ElasticMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if emt.messageCursor >= len(emt.searchMessages) {
		return nil, ErrorNoMoreMessages
	}

	request, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading request body: %w", err)
	}

	message := emt.searchMessages[emt.messageCursor]
	if string(request) != message.Request {
		return nil, fmt.Errorf("request mismatch: got %s, expected %s", request, message.Request)
	}

	if message.WithError {
		return nil, message.Error
	}

	emt.messageCursor++

	return &http.Response{
		StatusCode: message.Status,
		Body:       io.NopCloser(strings.NewReader(message.Response)),
		Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
	}, nil
}
