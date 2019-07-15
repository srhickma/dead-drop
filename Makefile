all: test build

build:
	cd client; \
        go build -o ../bin/dead -v
	cd server; \
	    go-bindata -o generated.go -ignore=\\.gitignore data/...; \
	    go build -o ../bin/deadd -v; \
	    rm generated.go

test:
	go test -v ./...

clean:
	go clean
	rm -r -f bin
