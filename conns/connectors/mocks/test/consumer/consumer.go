package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nenormalka/freya/conns/connectors/mocks"
)

func main() {
	type m struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Tick int    `json:"tick"`
	}

	f := func(topic string) func(msg json.RawMessage) error {
		return func(msg json.RawMessage) error {
			message := &m{}

			if err := json.Unmarshal(msg, message); err != nil {
				return fmt.Errorf("unmarshal message err: %w", err)
			}

			fmt.Printf("topic: %s id: %d, name: %s, tick: %d \n", topic, message.ID, message.Name, message.Tick)

			return nil
		}
	}

	consumer := mocks.NewConsumerMock(mocks.WithConsumerMockMessages([]mocks.KafkaMessagesMock{
		{
			Message: []json.RawMessage{
				[]byte(`{"id": 1, "name": "test", "tick": 1}`),
				[]byte(`{"id": 2, "name": "test2", "tick": 1}`),
				[]byte(`{"id": 3, "name": "test3", "tick": 1}`),
			},
			Topic: "test",
			Tick:  time.Second,
			Cycle: true,
		},
		{
			Message: []json.RawMessage{
				[]byte(`{"id": 4, "name": "test4", "tick": 2}`),
				[]byte(`{"id": 5, "name": "test5", "tick": 2}`),
				[]byte(`{"id": 6, "name": "test6", "tick": 2}`),
			},
			Topic: "test",
			Tick:  2 * time.Second,
			Cycle: false,
		},
		{
			Message: []json.RawMessage{
				[]byte(`{"id": 7, "name": "test7", "tick": 3}`),
				[]byte(`{"id": 8, "name": "test8", "tick": 3}`),
				[]byte(`{"id": 9, "name": "test9", "tick": 3}`),
				[]byte(`{"id": 10, "name": "test10", "tick": 3}`),
			},
			Topic: "test_another",
			Tick:  3 * time.Second,
			Cycle: true,
		},
	}))

	if err := consumer.AddHandler("test", f("test")); err != nil {
		panic(err)
	}

	if err := consumer.AddHandler("test_another", f("test_another")); err != nil {
		panic(err)
	}

	if err := consumer.Consume(); err != nil {
		panic(err)
	}

	time.Sleep(10 * time.Second)

	consumer.PauseAll()
	fmt.Println("pause all")

	time.Sleep(5 * time.Second)

	fmt.Println("resume all")
	consumer.ResumeAll()

	time.Sleep(10 * time.Second)

	if err := consumer.Close(); err != nil {
		panic(err)
	}
}
