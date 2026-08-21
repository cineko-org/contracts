.PHONY: check generate lint test

check:
	$(MAKE) lint
	$(MAKE) generate
	git diff --exit-code -- gen
	$(MAKE) test

generate:
	buf generate
	bash scripts/normalize-generated.sh

lint:
	buf format --diff --exit-code
	buf lint
	bash scripts/check-contract-rules.sh

test:
	go test -race ./...
	npm test
