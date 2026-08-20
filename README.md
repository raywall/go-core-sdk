# go-core-sdk

`go-core-sdk` reune services Go autocontidos para uso em aplicacoes e bibliotecas.

## Services

| Service | Package | Descricao |
| --- | --- | --- |
| Token | `github.com/raywall/go-core-sdk/services/token` | Gerencia tokens STS com client credentials, renovacao automatica, refresh manual e logs estruturados em JSON. |
| Validation | `github.com/raywall/go-core-sdk/services/validation` | Valida structs e substructs com validator/v10, retornando todos os campos invalidos em um erro tipado. |

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
