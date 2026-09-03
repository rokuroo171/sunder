.PHONY: overseer run vet test wraith fmt clean

VERSION ?= 0.1.1

overseer:
	cd overseer && go build -ldflags "-X main.version=$(VERSION)" -o bin/overseer ./cmd/overseer

run:
	cd overseer && go run ./cmd/overseer -listen :8443

vet:
	cd overseer && go vet ./...

test:
	cd overseer && go test ./...

wraith:
	cd wraith && cargo build --release

fmt:
	cd overseer && gofmt -w .
	cd wraith && cargo fmt

clean:
	rm -rf overseer/bin wraith/target
