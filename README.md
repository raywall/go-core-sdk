# go-core-sdk

`go-core-sdk` reune packages e services Go autocontidos para uso em aplicacoes e bibliotecas.

## Packages

| Package | Import | Descricao |
| --- | --- | --- |
| Config | `github.com/raywall/go-core-sdk/config` | Centraliza carregamento de configuracoes compartilhadas, com loaders/resolvers inspirados no AWS SDK. |
| Core | `github.com/raywall/go-core-sdk/core` | Compoe services em um runtime de aplicacao, resolvendo secrets e gerenciando lifecycle de token managers. |

## Services

| Service | Package | Descricao |
| --- | --- | --- |
| Cache | `github.com/raywall/go-core-sdk/services/cache` | Mantem entidades temporarias em memoria com TTL, consulta, limpeza e expurgo automatico. |
| Consumer | `github.com/raywall/go-core-sdk/services/consumer` | Simplifica chamadas REST com token opcional e operacoes comuns de DynamoDB, S3, Secrets Manager e SQS. |
| Decision | `github.com/raywall/go-core-sdk/services/decision` | Avalia regras de decisao em CEL expression contra multiplas entidades com cache de compilacao. |
| Environment | `github.com/raywall/go-core-sdk/services/environment` | Facilita leitura de variaveis de ambiente obrigatorias ou com valores padrao. |
| MCP Proxy | `github.com/raywall/go-core-sdk/services/mcp/proxy` | Expoe servicos HTTP existentes como tools MCP-friendly para aceleracao tatica de agentes. |
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

	consumer := runtime.Consumer()
	validator := runtime.Validator()
	decision := runtime.Decision()

	_, _, _ = consumer, validator, decision
}
```

Esse uso e opcional. Cada service continua podendo ser importado e configurado diretamente.

## Samples

Os exemplos em `samples/` sao executaveis com `go run` e tambem possuem testes. Cada sample deixa o `main` como composition root e move o comportamento para um `run` ou use case com dependencias injetadas, facilitando o uso em contextos de clean architecture, ports and adapters e testes unitarios.

O sample composto em `samples/microservice` demonstra um fluxo local de microservico que combina `config`, `core`, Secrets Manager, token management, S3, REST, validation, parser, selector, decision, SQS, logs estruturados e metricas customizadas.

```sh
go run ./samples/microservice
go test ./samples/...
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

O service `mcp/proxy` permite mapear endpoints HTTP ja existentes, como API Gateway, Lambda URL, ECS ou servicos atras de Load Balancer, para contratos de tools que podem ser expostos por um MCP server. Ele e uma solucao tatica para acelerar agentes enquanto uma integracao MCP definitiva e desenhada.

```go
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/raywall/go-core-sdk/services/mcp/proxy"
	proxytypes "github.com/raywall/go-core-sdk/services/mcp/proxy/types"
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

O service `consumer` centraliza integracoes comuns de microservicos: chamadas REST com headers e body flexiveis, injecao opcional de Authorization a partir do `services/token`, e operacoes simples em DynamoDB, S3, Secrets Manager e SQS usando AWS SDK v2.

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/raywall/go-core-sdk/services/consumer"
	consumertypes "github.com/raywall/go-core-sdk/services/consumer/types"
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

	client, err := consumer.New(consumer.Config{AWSRegion: "us-east-1"},
		consumer.WithTokenProvider(manager),
	)
	if err != nil {
		log.Fatal(err)
	}

	response, err := client.REST(http.MethodPost, "https://api.example.com/orders").
		WithHeader("X-App", "orders-api").
		WithBody(map[string]any{"customerId": "123"}).
		WithToken().
		Do(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = client.PutDynamoDB(ctx, consumertypes.DynamoDBPutInput{
		TableName: "orders",
		Item:      map[string]any{"PK": "ORDER#1", "status": "CREATED"},
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.PutS3(ctx, consumertypes.S3PutInput{
		Bucket:      "orders-files",
		Key:         "ORDER#1.json",
		Body:        response.Body,
		ContentType: "application/json",
	})
	if err != nil {
		log.Fatal(err)
	}

	var database DatabaseSecret
	_, err = client.GetSecretJSON(ctx, consumertypes.SecretGetInput{
		SecretID: "orders/database",
	}, &database)
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.SendSQS(ctx, consumertypes.SQSSendInput{
		QueueURL: "https://sqs.us-east-1.amazonaws.com/123/orders",
		Body:     string(response.Body),
	})
	if err != nil {
		log.Fatal(err)
	}
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

O service `selector` ordena itens por um atributo configurado e aplica um valor disponivel sobre essa lista ordenada. Valores financeiros usam `int64` na unidade minima do dominio, por exemplo centavos.

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
