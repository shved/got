build:
	go build -o got
lint:
	go fmt ./...
test:
	go test ./...
clean:
	rm -rf test/dummy_app/.got
	
