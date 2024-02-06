package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/nenormalka/freya"
	"github.com/nenormalka/freya/conns/consul/client"
	"github.com/nenormalka/freya/conns/consul/config"
	"github.com/nenormalka/freya/conns/consul/leader"
	"github.com/nenormalka/freya/conns/consul/lock"
	"github.com/nenormalka/freya/conns/consul/session"
	"github.com/nenormalka/freya/conns/consul/watcher"
	"github.com/nenormalka/freya/types"
)

func main() {
	if err := freya.NewMockEngine(
		false,
		freya.WithModulesOpt(types.Module{
			{CreateFunc: func() config.Config {
				return config.Config{
					Address:            "localhost:8500",
					Scheme:             "http",
					Token:              "",
					ServiceName:        "test_leader",
					SessionTTL:         "30s",
					LeaderTTL:          20 * time.Second,
					InsecureSkipVerify: true,
				}
			}},
			{CreateFunc: func() *zap.Logger {
				return zap.NewNop()
			}},
		}),
	).Run(func(cfg config.Config, logger *zap.Logger) error {
		wg := sync.WaitGroup{}
		wg.Add(3)
		ctx := context.Background()

		for i, t := range []time.Duration{1 * time.Minute, 2 * time.Minute, 3 * time.Minute} {
			go func(i int, t time.Duration) {
				for {
					time.Sleep(time.Duration(rand.Int63n(int64(3*(i+1)))) * time.Second)

					cli, err := client.NewClient(cfg)
					if err != nil {
						fmt.Printf("failed to create consul client: %s", err.Error())
						return
					}

					w := watcher.NewWatcher(cli, logger)
					s := session.NewSession(cli, cfg)
					l := lock.NewLocker(cli)

					lead := leader.NewLeader(l, s, w, logger, cfg)

					if err = lead.Start(ctx); err != nil {
						fmt.Printf("failed to start lead: %s", err.Error())
						return
					}

					fmt.Printf("start instance number %d time %s\n", i, time.Now().Format("2006-01-02 15:04:05"))

					stopCh := make(chan struct{})

					go func(i int) {
						ticker := time.NewTicker(20 * time.Second)

						for {
							<-ticker.C
							select {
							case <-stopCh:
								return
							default:
							}

							now := time.Now().Format("2006-01-02 15:04:05")

							if lead.IsLeader() {
								fmt.Printf("I'm leader 🤓. Instance number %d. Time %s\n", i, now)
							} else {
								fmt.Printf("I'm not leader 😭. Instance number %d. Time %s\n", i, now)
							}
						}
					}(i)

					<-time.After(t)

					close(stopCh)

					if err = lead.Stop(ctx); err != nil {
						fmt.Printf("failed to stop lead: %s", err.Error())
						return
					}

					now := time.Now().Format("2006-01-02 15:04:05")

					if lead.IsLeader() {
						fmt.Printf("I'm leader 🤓and I go to bed. Instance number %d. Time %s\n", i, now)
					} else {
						fmt.Printf("I'm not leader 😭and I go to gym. Instance number %d. Time %s\n", i, now)
					}

					time.Sleep(1 * time.Minute)
				}
			}(i, t)
		}

		wg.Wait()

		return nil
	}); err != nil {
		panic(err)
	}
}
