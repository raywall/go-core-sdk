# sts-token-management

Biblioteca Go para gestão de tokens STS (Security Token Service), construída
com Clean Architecture. Ela mantém um token de acesso sempre válido em
segundo plano e oferece uma função facilitadora (`RestAuthCaller`) para
chamar APIs externas com o bearer token injetado automaticamente, com
suporte a retries, delay entre tentativas e timeout por requisição.

## 1. Arquitetura

```
app/
  domain/         # Entidades puras e interfaces de repositório (portas). Zero dependências externas.
    entity/         TokenManagement, APIRequest, APIResponse, HTTPMethod
    repository/     TokenProvider, TokenManager, RestAuthCaller (interfaces)
    errors/         Erros de domínio tipados (pacote "errs")
  application/    # Casos de uso e orquestração. Depende só de domain.
    dto/            DTOs de entrada/saída dos casos de uso
    service/        TokenManagerService — renovação em background (goroutine + ticker)
    usecase/        GetCurrentTokenUseCase, CallAPIUseCase (+ testes com mocks)
  infrastructure/ # Implementações concretas dos ports do domínio.
    sts/            HTTPTokenProvider — busca token real num endpoint STS
    httpclient/     RestAuthCallerImpl — chamadas HTTP com bearer, retries e timeout
  presentation/   # Controllers HTTP da aplicação de teste (demo).
    http/           TokenHandler, ProxyHandler, middleware de logging, mock de STS
  main/           # Composition root: configuração e injeção de dependências.
    config/         Carrega configuração do ambiente
    factory/        Monta o grafo de dependências (Build)
cmd/
  main.go         # Aplicação de teste executável (ver seção 3)
```

Regra de dependência (Clean Architecture): as camadas internas nunca
importam as externas. `domain` não conhece `application`; `application` não
conhece `infrastructure`/`presentation`; toda a montagem acontece apenas em
`app/main/factory` e `cmd/main.go`.

### Entidades core

- **`TokenManagement`** (`app/domain/entity/token.go`): representa o token
  STS e sua validade (`IsExpired`, `IsNearExpiration`, `BearerHeader`).
- **`RestAuthCaller`** (`app/domain/repository/rest_auth_caller.go` +
  `app/domain/entity/rest_auth_caller.go`): porta/contrato para chamadas de
  API autenticadas, com `APIRequest`/`APIResponse` como objetos de valor.

## 2. Regras de negócio implementadas

- O gerenciador de token (`TokenManagerService`) roda em uma única goroutine
  de segundo plano, iniciada por `Start(ctx)` e encerrada de forma
  determinística por `Stop()` ou pelo cancelamento do `context.Context` —
  sem impactar a aplicação hospedeira e sem vazar goroutines.
- O token é mantido sempre atualizado: a cada tick, o gerenciador verifica
  se o token está próximo de expirar (`IsNearExpiration`, configurável via
  `STS_REFRESH_THRESHOLD`) e, se estiver, busca um novo antes que o atual
  expire, substituindo-o de forma thread-safe (`sync.RWMutex`).
- `RestAuthCallerImpl.Call` injeta automaticamente o header
  `Authorization: Bearer <token>` em toda chamada, de forma transparente ao
  chamador.
- `CallAPIUseCase`/`APIRequest` permitem customizar `method`, `url`,
  `headers` e `body` livremente, para qualquer verbo HTTP suportado (GET,
  POST, PUT, PATCH, DELETE).
- É possível definir `retries` (número de retentativas após a primeira
  tentativa) e `retryDelayMs` (intervalo entre tentativas). Respostas 5xx ou
  falhas de transporte disparam retry; respostas 4xx não são retentadas
  (erro do cliente, retry não ajudaria).
- É possível definir `timeoutMs`, aplicado por tentativa via
  `context.WithTimeout`, interrompendo a chamada ao atingir o limite.

## 3. Rodando o projeto localmente

Pré-requisitos: Go 1.25+.

```bash
make test-race   # roda a suíte de testes com detector de race
make run         # sobe a aplicação de teste em :8080
```

