build:
	go build -o got
run:
	go run -race got.go
lint:
	go fmt .
clean:
	rm -rf {.,}got
	
