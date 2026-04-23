BINARY := k8s-mcp-server

.PHONY: build run-stdio run-sse clean

build:
	go build -o $(BINARY) .

run-stdio: build
	./$(BINARY)

run-sse: build
	./$(BINARY) --port 8080

clean:
	rm -f $(BINARY)
