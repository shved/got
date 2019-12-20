build:
	go build -o got
lint:
	go fmt .
clean:
	rm -rf test/dummy_app/.got
	
