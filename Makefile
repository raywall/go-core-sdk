.PHONY: test

test:
	@echo "Iniciando testes da biblioteca go-core-sdk ..."; \
	 go test -v ./...;

cache:
	@echo "Executando exemplo de uso do services/cache ..."; \
	 go run samples/cache/main.go;

consumer:
	@echo "Executando exemplo de uso do services/consumer ..."; \
	 go run samples/consumer/main.go;

decision:
	@echo "Executando exemplo de uso do services/decision ..."; \
	 go run samples/decision/main.go;

selector:
	@echo "Executando exemplo de uso do services/selector ..."; \
	 go run samples/selector/main.go;

token:
	@echo "Executando exemplo de uso do services/token ..."; \
	 go run samples/token/main.go;

validation:
	@echo "Executando exemplo de uso do services/validation ..."; \
	 go run samples/validation/main.go;