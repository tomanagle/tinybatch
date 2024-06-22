.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -v ./... -race -cover  -coverprofile cover.out -timeout=10s -count=1

.PHONY: test-coverage
test-coverage:
	go tool cover -html=cover.out