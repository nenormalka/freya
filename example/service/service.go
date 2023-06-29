package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/conns/kafka"

	"github.com/nenormalka/freya/example/repo"
	"github.com/nenormalka/freya/types"

	"go.uber.org/dig"
	"go.uber.org/zap"
)

var Module = types.Module{
	{CreateFunc: NewService},
	{CreateFunc: Adapter},
	{CreateFunc: AdapterParam},
}

type (
	AdapterOut struct {
		dig.Out

		Service types.Runnable `group:"services"`
	}

	AdapterParams struct {
		dig.In

		Repo   *repo.Repo
		Logger *zap.Logger
	}
)

func Adapter(s *Service) AdapterOut {
	return AdapterOut{
		Service: s,
	}
}

func AdapterParam(in AdapterParams) Params {
	return Params{
		Repo:   in.Repo,
		Logger: in.Logger,
	}
}

type (
	Params struct {
		Repo   Repo
		Logger *zap.Logger
	}

	Repo interface {
		GetNow(ctx context.Context) (string, error)
	}

	Service struct {
		logger *zap.Logger
		repo   Repo
		cg     kafka.ConsumerGroup
		ch     chan KafkaMessage
	}

	KafkaMessage struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
)

func NewService(p Params, conns *conns.Conns) (*Service, error) {
	k, err := conns.GetKafka()
	if err != nil {
		return nil, fmt.Errorf("get kafka err: %w", err)
	}

	cg, err := k.NewConsumerGroup("test", nil)
	if err != nil {
		return nil, fmt.Errorf("new consumer group err: %w", err)
	}

	s := &Service{
		logger: p.Logger,
		repo:   p.Repo,
		cg:     cg,
		ch:     make(chan KafkaMessage),
	}

	s.addHandler()
	s.process()

	return s, nil
}

func (s *Service) Start(ctx context.Context) error {
	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return fmt.Errorf("get now from repo err: %w", err)
	}

	s.logger.Info("service run at", zap.String("time", now))

	s.cg.Consume()

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if err := s.cg.Close(); err != nil {
		return fmt.Errorf("close consumer group err: %w", err)
	}

	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return fmt.Errorf("start service err: %w", err)
	}

	s.logger.Info("service stopped at", zap.String("time", now))

	return nil
}

func (s *Service) Now(ctx context.Context) (string, error) {
	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return "", fmt.Errorf("get now from repo err: %w", err)
	}

	return now, nil
}

func (s *Service) addHandler() {
	s.cg.AddHandler("utp.example.test", func(msg json.RawMessage) error {
		m := KafkaMessage{}

		if err := json.Unmarshal(msg, &m); err != nil {
			return fmt.Errorf("unmarshal message err: %w", err)
		}

		s.ch <- m

		return nil
	})
}

func (s *Service) process() {
	go func() {
		for msg := range s.ch {
			s.logger.Info("kafka message", zap.Any("msg", msg))
		}
	}()
}
