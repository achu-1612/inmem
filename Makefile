generate:
	go generate -x -run="mockgen" ./...

test:
	go test -v -cover ./...
lint:
	which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0
	golangci-lint run --exclude-dirs  example --allow-parallel-runners ./...

gosec:
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@gosec -fmt=json -out="/tmp/gosec.json" -exclude-generated -no-fail ./...
	@cat /tmp/gosec.json

	# if [ $(shell cat /tmp/gosec.json | jq -r '.Stats.found') -gt 0 ]; then \
	# 	exit 1; \
	# fi

spell-check:
	go install github.com/client9/misspell/cmd/misspell@latest
	find . -type f -name \*.go -ls | xargs misspell -source text -error
