.PHONY: build run test clean

build:
	go build -o main .

run: build
	./main

test:
	go test ./...

clean:
	rm -f main
