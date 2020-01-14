build:
	go build -o build/got
lint:
	go fmt ./...
test:
	go test main_test.go
clean:
	rm -rf test/dummy_app/*

.PHONY: build lint test clean

