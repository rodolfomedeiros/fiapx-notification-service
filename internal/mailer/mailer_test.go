package mailer

import (
	"errors"
	"net/smtp"
	"strings"
	"testing"
	"time"

	"github.com/fiapx/fiapx-notification-service/internal/events"
)

type envio struct {
	addr string
	from string
	to   []string
	msg  []byte
}

func fixo() time.Time {
	return time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
}

func mailerDeTeste(erro error) (*Mailer, *[]envio) {
	var enviados []envio
	m := New("mailpit", "1025", "no-reply@fiapx.local")
	m.Now = fixo
	m.Send = func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		enviados = append(enviados, envio{addr, from, to, msg})
		return erro
	}
	return m, &enviados
}

func falha() events.VideoEvent {
	return events.VideoEvent{
		VideoID:      "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		UserEmail:    "ana@example.com",
		Attempt:      2,
		ErrorMessage: "ffmpeg falhou",
	}
}

func TestMensagemTrazOsCabecalhosObrigatorios(t *testing.T) {
	m, _ := mailerDeTeste(nil)

	mensagem := string(m.Compose(falha()))

	cabecalhos, corpo, encontrou := strings.Cut(mensagem, "\r\n\r\n")
	if !encontrou {
		t.Fatal("mensagem sem separação entre cabeçalhos e corpo")
	}
	for _, esperado := range []string{
		"From: no-reply@fiapx.local",
		"To: ana@example.com",
		"Date: Thu, 06 Aug 2026 10:00:00 +0000",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	} {
		if !strings.Contains(cabecalhos, esperado) {
			t.Errorf("faltou o cabeçalho %q", esperado)
		}
	}
	if strings.TrimSpace(corpo) == "" {
		t.Error("corpo vazio")
	}
}

func TestAssuntoComAcentoEhCodificadoParaTransporte(t *testing.T) {
	m, _ := mailerDeTeste(nil)

	mensagem := string(m.Compose(falha()))

	if !strings.Contains(mensagem, "Subject: =?utf-8?q?") {
		t.Errorf("assunto não foi codificado: %q", mensagem)
	}
	if strings.Contains(mensagem, "Subject: [FIAP X] Falha no processamento do vídeo") {
		t.Error("assunto com acento foi enviado sem codificação")
	}
}

func TestCorpoExplicaOVideoOMotivoEAsTentativas(t *testing.T) {
	m, _ := mailerDeTeste(nil)

	corpo := string(m.Compose(falha()))

	for _, esperado := range []string{"6ba7b810-9dad-11d1-80b4-00c04fd430c8", "ffmpeg falhou", "3 tentativas"} {
		if !strings.Contains(corpo, esperado) {
			t.Errorf("corpo não menciona %q", esperado)
		}
	}
}

func TestCorpoUsaMotivoPadraoQuandoOWorkerNaoInformou(t *testing.T) {
	m, _ := mailerDeTeste(nil)
	evento := falha()
	evento.ErrorMessage = ""

	if !strings.Contains(string(m.Compose(evento)), "motivo não informado") {
		t.Error("sem error_message o corpo deveria trazer o texto padrão")
	}
}

func TestNotifyEnviaParaODonoDoVideoNoServidorConfigurado(t *testing.T) {
	m, enviados := mailerDeTeste(nil)

	if err := m.Notify(falha()); err != nil {
		t.Fatalf("Notify devolveu erro: %v", err)
	}

	if len(*enviados) != 1 {
		t.Fatalf("esperava 1 envio, houve %d", len(*enviados))
	}
	enviado := (*enviados)[0]
	if enviado.addr != "mailpit:1025" {
		t.Errorf("addr = %q", enviado.addr)
	}
	if enviado.from != "no-reply@fiapx.local" {
		t.Errorf("from = %q", enviado.from)
	}
	if len(enviado.to) != 1 || enviado.to[0] != "ana@example.com" {
		t.Errorf("to = %v", enviado.to)
	}
}

func TestNotifyPropagaFalhaDoServidorSmtp(t *testing.T) {
	m, _ := mailerDeTeste(errors.New("conexão recusada"))

	err := m.Notify(falha())

	if err == nil {
		t.Fatal("falha do SMTP deveria ser propagada")
	}
	if !strings.Contains(err.Error(), "ana@example.com") {
		t.Errorf("o erro deveria dizer para quem o envio falhou: %v", err)
	}
}
