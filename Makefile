.PHONY: clean vet test build zip

default: clean vet build zip

clean:
	rm -rf bin/*

vet:
	go vet -v ./...

test:
	go test -v -coverprofile=cover.out -json > tests.out ./...
	awk '/\"coverage/{print $0}' tests.out

build:
	GOOS=linux GOARCH=amd64 go build -v -o ./bin/ ./cmd/...
	
zip:
	@cd ./bin && find . -type f -exec zip -D '{}.zip' '{}' \;
