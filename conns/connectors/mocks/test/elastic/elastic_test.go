package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/nenormalka/freya/conns/connectors/mocks"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/stretchr/testify/require"
)

func TestElastic(t *testing.T) {
	req1 := `{"query": {"match_all_test": {}}}`
	req2 := `{"query": {"match_all": {}}}`
	resp1 := `{"hits": {"hits": [{"_source": {"name": "test"}}]}}`
	errEx := fmt.Errorf("error")

	esMock, err := mocks.NewElasticMock([]mocks.ElasticMessageMock{
		{
			Request:   req1,
			Response:  resp1,
			Status:    200,
			WithError: false,
			Error:     nil,
		},
		{
			Request:   req2,
			Response:  ``,
			Status:    500,
			WithError: true,
			Error:     errEx,
		},
	})

	require.Nilf(t, err, "error creating elastic mock: %v", err)

	if err = esMock.CallContext(context.TODO(), "test", func(ctx context.Context, client *elasticsearch.Client) error {
		resp, errS := client.Search(
			client.Search.WithContext(ctx),
			client.Search.WithIndex("testIndex"),
			client.Search.WithBody(strings.NewReader(req1)),
		)
		if errS != nil {
			return fmt.Errorf("error searching: %w", errS)
		}

		if resp.IsError() {
			return fmt.Errorf("error searching: %w", errS)
		}

		response, ereS := io.ReadAll(resp.Body)
		if ereS != nil {
			return fmt.Errorf("error reading response: %w", ereS)
		}

		if string(response) != resp1 {
			return fmt.Errorf("invalid response")
		}

		return nil
	}); err != nil {
		require.Nilf(t, err, "error searching: %v", err)
		return
	}

	if err = esMock.CallContext(context.TODO(), "test", func(ctx context.Context, client *elasticsearch.Client) error {
		resp, errS := client.Search(
			client.Search.WithContext(ctx),
			client.Search.WithIndex("testIndex"),
			client.Search.WithBody(strings.NewReader(req2)),
		)
		if errS != nil {
			return fmt.Errorf("error searching: %w", errS)
		}

		if resp.IsError() {
			return fmt.Errorf("error searching: %w", errS)
		}

		return nil
	}); err != nil {
		require.ErrorIsf(t, err, errEx, "error expected")
	} else {
		require.NotNilf(t, err, "error expected")
	}
}
