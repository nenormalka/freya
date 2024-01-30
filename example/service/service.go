package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/couchbase/couchlock"
	"github.com/nenormalka/freya/conns/couchbase/types"
	"github.com/nenormalka/freya/conns/kafka"
	ferrors "github.com/nenormalka/freya/types/errors"

	"github.com/couchbase/gocb/v2"
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
		logger     *zap.Logger
		repo       Repo
		cg         kafka.ConsumerGroup
		sp         kafka.SyncProducer
		ch         chan KafkaMessage
		close      chan struct{}
		collection connectors.DBConnector[*gocb.Collection, *types.CollectionTx]
		locker     *couchlock.CouchLock
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

	cb, err := conns.GetCouchbase()
	if err != nil {
		return nil, fmt.Errorf("get couchbase err: %w", err)
	}

	coll, err := cb.GetCollection("example", "example")
	if err != nil {
		return nil, fmt.Errorf("get collection err: %w", err)
	}

	locker, err := couchlock.NewCouchLock(
		conns,
		"example",
		"example",
		couchlock.WithRetryCountOption(0),
		couchlock.WithTTLOption(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("new couch lock err: %w", err)
	}

	s := &Service{
		logger:     p.Logger,
		repo:       p.Repo,
		cg:         cg,
		sp:         sp,
		ch:         make(chan KafkaMessage),
		close:      make(chan struct{}),
		collection: coll,
		locker:     locker,
	}

	if err = s.addTypedHandler(); err != nil {
		return nil, fmt.Errorf("add typed handler err: %w", err)
	}

	s.process()

	return s, nil
}

func (s *Service) Start(ctx context.Context) error {
	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return fmt.Errorf("get now from repo err: %w", err)
	}

	s.logger.Info("service run at", zap.String("time", now))

	if err = s.cg.Consume(); err != nil {
		return fmt.Errorf("consume err: %w", err)
	}

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
	var now string

	s.collection.CallContext(ctx, "get_now", func(ctx context.Context, collection *gocb.Collection) error {
		resp, errC := collection.Get("time_now", &gocb.GetOptions{
			WithExpiry: true,
			Context:    ctx,
		})
		if errC != nil {
			return fmt.Errorf("collection.Get: %w", errC)
		}

		if errC = resp.Content(&now); errC != nil {
			return fmt.Errorf("resp.Content: %w", errC)
		}

		return nil
	})

	if now != "" {
		return now, nil
	}

	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return "", fmt.Errorf("get now from repo err: %w", err)
	}

	s.collection.CallContext(ctx, "set_now", func(ctx context.Context, collection *gocb.Collection) error {
		_, errC := collection.Upsert("time_now", now, &gocb.UpsertOptions{
			Expiry:  10 * time.Second,
			Context: ctx,
		})

		if errC != nil {
			return fmt.Errorf("collection.Upsert: %w", err)
		}

		return nil
	})

	return now, nil
}

func (s *Service) GetErr(code ferrors.Code) error {
	switch code {
	case ferrors.InvalidArgument:
		return ferrors.NewInvalidError(errors.New("invalid argument")).AddDetail("id", "1")
	case ferrors.NotFound:
		return ferrors.NewNotFoundError(errors.New("not found")).AddDetail("id", "2")
	default:
		return ferrors.NewUnknownError(errors.New("unknown error")).AddDetail("id", "3")
	}
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

func (s *Service) addTypedHandler() error {
	if err := kafka.AddTypedHandler(s.cg, topic, func(msg KafkaMessage) error {
		s.ch <- msg

		return nil
	}); err != nil {
		return fmt.Errorf("add typed handler err: %w", err)
	}

	return nil
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

			if err := kafka.TypedSend(s.sp, topic, msg); err != nil {
				s.logger.Error("produce message err", zap.Error(err))
			}

			i++
		}
	}()
}
