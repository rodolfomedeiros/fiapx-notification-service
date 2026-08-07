// Consome a fila de falhas de processamento e avisa o usuário por e-mail.
package main

import (
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/fiapx/fiapx-notification-service/internal/consumer"
	"github.com/fiapx/fiapx-notification-service/internal/mailer"
	"github.com/fiapx/fiapx-notification-service/internal/metrics"
)

const (
	exchange   = "video.events"
	queue      = "video-processing-dlq"
	routingKey = "video.failed"
	consumerID = "fiapx-notification"
)

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func dial(url string) *amqp.Connection {
	for {
		connection, err := amqp.Dial(url)
		if err == nil {
			return connection
		}
		log.Printf("RabbitMQ indisponível: %v", err)
		time.Sleep(3 * time.Second)
	}
}

// consume roda até a conexão cair, para que main possa reconectar.
func consume(url string, notifier consumer.Notifier) error {
	connection := dial(url)
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := channel.QueueBind(queue, routingKey, exchange, false, nil); err != nil {
		return err
	}
	deliveries, err := channel.Consume(queue, consumerID, false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Println("notification-service aguardando eventos de falha")
	metrics.Connected.Set(1)
	defer metrics.Connected.Set(0)

	for delivery := range deliveries {
		started := time.Now()
		outcome := consumer.Process(delivery.Body, delivery.Redelivered, notifier)
		metrics.Observe(started)
		switch {
		case outcome.Err == nil:
			metrics.Messages.WithLabelValues("notified").Inc()
			_ = delivery.Ack(false)
		case outcome.Requeue:
			log.Printf("reentregando mensagem: %v", outcome.Err)
			metrics.Messages.WithLabelValues("requeued").Inc()
			_ = delivery.Nack(false, true)
		default:
			log.Printf("descartando mensagem: %v", outcome.Err)
			metrics.Messages.WithLabelValues("dropped").Inc()
			_ = delivery.Nack(false, false)
		}
	}
	return nil
}

func main() {
	metrics.Serve(getenv("METRICS_ADDR", ":9100"))
	url := getenv("RABBITMQ_URL", "amqp://fiapx:fiapx@localhost:5672/%2F")
	notifier := mailer.New(
		getenv("SMTP_HOST", "localhost"),
		getenv("SMTP_PORT", "1025"),
		getenv("SMTP_FROM", "no-reply@fiapx.local"),
	)
	// O canal de entregas fecha quando a conexão cai; sem este laço o processo
	// terminava em silêncio e as falhas deixavam de ser notificadas.
	for {
		if err := consume(url, notifier); err != nil {
			log.Printf("consumo interrompido: %v", err)
		}
		log.Println("reconectando ao RabbitMQ em 3s")
		time.Sleep(3 * time.Second)
	}
}
