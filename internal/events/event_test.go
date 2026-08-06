package events

import "testing"

const valido = `{
  "event_id": "3f2504e0-4f89-11d3-9a0c-0305e82c3301",
  "event_type": "video.failed",
  "occurred_at": "2026-08-06T10:00:00+00:00",
  "video_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "user_id": "6ba7b811-9dad-11d1-80b4-00c04fd430c8",
  "attempt": 2,
  "raw_file_path": "raw/ana/ferias.mp4",
  "error_message": "ffmpeg falhou",
  "user_email": "ana@example.com"
}`

func TestParseAceitaEventoPublicadoPeloWorker(t *testing.T) {
	event, err := Parse([]byte(valido))
	if err != nil {
		t.Fatalf("evento válido recusado: %v", err)
	}
	if event.UserEmail != "ana@example.com" {
		t.Errorf("user_email = %q", event.UserEmail)
	}
	if event.VideoID != "6ba7b810-9dad-11d1-80b4-00c04fd430c8" {
		t.Errorf("video_id = %q", event.VideoID)
	}
	if event.Attempt != 2 {
		t.Errorf("attempt = %d", event.Attempt)
	}
	if event.ErrorMessage != "ffmpeg falhou" {
		t.Errorf("error_message = %q", event.ErrorMessage)
	}
}

func TestParseRecusaMensagensQueNaoDaoParaNotificar(t *testing.T) {
	casos := map[string]string{
		"json quebrado":        `{"video_id":`,
		"sem video_id":         `{"user_email":"ana@example.com"}`,
		"sem user_email":       `{"video_id":"abc"}`,
		"user_email em branco": `{"video_id":"abc","user_email":"   "}`,
	}
	for nome, corpo := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, err := Parse([]byte(corpo)); err == nil {
				t.Fatal("deveria recusar")
			}
		})
	}
}

func TestParseRecusaEnderecoComQuebraDeLinha(t *testing.T) {
	injecao := `{"video_id":"abc","user_email":"ana@example.com\r\nBcc: vitima@example.com"}`

	if _, err := Parse([]byte(injecao)); err == nil {
		t.Fatal("um endereço com CRLF permitiria injetar cabeçalhos SMTP")
	}
}

func TestParseIgnoraCamposDesconhecidosDoContrato(t *testing.T) {
	comExtras := `{"video_id":"abc","user_email":"ana@example.com","zip_file_path":"x.zip","frame_count":10}`

	if _, err := Parse([]byte(comExtras)); err != nil {
		t.Fatalf("campos extras não deveriam quebrar a leitura: %v", err)
	}
}

func TestReasonUsaTextoPadraoQuandoNaoHaMotivo(t *testing.T) {
	if got := (VideoEvent{}).Reason(); got != "motivo não informado pelo processador" {
		t.Errorf("Reason() = %q", got)
	}
}

func TestReasonAchataQuebrasDeLinhaDoMotivo(t *testing.T) {
	event := VideoEvent{ErrorMessage: "ffmpeg falhou:\r\nstream not found"}

	got := event.Reason()

	if got != "ffmpeg falhou:  stream not found" {
		t.Errorf("Reason() = %q", got)
	}
}
