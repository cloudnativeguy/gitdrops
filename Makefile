.PHONY: test build run

test:
	go test ./... -v *_test.go

build:
	go build -o _output/gitdrops
	rm -rf _output

run:
	go run main.go
