default: install

install:
	GOOS=$(GOOS) GOARCH=amd64 go build -mod=vendor -o ./bin/logScanner .

clean:
	GOOS=$(GOOS) GOARCH=amd64 go clean