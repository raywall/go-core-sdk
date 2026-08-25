# go-core-sdk

`go-core-sdk` reune packages e services Go autocontidos para uso em aplicacoes e bibliotecas.

## Packages

| Package | Import | Descricao |
| --- | --- | --- |
| Config | `github.com/raywall/go-core-sdk/config` | Centraliza carregamento de configuracoes compartilhadas, com loaders/resolvers inspirados no AWS SDK. |
| Core | `github.com/raywall/go-core-sdk/core` | Compoe services em um runtime de aplicacao, resolvendo secrets e gerenciando lifecycle de token managers. |
| Handlers | `github.com/raywall/go-core-sdk/handlers` | Define contratos runtime-neutral e adapters para Lambda, HTTP/ECS/EKS/EC2 e worker SQS. |

## AI Packages

| Package | Import | Descricao |
| --- | --- | --- |
| Agent | `github.com/raywall/go-core-sdk/ai/agent` | Cria e testa agentes com APIs OpenAI-compatible, prompt, tools e memoria. |
| MCP Proxy | `github.com/raywall/go-core-sdk/ai/mcp/proxy` | Expoe servicos HTTP existentes como tools MCP-friendly para aceleracao tatica de agentes. |

## Services

| Service | Package | Descricao |
| --- | --- | --- |
| Cache | `github.com/raywall/go-core-sdk/services/cache` | Mantem entidades temporarias em memoria com TTL, consulta, limpeza e expurgo automatico. |
| Consumer REST | `github.com/raywall/go-core-sdk/services/consumer/rest` | Client para chamadas REST com headers, body flexivel e token provider opcional. |
| Consumer AWS | `github.com/raywall/go-core-sdk/services/consumer/aws` | Clients para DynamoDB, S3, Secrets Manager e SQS usando AWS SDK v2. |
| Consumer Hazelcast | `github.com/raywall/go-core-sdk/services/consumer/hazelcast` | Client para carregar config Hazelcast e ler parametros em maps distribuidos. |
| Decision | `github.com/raywall/go-core-sdk/services/decision` | Avalia regras de decisao em CEL expression contra multiplas entidades com cache de compilacao. |
| Environment | `github.com/raywall/go-core-sdk/services/environment` | Facilita leitura de variaveis de ambiente obrigatorias ou com valores padrao. |
| Observability | `github.com/raywall/go-core-sdk/services/observability` | Centraliza logs JSON estruturados e envio simplificado de custom metrics para Datadog. |
| Parser | `github.com/raywall/go-core-sdk/services/parser` | Converte DTOs, entidades, maps e colecoes usando JSON como formato intermediario e tags `json` compativeis. |
| Selector | `github.com/raywall/go-core-sdk/services/selector` | Ordena itens financeiros por atributo e seleciona pagamentos integrais ou parciais com valores em unidade minima. |
| Token | `github.com/raywall/go-core-sdk/services/token` | Gerencia tokens STS com client credentials, renovacao automatica, refresh manual e logs estruturados em JSON. |
| Validation | `github.com/raywall/go-core-sdk/services/validation` | Valida structs e substructs com validator/v10, retornando todos os campos invalidos em um erro tipado. |

## Config e Core

Os packages `config` e `core` ajudam quando uma aplicacao precisa coordenar varios services no mesmo runtime. O `config` carrega valores compartilhados e projeta configuracoes especificas; o `core` monta os services, resolve credenciais em Secrets Manager quando configurado e gerencia o ciclo de vida dos token managers.

```go
package main

import (
	"context"
	"log"

	"github.com/raywall/go-core-sdk/config"
	"github.com/raywall/go-core-sdk/core"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load(ctx,
		config.WithEnv("APP"),
		config.WithAWSDefaultConfig(),
		config.WithServiceName("orders-file-worker"),
		config.WithToken("partner-api", config.TokenConfig{
			BaseURL:  "https://sts.example.com",
			Endpoint: "/oauth/token",
			SecretID: "orders/partner-api",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	runtime, err := core.New(ctx, cfg, core.WithTokenAutoStart(true))
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.Stop()

	restClient := runtime.REST()
	awsClient := runtime.AWS()
	validator := runtime.Validator()
	decision := runtime.Decision()

	_, _, _, _ = restClient, awsClient, validator, decision
}
```

