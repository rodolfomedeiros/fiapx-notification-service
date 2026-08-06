package consumer

import (
	"errors"
	"testing"

	"github.com/fiapx/fiapx-notification-service/internal/events"
)

type notifierFalso struct {
	recebidos []events.VideoEvent
	erro      error
}

func (n *notifierFalso) Notify(event events.VideoEvent) error {
	n.recebidos = append(n.recebidos, event)
	return n.erro
}

const falha = `{"video_id":"abc","user_email":"ana@example.com","attempt":2,"error_message":"ffmpeg falhou"}`

func TestNotificaEConfirmaAMensagem(t *testing.T) {
	notifier := &notifierFalso{}

	outcome := Process([]byte(falha), false, notifier)

	if outcome.Err != nil {
		t.Fatalf("Err = %v", outcome.Err)
	}
	if outcome.Requeue {
		t.Error("uma notificação enviada não deve voltar para a fila")
	}
	if len(notifier.recebidos) != 1 || notifier.recebidos[0].UserEmail != "ana@example.com" {
		t.Errorf("recebidos = %v", notifier.recebidos)
	}
}

func TestDescartaEventoIlegivelSemTentarEnviar(t *testing.T) {
	notifier := &notifierFalso{}

	outcome := Process([]byte(`{"video_id":`), false, notifier)

	if outcome.Err == nil {
		t.Fatal("evento ilegível deveria reportar erro")
	}
	if outcome.Requeue {
		t.Error("retentar não vai tornar o JSON válido")
	}
	if len(notifier.recebidos) != 0 {
		t.Error("não deveria ter tentado enviar")
	}
}

func TestDescartaEventoSemDestinatario(t *testing.T) {
	outcome := Process([]byte(`{"video_id":"abc"}`), false, &notifierFalso{})

	if outcome.Err == nil || outcome.Requeue {
		t.Errorf("outcome = %+v", outcome)
	}
}

func TestReentregaUmaVezQuandoOSmtpFalha(t *testing.T) {
	notifier := &notifierFalso{erro: errors.New("conexão recusada")}

	outcome := Process([]byte(falha), false, notifier)

	if outcome.Err == nil {
		t.Fatal("a falha do SMTP deveria ser reportada")
	}
	if !outcome.Requeue {
		t.Error("a primeira falha de envio merece uma nova tentativa")
	}
}

func TestNaoInsisteEmMensagemJaReentregue(t *testing.T) {
	notifier := &notifierFalso{erro: errors.New("conexão recusada")}

	outcome := Process([]byte(falha), true, notifier)

	if outcome.Requeue {
		t.Error("insistir em uma mensagem já reentregue vira laço quente contra o SMTP")
	}
}
