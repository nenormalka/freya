package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"

	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/consul"
	"github.com/nenormalka/freya/conns/consul/watcher"
	"github.com/nenormalka/freya/conns/couchbase/couchlock"
	"github.com/nenormalka/freya/conns/couchbase/types"
	"github.com/nenormalka/freya/conns/kafka"
	ferrors "github.com/nenormalka/freya/types/errors"
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
		logger        *zap.Logger
		repo          Repo
		cg            kafka.ConsumerGroup
		sp            kafka.SyncProducer
		ch            chan KafkaMessage
		close         chan struct{}
		collection    connectors.DBConnector[*gocb.Collection, *types.CollectionTx]
		consulKV      connectors.DBConnector[*api.KV, *api.Txn]
		consulWatcher consul.Watcher
		locker        *couchlock.CouchLock
		leader        consul.Leader
	}

	KafkaMessage struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
)

const (
	topic = "utp.example.test"

	kvPrefixKey = "service/"
	kvWatchKey1 = kvPrefixKey + "kv_test_1"
	kvWatchKey2 = kvPrefixKey + "kv_test_2"
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

	csl, err := conns.GetConsul()
	if err != nil {
		return nil, fmt.Errorf("get csl err: %w", err)
	}

	s := &Service{
		logger:        p.Logger,
		repo:          p.Repo,
		cg:            cg,
		sp:            sp,
		ch:            make(chan KafkaMessage),
		close:         make(chan struct{}),
		collection:    coll,
		locker:        locker,
		consulKV:      csl.KV(),
		consulWatcher: csl.Watcher(),
		leader:        csl.Leader(),
	}

	if err = s.addTypedHandler(); err != nil {
		return nil, fmt.Errorf("add typed handler err: %w", err)
	}

	s.processKafkaMessages()

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

	if err = s.leader.Start(ctx); err != nil {
		return fmt.Errorf("start leader err: %w", err)
	}

	s.produceKafka()
	s.processUpdateConsulKV()
	s.checkIsLeader()

	if err = s.startWatchConsul(ctx); err != nil {
		return fmt.Errorf("start watch consul err: %w", err)
	}

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	close(s.close)

	_ = s.cg.Close()
	_ = s.sp.Close()

	_ = s.consulWatcher.Stop(ctx)
	_ = s.leader.Stop(ctx)

	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return fmt.Errorf("start service err: %w", err)
	}

	s.logger.Info("service stopped at", zap.String("time", now))

	return nil
}

func (s *Service) Now(ctx context.Context) (string, error) {
	var now string

	if err := s.collection.CallContext(ctx, "get_now", func(ctx context.Context, collection *gocb.Collection) error {
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
	}); err != nil {
		s.logger.Error("get now from couchbase err", zap.Error(err))
	}

	if now != "" {
		return now, nil
	}

	now, err := s.repo.GetNow(ctx)
	if err != nil {
		return "", fmt.Errorf("get now from repo err: %w", err)
	}

	if err = s.collection.CallContext(ctx, "set_now", func(ctx context.Context, collection *gocb.Collection) error {
		_, errC := collection.Upsert("time_now", now, &gocb.UpsertOptions{
			Expiry:  10 * time.Second,
			Context: ctx,
		})

		if errC != nil {
			return fmt.Errorf("collection.Upsert: %w", err)
		}

		return nil
	}); err != nil {
		s.logger.Error("set now to couchbase err", zap.Error(err))
	}

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

func (s *Service) startWatchConsul(ctx context.Context) error {
	if err := s.consulWatcher.WatchKeys(watcher.WatchKeys{
		kvWatchKey1: func(key, sessionID string, value []byte) {
			s.logger.Info(
				"watch key",
				zap.ByteString("value", value),
				zap.String("sessionID", sessionID),
				zap.String("key", key),
			)
		},
		kvWatchKey2: func(key, sessionID string, value []byte) {
			s.logger.Info(
				"watch key",
				zap.ByteString("value", value),
				zap.String("sessionID", sessionID),
				zap.String("key", key),
			)
		},
	}); err != nil {
		s.logger.Error("watch consul keys err", zap.Error(err))
	}

	if err := s.consulWatcher.WatchPrefixKeys(watcher.WatchPrefixKey{
		kvPrefixKey: func(params map[string][]byte) {
			for key, value := range params {
				s.logger.Info(
					"watch prefix key",
					zap.ByteString("value", value),
					zap.String("key", key),
				)
			}
		},
	}); err != nil {
		s.logger.Error("watch consul keys err", zap.Error(err))
	}

	if err := s.consulWatcher.Start(ctx); err != nil {
		return fmt.Errorf("start consul watcher err: %w", err)
	}

	return nil
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

func (s *Service) processKafkaMessages() {
	go func() {
		for msg := range s.ch {
			s.logger.Info("kafka message", zap.Any("msg", msg))
		}
	}()
}

func (s *Service) produceKafka() {
	s.produce(2*time.Second, func(i int) {
		msg := KafkaMessage{
			Name: "test" + strconv.Itoa(i),
			Age:  i,
		}

		if err := kafka.TypedSend(s.sp, topic, msg); err != nil {
			s.logger.Error("produceKafka message err", zap.Error(err))
		}
	})
}

func (s *Service) processUpdateConsulKV() {
	s.produce(3*time.Second, func(i int) {
		if err := s.consulKV.CallContext(context.Background(),
			"processUpdateConsulKV1",
			func(_ context.Context, db *api.KV) error {
				if _, err := db.Put(&api.KVPair{
					Key:   kvWatchKey1,
					Value: []byte(strconv.Itoa(i)),
				}, nil); err != nil {
					return fmt.Errorf("kv: %w", err)
				}

				return nil
			}); err != nil {
			s.logger.Error("update consul kv err", zap.Error(err))
		}
	})

	s.produce(5*time.Second, func(i int) {
		if err := s.consulKV.CallContext(context.Background(),
			"processUpdateConsulKV2",
			func(_ context.Context, db *api.KV) error {
				if _, err := db.Put(&api.KVPair{
					Key:   kvWatchKey2,
					Value: []byte(strconv.Itoa(i * 10)),
				}, nil); err != nil {
					return fmt.Errorf("kv: %w", err)
				}

				return nil
			}); err != nil {
			s.logger.Error("update consul kv err", zap.Error(err))
		}
	})
}

func (s *Service) checkIsLeader() {
	s.produce(2*time.Second, func(_ int) {
		if s.leader.IsLeader() {
			s.logger.Info("I'm leader 🤓")
		} else {
			s.logger.Info("I'm not leader 😭")
		}
	})
}

func (s *Service) produce(t time.Duration, f func(i int)) {
	go func() {
		ticker := time.NewTicker(t)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-s.close:
				return
			case <-ticker.C:
			}

			f(i)

			i++
		}
	}()
}