Esse uso e opcional. Cada service continua podendo ser importado e configurado diretamente.

## Samples

Os exemplos em `samples/` sao executaveis com `go run` e tambem possuem testes. Cada sample deixa o `main` como composition root e move o comportamento para um `run` ou use case com dependencias injetadas, facilitando o uso em contextos de clean architecture, ports and adapters e testes unitarios.

O sample composto em `samples/microservice` demonstra um fluxo local de microservico que combina `config`, `core`, Secrets Manager, token management, S3, REST, validation, parser, selector, decision, SQS, logs estruturados e metricas customizadas.

O sample `samples/nanoservice` demonstra um servico menor, com uma unica responsabilidade: consumir eventos SQS em um worker ECS, validar o DTO, montar a entidade interna, consultar uma API REST com token, ordenar parcelas abertas/em atraso com `selector`, permitir pagamento parcial, avaliar regras de negocio e publicar um evento SQS de pagamento.

```sh
go run ./samples/microservice
go run ./samples/nanoservice
go run ./samples/agent
go run ./samples/hazelcast
go test ./samples/...
```

## Agent

O package `ai/agent` facilita prototipar agentes em Go usando APIs OpenAI-compatible. Ele permite configurar o endpoint do modelo, especializar com prompt de sistema, registrar tools com JSON Schema, acoplar memorias de curto e longo prazo e executar inferencias com loop de tool-calling.

```go
client, err := agent.New(agent.Config{
	BaseURL:      "http://localhost:12434/engines/v1",
	Model:        "ai/smollm2",
	SystemPrompt: "You are a concise assistant.",
}, agent.WithShortTermMemory(agent.NewShortTermMemory(12)), agent.WithTools(agent.Tool{
	Name:        "lookup_payment",
	Description: "Looks up payment information.",
	Parameters: json.RawMessage(`{"type":"object"}`),
	Handler: func(ctx context.Context, args json.RawMessage) (any, error) {
		return map[string]any{"status": "ok"}, nil
	},
}))
if err != nil {
	return err
}

result, err := client.Infer(ctx, agent.InferenceInput{Prompt: "Check payment status"})
```

## Handlers

O package `handlers` define um contrato neutro para processors de aplicacao e adapters para diferentes runtimes. Isso permite manter a regra de negocio igual enquanto o deploy muda entre Lambda, HTTP em ECS/EKS/EC2 ou worker SQS.

```go
processor := handlers.ProcessorFunc(func(ctx context.Context, event handlers.Event) (handlers.Result, error) {
	for _, record := range event.Records {
		_ = record
	}
	return handlers.Result{Processed: len(event.Records)}, nil
})

selected, err := runner.NewFromEnv("APP", processor)
if err != nil {
	return err
}
if err := selected.Start(ctx); err != nil {
	return err
}
```

Variaveis comuns:

```sh
APP_HANDLER_RUNTIME=lambda
APP_HANDLER_TRIGGER=s3

APP_HANDLER_RUNTIME=http
APP_HANDLER_TRIGGER=s3
APP_HANDLER_HTTP_ADDRESS=:8080

APP_HANDLER_RUNTIME=sqs-worker
APP_HANDLER_TRIGGER=sqs
APP_HANDLER_SQS_QUEUE_URL=https://sqs.us-east-1.amazonaws.com/123/orders
```

## Observability

O service `observability` facilita logs JSON estruturados e custom metrics para Datadog via DogStatsD. Ao registrar uma metrica, o service aplica o prefixo configurado, combina tags padrao com tags adicionais e adiciona sempre a tag `env:<environment>`.

