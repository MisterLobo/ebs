package lib

import (
	"context"
	"ebs/src/types"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func GetKafkaProducerConfig() kafka.ConfigMap {
	return kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BROKER"),
		"client.id":         "myProducer",
		"acks":              "all",
	}
}

func GetKafkaConsumerConfig() kafka.ConfigMap {
	return kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BROKER"),
		"group.id":          "foo",
		"auto.offset.reset": "smallest",
	}
}

func KafkaConsumers(groupId string, topics ...string) {
	log.Println("Initializing kafka Consumer...")
	master, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BROKER"),
		"group.id":          groupId,
		"auto.offset.reset": "smallest",
	})

	if err != nil {
		log.Printf("Error on master: %s\n", err.Error())
		return
	}
	err = master.SubscribeTopics(topics, func(c *kafka.Consumer, e kafka.Event) error {
		log.Println("[KAFKA] Consumer rebalancing...")
		return nil
	})
	if err != nil {
		log.Printf("Error on consumer: %s\n", err.Error())
		return
	}
	go func() {
		log.Println("[BACKGROUND]: waiting for messages...")
		run := true
		for run == true {
			ev := master.Poll(100)
			switch e := ev.(type) {
			case *kafka.Message:
				log.Printf("message received: %s\n", string(e.Value))
				break
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
				run = false
			default:
				break
			}
		}
		master.Close()
	}()
}

func KafkaConsumer(groupId string, topic string, fn *types.Handler) {
	log.Println("Initializing kafka Consumer...")
	master, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BROKER"),
		"group.id":          groupId,
		"auto.offset.reset": "smallest",
	})

	if err != nil {
		log.Printf("Error on master: %s\n", err.Error())
		return
	}
	err = master.SubscribeTopics([]string{topic}, func(c *kafka.Consumer, e kafka.Event) error {
		log.Printf("[KAFKA:%s] Consumer rebalancing...\n", topic)
		return nil
	})
	if err != nil {
		log.Printf("Error on consumer: %s\n", err.Error())
		return
	}
	go func() {
		log.Println("[BACKGROUND]: waiting for messages...")
		run := true
		for run == true {
			ev := master.Poll(100)
			switch e := ev.(type) {
			case *kafka.Message:
				log.Printf("message received: %s\n", string(e.Value))
				h := *fn
				h(string(e.Value))
				break
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
				run = false
			default:
				break
			}
		}
		master.Close()
	}()
}

func KafkaProducer(clientId string) {
	broker := os.Getenv("KAFKA_BROKER")
	log.Printf("Broker: %s\n", broker)
	log.Println("Initializing kafka Producer...")
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BROKER"),
		"client.id":         clientId,
		"acks":              "all",
	})

	if err != nil {
		log.Printf("Error on producer: %s\n", err.Error())
		return
	}

	topic := "topic3"
	err = p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte("test value"),
	}, nil)
}

func KafkaProduceMessage(clientId, topic string, payload *types.JSONB) error {
	log.Println("STEP 1: Create producer")
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BROKER"),
		"client.id":         clientId,
		"acks":              "all",
	})
	if err != nil {
		log.Printf("Error on 1st steap: %s\n", err.Error())
		return err
	}
	log.Println("1st step PASS")

	log.Println("STEP 2: Processing payload")
	value, err := json.Marshal(*payload)
	if err != nil {
		log.Printf("Error on 2nd step: %s\n", err.Error())
		return err
	}
	log.Println("2nd step PASS")

	log.Println("STEP 3: Send data to queue")
	err = p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          value,
	}, nil)
	if err != nil {
		log.Printf("Error on 3rd step: %s\n", err.Error())
		return err
	}
	log.Println("3rd step PASS")
	return nil
}

func KafkaCreateTopics(topics ...string) ([]kafka.TopicResult, error) {
	a, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BROKER"),
	})
	if err != nil {
		log.Printf("Error on AdminClient: %s\n", err.Error())
		return nil, err
	}
	topicsDef := []kafka.TopicSpecification{}
	for _, topic := range topics {
		topicsDef = append(topicsDef, kafka.TopicSpecification{
			Topic:         topic,
			NumPartitions: 10,
		})
	}
	result, err := a.CreateTopics(context.Background(), topicsDef)
	if err != nil {
		log.Printf("Error creating topics: %s\n", err.Error())
		return nil, err
	}
	return result, nil
}
