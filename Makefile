.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -v ./... -race -cover -timeout=10s -count=1