```go
package main

import (
	"context"
	"log"

	"github.com/raywall/go-core-sdk/services/observability"
)

func main() {
	ctx := context.Background()
	telemetry, err := observability.New(observability.Config{
		ServiceName:    "orders-worker",
		Environment:    "prod",
		Version:        "1.0.0",
		MetricPrefix:   "orders",
		DatadogAddress: "127.0.0.1:8125",
		DefaultTags:    []string{"team:platform"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer telemetry.Close()

	telemetry.Logger().InfoContext(ctx, "file_received", "bucket", "orders-files")
	if err := telemetry.Increment(ctx, "events.received", "source:s3"); err != nil {
		log.Fatal(err)
	}
}
```

## MCP Proxy

O package `ai/mcp/proxy` permite mapear endpoints HTTP ja existentes, como API Gateway, Lambda URL, ECS ou servicos atras de Load Balancer, para contratos de tools que podem ser expostos por um MCP server. Ele e uma solucao tatica para acelerar agentes enquanto uma integracao MCP definitiva e desenhada.

```go
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/raywall/go-core-sdk/ai/mcp/proxy"
	proxytypes "github.com/raywall/go-core-sdk/ai/mcp/proxy/types"
)

func main() {
	ctx := context.Background()
	mcpProxy, err := proxy.New(proxy.Config{
		BaseURL: "https://api.example.com",
		Tools: []proxytypes.Tool{
			{
				Name:        "simulate_payment",
				Description: "Simulate a payment before creating the final event.",
				Method:      http.MethodPost,
				Path:        "/payments/simulate",
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	output, err := mcpProxy.Invoke(ctx, proxytypes.InvokeInput{
		ToolName:  "simulate_payment",
		Arguments: map[string]any{"amount": 120000},
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = output
}
```

## Environment

O service `environment` simplifica a leitura de variaveis de ambiente obrigatorias ou opcionais com default. Variaveis existentes com valor vazio sao preservadas como existentes.

```go
package main

import (
	"context"
	"log"

	"github.com/raywall/go-core-sdk/services/environment"
)

func main() {
	ctx := context.Background()

	serviceName, err := environment.Get(ctx, "APP_SERVICE_NAME")
	if err != nil {
		log.Fatal(err)
	}
	environmentName, err := environment.GetDefault(ctx, "APP_ENVIRONMENT", "local")
	if err != nil {
		log.Fatal(err)
	}

	_, _ = serviceName, environmentName
}
```

## Parser

O service `parser` facilita a conversao de um DTO ou entidade em outra estrutura quando ambos compartilham tags `json` compativeis. Internamente ele serializa a origem para JSON e decodifica esse JSON no destino.

```go
package main

import (
	"context"
	"log"

	"github.com/raywall/go-core-sdk/services/parser"
)

type ProposalDTO struct {
	ID     string `json:"id"`
	Amount int64  `json:"amount"`
}

type Proposal struct {
	ID     string `json:"id"`
	Amount int64  `json:"amount"`
}

func main() {
	proposal, err := parser.ParseAs[Proposal](context.Background(), ProposalDTO{
		ID:     "proposal-123",
		Amount: 75000,
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = proposal
}
```

## Consumer

O service `consumer` e dividido em packages explicitos: `consumer/rest` para chamadas HTTP, `consumer/aws` para DynamoDB, S3, Secrets Manager e SQS, e `consumer/hazelcast` para parametros em maps Hazelcast. Nao ha fachada no package raiz; aplicacoes que usam varios adapters devem instanciar os clients necessarios ou usar os clients expostos por `core` quando aplicavel.

