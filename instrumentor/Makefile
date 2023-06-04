.PHONY: run-checks
run-go-checks:
	make tidy
	make fmt

.PHONY: tidy
tidy:
	rm -fr go.sum
	go mod tidy -compat=1.19

.PHONY: fmt
fmt:
	gofmt  -w -s ./
	goimports -w ./
