package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/conns/kafka"

	"go.uber.org/zap"
)

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
		sp     kafka.SyncProducer
		ch     chan KafkaMessage
		close  chan struct{}
	}

	KafkaMessage struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
)

const (
	topic = "utp.example.test"
)

func NewService(p Params, conns *conns.Conns) (*Service, error) {
	k, err := conns.GetKafka()
	if err != nil {
		return nil, fmt.Errorf("get kafka err: %w", err)
	}

	cg, err := k.NewConsumerGroup("test")
	if err != nil {
		return nil, fmt.Errorf("new consumer group err: %w", err)
	}

	sp, err := k.NewSyncProducer()
	if err != nil {
		return nil, fmt.Errorf("new sync producer err: %w", err)
	}

	s := &Service{
		logger: p.Logger,
		repo:   p.Repo,
		cg:     cg,
		sp:     sp,
		ch:     make(chan KafkaMessage),
		close:  make(chan struct{}),
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
	s.produce()

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	close(s.close)

	_ = s.cg.Close()
	_ = s.sp.Close()

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
	s.cg.AddHandler(topic, func(msg json.RawMessage) error {
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

func (s *Service) produce() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-s.close:
				return
			default:
			}

			<-ticker.C

			msg := KafkaMessage{
				Name: "test" + strconv.Itoa(i),
				Age:  i,
			}

			data, err := json.Marshal(msg)
			if err != nil {
				s.logger.Error("marshal message err", zap.Error(err))
				continue
			}

			if err = s.sp.Send(topic, data); err != nil {
				s.logger.Error("produce message err", zap.Error(err))
			}

			i++
		}
	}()
}
