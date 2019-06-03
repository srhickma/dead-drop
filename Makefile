all: test build

build:
	cd client; go build -o ../bin/dead -v
	cd server; go build -o ../bin/deadd -v

test:
	go test -v ./...

clean:
	go clean
	rm -r -f bin
