# fiapx-notification-service

O último elo do **FIAP X**: escuta a fila de falhas e avisa o usuário por e-mail quando o
processamento de um vídeo não deu certo, explicando o motivo.

**Go 1.25** · amqp091-go · prometheus/client_golang

## Repositórios do projeto

| Repositório | Linguagem | Papel |
| :--- | :--- | :--- |
| [fiapx-platform](https://github.com/rodolfomedeiros/fiapx-platform) | — | Compose, Kubernetes, contratos, topologia do broker |
| [fiapx-auth-service](https://github.com/rodolfomedeiros/fiapx-auth-service) | Java 25 · Spring Boot 4 | Cadastro, login, emissão e introspecção de JWT |
| [fiapx-video-management-service](https://github.com/rodolfomedeiros/fiapx-video-management-service) | Python 3.13 · FastAPI | Upload, listagem, download e WebSocket de tempo real |
| [fiapx-video-processor-worker](https://github.com/rodolfomedeiros/fiapx-video-processor-worker) | Rust 1.94 · Tokio | Extração de quadros com FFmpeg e compactação em `.zip` |
| **fiapx-notification-service** *(você está aqui)* | Go 1.25 | Consumo da DLQ e envio de e-mail de falha |
| [fiapx-web](https://github.com/rodolfomedeiros/fiapx-web) | React 19 · TypeScript 6 | Interface de upload, acompanhamento e download |

> Para subir o sistema inteiro, use o **fiapx-platform**. Este repositório sozinho precisa
> de RabbitMQ e um servidor SMTP acessíveis.

## Onde este serviço entra

```mermaid
flowchart TB
    VMS["video-management-service<br/>Python 3.13"]
    Worker["video-processor-worker<br/>Rust 1.94"]
    Notif["notification-service<br/>Go 1.25"]
    MQ{{"RabbitMQ<br/>video.events"}}
    DLQ[["video-processing-dlq"]]
    SMTP[/"SMTP<br/>Mailpit · SES · SendGrid"/]
    User(["Usuário"])

    VMS -->|"video.received"| MQ
    MQ --> Worker
    Worker -->|"video.failed<br/>tentativas esgotadas"| MQ
    MQ -.->|"dead letter<br/>mensagem rejeitada"| DLQ
    MQ -->|"video.failed"| DLQ
    DLQ --> Notif
    Notif --> SMTP
    SMTP --> User

    classDef atual stroke:#e8590c,stroke-width:4px
    class Notif atual
```

O serviço recebe mensagens por **dois caminhos**, sem precisar distinguir entre eles:

1. O worker publica `video.failed` explicitamente, depois de esgotar as três tentativas.
2. `video-processing-queue` é declarada com `x-dead-letter-routing-key: video.failed`, então
   qualquer mensagem rejeitada sem reenfileiramento cai sozinha na DLQ.

## Política de reentrega

```mermaid
flowchart LR
    A["mensagem da DLQ"] --> B{"JSON válido<br/>e com destinatário?"}
    B -->|não| C["descarta<br/>nack sem requeue"]
    B -->|sim| D["monta e envia<br/>o e-mail"]
    D --> E{"SMTP aceitou?"}
    E -->|sim| F["confirma<br/>ack"]
    E -->|não| G{"já foi<br/>reentregue?"}
    G -->|não| H["devolve à fila<br/>nack com requeue"]
    G -->|sim| I["descarta<br/>nack sem requeue"]
```

Evento ilegível ou sem destinatário é **descartado**, porque retentar não o tornaria válido.
Falha de envio é reentregue **uma única vez**: insistir em uma mensagem já reentregue
transformaria a fila em um laço quente contra o servidor SMTP.

O consumidor **reconecta automaticamente** ao broker. Sem isso, a queda da conexão encerraria
o processo em silêncio e as falhas deixariam de ser notificadas até alguém reparar.

## O e-mail

Mensagem RFC 5322 completa, em texto puro UTF-8:

```
From: no-reply@fiapx.local
To: ana@example.com
Subject: =?utf-8?q?...?=          (assunto acentuado, codificado em MIME)
Date: Thu, 06 Aug 2026 10:00:00 +0000
MIME-Version: 1.0
Content-Type: text/plain; charset=UTF-8

Olá,

Não foi possível processar o vídeo <id> após 3 tentativas.

Motivo: <mensagem de erro do processador>

Envie o arquivo novamente. Se o erro persistir, fale com o suporte.

Equipe FIAP X
```

Detalhes que importam:

- `From` e `Date` estão presentes porque vários servidores classificam como spam mensagens
  sem eles.
- O assunto é codificado em MIME, já que contém acento.
- O endereço do destinatário é **validado contra CRLF**: um `\r\n` no `user_email`
  permitiria injetar cabeçalhos arbitrários na mensagem SMTP.
- A contagem exibida é `attempt + 1`, porque o contador do evento é indexado em zero.

## Configuração

| Variável | Padrão | Descrição |
| :--- | :--- | :--- |
| `RABBITMQ_URL` | `amqp://fiapx:fiapx@localhost:5672/%2F` | Barramento |
| `SMTP_HOST` | `localhost` | Servidor SMTP |
| `SMTP_PORT` | `1025` | Porta SMTP (1025 é o padrão do Mailpit) |
| `SMTP_FROM` | `no-reply@fiapx.local` | Remetente |
| `METRICS_ADDR` | `:9100` | Endereço de `/metrics` e `/health` |

Em desenvolvimento o Compose aponta para o **Mailpit**, que captura tudo em
http://localhost:8025 sem enviar nada de verdade. Em produção, aponte para SES, SendGrid ou
outro provedor.

## Organização

| Pacote | Responsabilidade |
| :--- | :--- |
| `internal/events` | Leitura do envelope compartilhado e recusa do que não dá para notificar |
| `internal/mailer` | Montagem da mensagem RFC 5322 e envio por SMTP |
| `internal/consumer` | Decisão sobre cada mensagem: confirmar, reentregar ou descartar |
| `internal/metrics` | Endpoints `/metrics` e `/health` |
| `cmd/notification` | Conexão com o RabbitMQ e laço de consumo |

A decisão sobre cada mensagem vive isolada do transporte, o que a torna testável sem broker.

## Métricas

Servidor HTTP próprio em `:9100`, com `/metrics` e `/health`.

| Métrica | O que mede |
| :--- | :--- |
| `fiapx_notification_messages_total{outcome}` | Mensagens por desfecho: `notified`, `requeued`, `dropped` |
| `fiapx_notification_send_duration_seconds` | Histograma da duração do envio SMTP |
| `fiapx_notification_broker_connected` | `1` enquanto o consumidor está ligado ao broker |

## Executar

```sh
go run ./cmd/notification
```

Com a infraestrutura do Compose de pé, os padrões das variáveis funcionam sem configuração
adicional.

## Testes

```sh
go test ./...
```

23 testes cobrindo leitura do envelope (incluindo a recusa de endereços com CRLF), montagem
e envio do e-mail, política de reentrega e os endpoints de métricas. Os testes substituem
`smtp.SendMail` por uma função de mentira, então não abrem conexão.

O CI ainda roda `gofmt`, `go vet` e um piso de **80% de cobertura** sobre `internal/`:

```sh
go test -coverprofile=coverage.out ./internal/... && go tool cover -func=coverage.out
```

`cmd/` fica de fora porque é só a fiação do processo; a decisão mora em `internal/`, hoje em
97,5%. É o mesmo recorte que o `gear-up` faz ao excluir `drivers/` do JaCoCo.

## Contrato de eventos

`contracts/video-event.schema.json` é uma cópia do contrato canônico mantido em
[fiapx-platform](https://github.com/rodolfomedeiros/fiapx-platform).

Este serviço **consome** `video.failed` e não publica nada no barramento. Do envelope, usa
`video_id`, `user_email`, `attempt` e `error_message`; campos desconhecidos são ignorados,
para que o contrato possa crescer sem quebrar este consumidor.
