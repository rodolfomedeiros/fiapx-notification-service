// Package events decodifica o envelope compartilhado em contracts/video-event.schema.json.
package events

import (
	"encoding/json"
	"fmt"
	"strings"
)

// VideoEvent traz apenas os campos que a notificação usa; os demais são ignorados.
type VideoEvent struct {
	EventID      string `json:"event_id"`
	EventType    string `json:"event_type"`
	OccurredAt   string `json:"occurred_at"`
	VideoID      string `json:"video_id"`
	UserID       string `json:"user_id"`
	UserEmail    string `json:"user_email"`
	Attempt      int    `json:"attempt"`
	ErrorMessage string `json:"error_message"`
}

// Parse decodifica e recusa o que não dá para notificar.
func Parse(data []byte) (VideoEvent, error) {
	var event VideoEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return VideoEvent{}, fmt.Errorf("evento ilegível: %w", err)
	}
	if event.VideoID == "" {
		return VideoEvent{}, fmt.Errorf("evento sem video_id")
	}
	// Um \r ou \n no endereço permitiria injetar cabeçalhos na mensagem SMTP.
	if strings.ContainsAny(event.UserEmail, "\r\n") {
		return VideoEvent{}, fmt.Errorf("user_email com quebra de linha")
	}
	event.UserEmail = strings.TrimSpace(event.UserEmail)
	if !strings.Contains(event.UserEmail, "@") {
		return VideoEvent{}, fmt.Errorf("evento sem user_email utilizável, não há para quem notificar")
	}
	return event, nil
}

// Reason devolve o motivo da falha, com um texto padrão quando o worker não informou.
func (e VideoEvent) Reason() string {
	reason := strings.TrimSpace(e.ErrorMessage)
	if reason == "" {
		return "motivo não informado pelo processador"
	}
	return strings.ReplaceAll(strings.ReplaceAll(reason, "\r", " "), "\n", " ")
}
