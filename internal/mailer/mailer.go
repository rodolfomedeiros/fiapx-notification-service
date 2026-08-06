// Package mailer monta e envia o e-mail que avisa o usuário sobre a falha no processamento.
package mailer

import (
	"fmt"
	"mime"
	"net/smtp"
	"strings"
	"time"

	"github.com/fiapx/fiapx-notification-service/internal/events"
)

const subject = "[FIAP X] Falha no processamento do vídeo"

// SendFunc tem a assinatura de smtp.SendMail, para que os testes não abram conexão.
type SendFunc func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error

type Mailer struct {
	Host string
	Port string
	From string
	Send SendFunc
	Now  func() time.Time
}

func New(host, port, from string) *Mailer {
	return &Mailer{Host: host, Port: port, From: from, Send: smtp.SendMail, Now: time.Now}
}

func (m *Mailer) address() string {
	return m.Host + ":" + m.Port
}

// Compose monta a mensagem RFC 5322 completa, com Date e assunto codificado.
func (m *Mailer) Compose(event events.VideoEvent) []byte {
	body := strings.Join([]string{
		"Olá,",
		"",
		fmt.Sprintf("Não foi possível processar o vídeo %s após %d tentativas.", event.VideoID, event.Attempt+1),
		"",
		"Motivo: " + event.Reason(),
		"",
		"Envie o arquivo novamente. Se o erro persistir, fale com o suporte.",
		"",
		"Equipe FIAP X",
	}, "\r\n")

	headers := []string{
		"From: " + m.From,
		"To: " + event.UserEmail,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"Date: " + m.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body + "\r\n")
}

func (m *Mailer) Notify(event events.VideoEvent) error {
	if err := m.Send(m.address(), nil, m.From, []string{event.UserEmail}, m.Compose(event)); err != nil {
		return fmt.Errorf("envio para %s falhou: %w", event.UserEmail, err)
	}
	return nil
}
