package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type ConsumerRef struct {
	Stream   string
	Consumer string
}

func main() {
	natsURL := "nats://localhost:30222"
	username := "js"
	password := "js"

	consumers := []ConsumerRef{
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
		{Stream: "cob2-orders-internal-partitioned", Consumer: "cob2-place-partition-consumer-0"},
	}

	// Single NATS connection
	nc, err := nats.Connect(
		natsURL,
		nats.UserInfo(username, password),
	)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()

	// Single JetStream context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Error creating JetStream context: %v", err)
	}

	// Poll loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("---- Consumer Stats ----")

		for _, c := range consumers {
			ci, err := js.ConsumerInfo(c.Stream, c.Consumer)
			if err != nil {
				log.Printf("Error fetching consumer info (%s/%s): %v",
					c.Stream, c.Consumer, err)
				continue
			}

			printConsumerStats(ci)
		}

		fmt.Println()
	}
}

func printConsumerStats(ci *nats.ConsumerInfo) {
	fmt.Printf(
		"[%s]\n"+
			"  Delivered:     %d\n"+
			"  Ack Floor:     %d\n"+
			"  Pending:       %d\n"+
			"  Redelivered:   %d\n"+
			"  Last Delivered:%d\n"+
			"  Num Ack Pending:%d\n",
		ci.Name,
		ci.Delivered.Consumer,
		ci.AckFloor.Consumer,
		ci.NumPending,
		ci.NumRedelivered,
		ci.Delivered.Stream,
		ci.NumAckPending,
	)
}
