# fiapx-notification-service

Consumidor Go da DLQ. Em desenvolvimento, envie e-mails ao Mailpit configurado pelo Compose.

## Organização

- `internal/events` — leitura do envelope de `contracts/video-event.schema.json` e recusa do que não dá para notificar.
- `internal/mailer` — montagem da mensagem RFC 5322 e envio por SMTP.
- `internal/consumer` — o que fazer com cada mensagem: confirmar, reentregar ou descartar.
- `cmd/notification` — conexão com o RabbitMQ e laço de consumo.

## Política de reentrega

Evento ilegível ou sem destinatário é descartado, porque retentar não o tornaria válido.
Falha de envio é reentregue uma única vez: insistir em uma mensagem já reentregue
transformaria a fila em um laço quente contra o servidor SMTP.

## Testes

```sh
go test ./...
```

Os testes substituem `smtp.SendMail` por uma função de mentira, então não abrem conexão.
