// Package consumer decide o que fazer com cada mensagem da fila de falhas.
package consumer

import (
	"github.com/fiapx/fiapx-notification-service/internal/events"
)

// Outcome diz ao laço de consumo como responder ao broker.
type Outcome struct {
	// Requeue devolve a mensagem à fila para uma nova tentativa.
	Requeue bool
	// Err é nil quando a notificação saiu.
	Err error
}

// Notifier é o que o consumidor precisa saber sobre o envio de e-mail.
type Notifier interface {
	Notify(events.VideoEvent) error
}

// Process trata uma mensagem.
//
// Evento ilegível é descartado: retentar não vai torná-lo válido. Falha de envio é
// retentada uma única vez; insistir em uma mensagem já reentregue transformaria a
// fila em um laço quente contra o servidor SMTP.
func Process(body []byte, redelivered bool, notifier Notifier) Outcome {
	event, err := events.Parse(body)
	if err != nil {
		return Outcome{Requeue: false, Err: err}
	}
	if err := notifier.Notify(event); err != nil {
		return Outcome{Requeue: !redelivered, Err: err}
	}
	return Outcome{}
}
