.PHONY: test

build:
	go build -o got
lint:
	go fmt ./...
test:
	go test main_test.go
clean:
	rm -rf test/dummy_app/*
	