`make run` inicia uma aplicação **totalmente self-contained** — sem
dependência de rede externa:

1. Um mock de STS local (`/mock/sts/token`) simula o provedor externo,
   emitindo tokens de curta duração (20s) para tornar a renovação visível
   nos logs.
2. O `factory.Build` conecta o `TokenManager` real a esse mock e inicia a
   renovação em background.
3. Um servidor HTTP de demonstração expõe:

   | Rota           | Método | Descrição                                                                 |
   |----------------|--------|----------------------------------------------------------------------------|
   | `/health`      | GET    | Readiness probe                                                            |
   | `/token`       | GET    | Token atualmente em cache no gerenciador                                  |
   | `/echo`        | POST   | Ecoa método/headers/body recebidos (simula uma API downstream)            |
   | `/call`        | POST   | Executa `CallAPIUseCase` contra a URL informada, com bearer/retries/timeout|

Exemplo de chamada usando `/call` para atingir o próprio `/echo` e comprovar
que o bearer token foi injetado automaticamente:

```bash
curl -s -X POST http://localhost:8080/call -d '{
  "method": "POST",
  "url": "http://localhost:8080/echo",
  "headers": {"X-Custom": "hello"},
  "body": "{\"ping\":true}",
  "retries": 2,
  "retryDelayMs": 100,
  "timeoutMs": 3000
}'
```

Para apontar para um STS real em vez do mock, defina as variáveis de
ambiente abaixo antes de `make run`.

### Variáveis de ambiente

| Variável                 | Padrão (demo)                          | Descrição                                   |
|---------------------------|-----------------------------------------|----------------------------------------------|
| `SERVER_ADDR`              | `:8080`                                 | Endereço do servidor de demonstração         |
| `STS_TOKEN_URL`             | mock local (`/mock/sts/token`)          | Endpoint STS (client-credentials)            |
| `STS_CLIENT_ID`             | `demo-client`                           | Client ID usado na busca do token            |
| `STS_CLIENT_SECRET`         | `demo-secret`                           | Client secret usado na busca do token        |
| `STS_SCOPE`                 | *(vazio)*                               | Scope opcional                               |
| `STS_REFRESH_THRESHOLD`     | `8` (segundos)                          | Renovar o token quando faltar esse tempo     |
| `STS_POLL_INTERVAL`         | `2` (segundos)                          | Intervalo de checagem do gerenciador         |

## 4. Usando como biblioteca em outra aplicação

O caso de uso típico é importar `app/main/factory` (ou reproduzir sua
montagem) dentro do seu próprio serviço:

```go
cfg := config.Config{
    STSTokenURL:      "https://sts.minha-empresa.com/oauth/token",
    STSClientID:       os.Getenv("STS_CLIENT_ID"),
    STSClientSecret:   os.Getenv("STS_CLIENT_SECRET"),
    RefreshThreshold:  60 * time.Second,
    PollInterval:      15 * time.Second,
}

app, err := factory.Build(ctx, cfg, logger)
if err != nil {
    log.Fatal(err)
}
defer app.TokenManager.Stop()

out, err := app.CallAPIUseCase.Execute(ctx, dto.CallAPIInput{
    Method:       "GET",
    URL:          "https://api.parceiro.com/recurso",
    Retries:      3,
    RetryDelayMs: 200,
    TimeoutMs:    5000,
})
```

`app.TokenManager` continua rodando em segundo plano por toda a vida do
processo hospedeiro; chame `Stop()` durante o shutdown gracioso da
aplicação.

## 5. Testes

Casos de uso em `app/application/usecase` têm testes unitários com mocks
das interfaces de domínio (`repository.RestAuthCaller`, `repository.TokenManager`).
O serviço de renovação em background (`app/application/service`) também tem
testes cobrindo início, renovação automática, encerramento sem vazamento de
goroutine e falha na busca inicial.

```bash
make test        # go test ./...
make test-race   # go test -race ./...  (recomendado)
```
# sts-token-management
