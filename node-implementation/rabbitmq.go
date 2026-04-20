// Author: Tomáš Hladký
// Master thesis: Design an Experimental PoS DAG-based Blockchain Consensual Protocol
// Brno University of Technology, Faculty of Information Technology, 2025

package main

import (
	"github.com/rabbitmq/amqp091-go"
	"github.com/wagslane/go-rabbitmq"
)

func initRabbitMqConnection() *rabbitmq.Conn {
	logInfo(RMQ_LOG, "Connecting to RabbitMQ... %s", "amqp://"+rabbitmqHostname)
	conn, err := rabbitmq.NewConn(
		"amqp://" + rabbitmqHostname,
	)
	if err != nil {
		logFatal(RMQ_LOG, "Failed to connect to RabbitMQ: %s", err)
	}

	return conn
}

func initRabbitMqConsumers(nodeId string) *rabbitmq.Consumer {
	consumerCommand, err := rabbitmq.NewConsumer(
		rabbitMqConn,
		"backend_commands_"+nodeId,
		rabbitmq.WithConsumerOptionsExchangeKind(amqp091.ExchangeDirect),
		rabbitmq.WithConsumerOptionsExchangeName("commands_exchange"),
		rabbitmq.WithConsumerOptionsConsumerName(nodeId),
		rabbitmq.WithConsumerOptionsRoutingKey(nodeId),
		rabbitmq.WithConsumerOptionsExchangeDeclare,
	)
	if err != nil {
		logFatal(RMQ_LOG, "Failed to create RabbitMQ consumer: %s", err)
	}

	return consumerCommand
}

func initRabbitMqPublisher() *rabbitmq.Publisher {
	publisherBlockchainData, err := rabbitmq.NewPublisher(
		rabbitMqConn,
		rabbitmq.WithPublisherOptionsExchangeName("blockchain_data"),
		rabbitmq.WithPublisherOptionsExchangeDeclare,
	)
	if err != nil {
		logFatal(RMQ_LOG, "Failed to create RabbitMQ publisher: %s", err)
	}

	return publisherBlockchainData
}

func runRabbitMqCommandConsumer(consumer *rabbitmq.Consumer) {
	for {
		err := consumer.Run(func(d rabbitmq.Delivery) rabbitmq.Action {
			logDebug(RMQ_LOG, "Consumed: %v", string(d.Body))
			processQueueCommandMessage(d.Body)
			return rabbitmq.Ack
		})
		if err != nil {
			logError(RMQ_LOG, "Failed to consume RabbitMQ message: %s", err)
		}
	}
}
