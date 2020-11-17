.PHONY: clean test build zip

default: clean test build zip

clean:
	rm -rf bin/*

test:
	go test -v -coverprofile=cover.out -json > tests.out ./...
	awk 'BEGIN{IGNORECASE=1} /\"coverage/{print $0}' tests.out

build:
	GOOS=linux GOARCH=amd64 go build -v -o ./bin/ ./cmd/...
	
zip:
	@cd ./bin && find . -type f -exec zip -D '{}.zip' '{}' \;
