.PHONY: test cache consumer core decision microservice observability selector token validation

test:
	@echo "Iniciando testes da biblioteca go-core-sdk ..."; \
	 go test -v ./...;

cache:
	@echo "Executando exemplo de uso do services/cache ..."; \
	 go run samples/cache/main.go;

consumer:
	@echo "Executando exemplo de uso do services/consumer ..."; \
	 go run samples/consumer/main.go;

core:
	@echo "Executando exemplo de uso do core ..."; \
	 go run samples/core/main.go;

decision:
	@echo "Executando exemplo de uso do services/decision ..."; \
	 go run samples/decision/main.go;

env:
	@echo "Executando exemplo de uso do services/environment ..."; \
	 go run samples/environment/main.go;

mcp-proxy:
	@echo "Executando exemplo de uso do services/mcp/proxy ..."; \
	 go run samples/mcp_proxy/main.go;

microservice:
	@echo "Executando exemplo de uso do services/microservice ..."; \
	 go run samples/microservice/main.go;

observability:
	@echo "Executando exemplo de uso do services/observability ..."; \
	 go run samples/observability/main.go;

parser:
	@echo "Executando exemplo de uso do services/parser ..."; \
	 go run samples/parser/main.go;

selector:
	@echo "Executando exemplo de uso do services/selector ..."; \
	 go run samples/selector/main.go;

token:
	@echo "Executando exemplo de uso do services/token ..."; \
	 go run samples/token/main.go;

validation:
	@echo "Executando exemplo de uso do services/validation ..."; \
	 go run samples/validation/main.go;