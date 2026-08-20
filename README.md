# go-core-sdk

`go-core-sdk` reune services Go autocontidos para uso em aplicacoes e bibliotecas.

## Services

| Service | Package | Descricao |
| --- | --- | --- |
| Decision | `github.com/raywall/go-core-sdk/services/decision` | Avalia regras de decisao em CEL expression contra multiplas entidades com cache de compilacao. |
| Selector | `github.com/raywall/go-core-sdk/services/selector` | Ordena itens financeiros por atributo e seleciona pagamentos integrais ou parciais com valores em unidade minima. |
| Token | `github.com/raywall/go-core-sdk/services/token` | Gerencia tokens STS com client credentials, renovacao automatica, refresh manual e logs estruturados em JSON. |
| Validation | `github.com/raywall/go-core-sdk/services/validation` | Valida structs e substructs com validator/v10, retornando todos os campos invalidos em um erro tipado. |

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