```go
package main

import (
	"context"
	"log"
	"net/http"

	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
	consumerrest "github.com/raywall/go-core-sdk/services/consumer/rest"
	"github.com/raywall/go-core-sdk/services/token"
)

type DatabaseSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	ctx := context.Background()

	manager, err := token.NewManager(token.Config{
		BaseURL:      "https://sts.example.com",
		Endpoint:     "/oauth/token",
		ClientID:     "uuid",
		ClientSecret: "secret",
		ValidateSSL:  true,
	})
	if err != nil {
		log.Fatal(err)
	}

	restClient, err := consumerrest.New(consumerrest.Config{}, consumerrest.WithTokenProvider(manager))
	if err != nil {
		log.Fatal(err)
	}
	awsClient, err := consumeraws.New(consumeraws.Config{Region: "us-east-1"})
	if err != nil {
		log.Fatal(err)
	}

	response, err := restClient.REST(http.MethodPost, "https://api.example.com/orders").
		WithHeader("X-App", "orders-api").
		WithBody(map[string]any{"customerId": "123"}).
		WithToken().
		Do(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = awsClient.PutDynamoDB(ctx, consumeraws.DynamoDBPutInput{
		TableName: "orders",
		Item:      map[string]any{"PK": "ORDER#1", "status": "CREATED"},
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = awsClient.PutS3(ctx, consumeraws.S3PutInput{
		Bucket:      "orders-files",
		Key:         "ORDER#1.json",
		Body:        response.Body,
		ContentType: "application/json",
	})
	if err != nil {
		log.Fatal(err)
	}

	var database DatabaseSecret
	_, err = awsClient.GetSecretJSON(ctx, consumeraws.SecretGetInput{
		SecretID: "orders/database",
	}, &database)
	if err != nil {
		log.Fatal(err)
	}

	_, err = awsClient.SendSQS(ctx, consumeraws.SQSSendInput{
		QueueURL: "https://sqs.us-east-1.amazonaws.com/123/orders",
		Body:     string(response.Body),
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

Uso do consumer Hazelcast:

```go
package main

import (
	"context"
	"log"

	consumerhazelcast "github.com/raywall/go-core-sdk/services/consumer/hazelcast"
)

