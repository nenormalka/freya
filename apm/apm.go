package apm

import (
	"fmt"
	"net/url"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/transport"
)

func NewAPM(cfg Config) (*apm.Tracer, error) {
	apmTransport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	if err != nil {
		return nil, fmt.Errorf("apm NewHTTPTransport %w", err)
	}

	serverURL, err := url.Parse(cfg.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("url Parse %w", err)
	}

	if err = apmTransport.SetServerURL(serverURL); err != nil {
		return nil, fmt.Errorf("apm SetServerURL %w", err)
	}

	apmTracer, err := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName:        cfg.ServiceName,
		ServiceVersion:     cfg.ReleaseID,
		ServiceEnvironment: cfg.Environment,
		Transport:          apmTransport,
	})
	if err != nil {
		return nil, fmt.Errorf("apm NewTracerOptions %w", err)
	}

	return apmTracer, nil
}
