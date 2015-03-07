package main

import (
	"./common"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	//"time"

	"github.com/streadway/amqp"
	"github.com/twinj/uuid"
)

var (
	url   = flag.String("u", "amqp://guest:guest@localhost:5672/", "The url to rabbitmq")
	queue = flag.String("q", "mstest", "The queue to use")
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func main() {
	flag.Parse()

	u := uuid.NewV4()
	workerId := u.String()
	log.Printf("Starting worker %s", workerId)

	log.Printf("Connecting to %s", *url)
	conn, err := amqp.Dial(*url)
	failOnError(err, "Failed to connect to RabbitMQ.")
	defer conn.Close()

	log.Println("Opening a channel")
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	log.Printf("Declaring a queue named %s", *queue)
	q, err := ch.QueueDeclare(
		*queue, // name
		false,  // durable
		false,  // delete when usused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			// disabling screen output to increase throughput
			log.Print("incoming")

			response := &common.HealthCheck{
				Healthy: true}

			response.WorkerId = workerId

			jresponse, _ := json.Marshal(response)

			err = ch.Publish(
				"",        // exchange
				d.ReplyTo, // routing key
				false,     // mandatory
				false,     // immediate
				amqp.Publishing{
					ContentType:   "text/plain",
					CorrelationId: d.CorrelationId,
					Body:          jresponse,
				})
			failOnError(err, "Failed to publish a message")

			d.Ack(false)
		}
	}()

	log.Printf(" [*] Awaiting requests")
	<-forever
}
