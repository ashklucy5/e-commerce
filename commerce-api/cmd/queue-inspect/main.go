package main

import (
	"context"
	"fmt"
	"log"

	"project.local/commerce-api/internal/platform/cache"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/queue"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	client, err := cache.NewRedis(
		ctx,
		cfg,
	)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()

	queueConfig :=
		queue.DefaultConfig()

	pong, err :=
		client.Ping(
			ctx,
		).Result()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(
		"Redis:",
		pong,
	)

	streamLength, err :=
		client.XLen(
			ctx,
			queueConfig.Stream,
		).Result()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(
		"Live stream length:",
		streamLength,
	)

	groups, err :=
		client.XInfoGroups(
			ctx,
			queueConfig.Stream,
		).Result()
	if err != nil {
		fmt.Println(
			"Group info error:",
			err,
		)
	} else {
		fmt.Println(
			"\nConsumer groups:",
		)

		for _, group := range groups {

			fmt.Printf(
				"%+v\n",
				group,
			)
		}
	}

	pending, err :=
		client.XPending(
			ctx,
			queueConfig.Stream,
			queueConfig.Group,
		).Result()
	if err != nil {
		fmt.Println(
			"Pending info error:",
			err,
		)
	} else {
		fmt.Printf(
			"\nPending: %+v\n",
			pending,
		)
	}

	retryCount, err :=
		client.ZCard(
			ctx,
			queueConfig.RetrySet,
		).Result()
	if err != nil {
		fmt.Println(
			"Retry-set error:",
			err,
		)
	} else {
		fmt.Println(
			"Retry jobs:",
			retryCount,
		)
	}

	deadLength, err :=
		client.XLen(
			ctx,
			queueConfig.DeadLetterStream,
		).Result()
	if err != nil {
		fmt.Println(
			"Dead-letter length error:",
			err,
		)
	} else {
		fmt.Println(
			"Dead-letter jobs:",
			deadLength,
		)
	}

	fmt.Println(
		"\nRecent live jobs:",
	)

	liveJobs, err :=
		client.XRevRangeN(
			ctx,
			queueConfig.Stream,
			"+",
			"-",
			20,
		).Result()
	if err != nil {
		fmt.Println(
			"Read live stream error:",
			err,
		)
	} else {
		for _, item := range liveJobs {

			fmt.Printf(
				"id=%s type=%v attempt=%v payload=%v\n",
				item.ID,
				item.Values["job_type"],
				item.Values["attempt"],
				item.Values["payload"],
			)
		}
	}

	fmt.Println(
		"\nRecent dead-letter jobs:",
	)

	deadJobs, err :=
		client.XRevRangeN(
			ctx,
			queueConfig.DeadLetterStream,
			"+",
			"-",
			20,
		).Result()
	if err != nil {
		fmt.Println(
			"Read dead-letter stream error:",
			err,
		)
	} else {
		for _, item := range deadJobs {

			fmt.Printf(
				"id=%s type=%v attempt=%v error=%v payload=%v\n",
				item.ID,
				item.Values["job_type"],
				item.Values["attempt"],
				item.Values["error"],
				item.Values["payload"],
			)
		}
	}
}