func main() {
	ctx := context.Background()
	client, err := consumerhazelcast.New(ctx, consumerhazelcast.Config{
		Source: consumerhazelcast.Source{
			Kind: consumerhazelcast.SourceS3,
			Bucket: "app-configs",
			Key: "hazelcast/client.json",
		},
		DefaultMap: "runtime-parameters",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close(ctx)

	enabled, found, err := client.GetBool(ctx, "", "payment.enabled")
	if err != nil {
		log.Fatal(err)
	}
	_, _ = enabled, found
}
```

## Cache

O service `cache` cria um cache temporario em memoria para armazenar entidades durante o ciclo de vida do runtime. Ele e util para runtimes reaproveitados, como Lambda warm starts, reduzindo chamadas repetidas a APIs ou bases externas.

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/raywall/go-core-sdk/services/cache"
)

type Customer struct {
	ID   string
	Name string
}

func main() {
	store, err := cache.New[Customer](cache.Config{
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: time.Minute,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := store.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer store.Stop()

	if err := store.Add(ctx, "customer-123", Customer{ID: "customer-123", Name: "Ana"}); err != nil {
		log.Fatal(err)
	}

	customer, found, err := store.Get(ctx, "customer-123")
	if err != nil {
		log.Fatal(err)
	}

	_, _ = customer, found
}
```

## Decision

O service `decision` avalia regras em CEL expression contra varias entidades nomeadas. As entidades podem ser structs ou maps; campos de structs usam a tag `json` quando existir.

```go
package main

import (
	"context"
	"log"

	"github.com/raywall/go-core-sdk/services/decision"
	decisiontypes "github.com/raywall/go-core-sdk/services/decision/types"
)

type Worker struct {
	Active          bool  `json:"active"`
	AvailableMargin int64 `json:"availableMargin"`
}

type Proposal struct {
	Amount int64 `json:"amount"`
}

func main() {
	engine, err := decision.New()
	if err != nil {
		log.Fatal(err)
	}

	result, err := engine.Evaluate(context.Background(), decisiontypes.EvaluationInput{
		Rule: decisiontypes.Rule{
			Name:       "margin-approved",
			Expression: "worker.active && proposal.amount <= worker.availableMargin",
		},
		Entities: map[string]any{
			"worker":   Worker{Active: true, AvailableMargin: 100000},
			"proposal": Proposal{Amount: 75000},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = result.Allowed
}
```

## Selector

O service `selector` ordena itens por um atributo configurado e aplica um valor disponivel sobre essa lista ordenada. Valores financeiros sao retornados como `int64` na unidade minima do dominio, por exemplo centavos. O valor disponivel e os valores dos itens tambem podem chegar como decimal/string/float quando `DecimalScale` for informado, e datas string no formato `YYYY-MM-DD` sao aceitas por `KindTime` sem layout customizado.

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/raywall/go-core-sdk/services/selector"
	selectortypes "github.com/raywall/go-core-sdk/services/selector/types"
)

type Installment struct {
	Number      int       `json:"number"`
	Status      string    `json:"status"`
	DueDate     time.Time `json:"dueDate"`
	AmountCents int64     `json:"amountCents"`
}

func main() {
	installments := []Installment{
		{Number: 2, Status: "OPEN", DueDate: mustDate("2026-02-01"), AmountCents: 10000},
		{Number: 1, Status: "OPEN", DueDate: mustDate("2026-01-01"), AmountCents: 10000},
	}

	ordered, result, err := selector.SortAndSelect(context.Background(), installments,
		selectortypes.SortConfig{
			Path:      "dueDate",
			Kind:      selectortypes.KindTime,
			Direction: selectortypes.Ascending,
		},
		selectortypes.SelectionConfig{
			AmountPath:      "amountCents",
			AvailableAmount: 15000,
			Mode:            selectortypes.ModePartial,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	_, _ = ordered, result
}

func mustDate(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
```

## Token

O service `token` inicializa um gerenciador de token STS no inicio da aplicacao, mantem o token renovado durante o ciclo de vida do processo e permite encerrar a rotina de renovacao no shutdown.

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/raywall/go-core-sdk/services/token"
)

func main() {
	manager, err := token.NewManager(token.Config{
		BaseURL:        "https://sts.example.com",
		Endpoint:       "/oauth/token",
		ClientID:       "uuid",
		ClientSecret:   "uuid",
		Headers:        map[string]string{"X-App": "orders-api"},
		ValidateSSL:    true,
		RefreshBefore:  30 * time.Second,
		RequestTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := manager.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer manager.Stop()

	authorization := manager.Token().ToString()
	_ = authorization
}
```

O token retornado por `Manager.Token()` e um ponteiro estavel. O gerenciador atualiza o mesmo objeto internamente, permitindo que componentes que mantenham a referencia observem os novos valores por meio dos metodos de leitura de `types.Token`.

## Validation

O service `validation` simplifica a validacao de structs, substructs e colecoes de substructs usando `github.com/go-playground/validator/v10`. Quando existem falhas, o erro retornado carrega todos os campos que precisam ser ajustados.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/raywall/go-core-sdk/services/validation"
	validationtypes "github.com/raywall/go-core-sdk/services/validation/types"
)

type Order struct {
	Customer Customer `json:"customer"`
	Items    []Item   `json:"items" validate:"required,min=1"`
}

type Customer struct {
	Document string `json:"document" validate:"required,len=11"`
}

type Item struct {
	SKU      string `json:"sku" validate:"required"`
	Quantity int    `json:"quantity" validate:"min=1"`
}

func main() {
	validator, err := validation.New()
	if err != nil {
		log.Fatal(err)
	}

	order := Order{
		Customer: Customer{Document: "123"},
		Items:    []Item{{SKU: "", Quantity: 0}},
	}

	if err := validator.Validate(context.Background(), order); err != nil {
		var validationErr *validationtypes.ValidationError
		if errors.As(err, &validationErr) {
			for _, field := range validationErr.Fields {
				fmt.Printf("%s: %s\n", field.Namespace, field.Message)
			}
		}
	}
}
```
