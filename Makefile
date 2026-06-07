.PHONY: build clean test run fmt

build:
	go build -o arahin .

clean:
	rm -f arahin

test:
	go test ./...

run: build
	./arahin chat

fmt:
	go fmt ./...
