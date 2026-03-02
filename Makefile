SHELL := /bin/bash

BIN := bin/bookinfo
TIMEOUT_SEC ?= 120

.PHONY: help build test run run-text run-full run-isbn clean

help:
	@echo "사용법:"
	@echo "  make build                  # 바이너리 빌드"
	@echo "  make test                   # 전체 테스트 실행"
	@echo "  make run TITLE='Clean Code' [ARGS='--full']"
	@echo "  make run-text TITLE='Clean Code'"
	@echo "  make run-full TITLE='Clean Code'"
	@echo "  make run-isbn ISBN='978-0132350884' [ARGS='--format text']"
	@echo "  (선택) TIMEOUT_SEC=180 make run-text TITLE='Clean Code'"

build:
	@mkdir -p bin
	go build -o $(BIN) ./cmd/bookinfo

test:
	go test ./...

run: build
	@if [[ -z "$(TITLE)" ]]; then \
		echo "ERROR: TITLE 변수를 지정하세요. 예) make run TITLE='Clean Code'"; \
		exit 1; \
	fi
	@set -a; \
	[[ -f .env ]] && source .env || true; \
	set +a; \
	BOOKINFO_HTTP_TIMEOUT_SEC=$(TIMEOUT_SEC) $(BIN) "$(TITLE)" $(ARGS)

run-text: build
	@if [[ -z "$(TITLE)" ]]; then \
		echo "ERROR: TITLE 변수를 지정하세요. 예) make run-text TITLE='Clean Code'"; \
		exit 1; \
	fi
	@set -a; \
	[[ -f .env ]] && source .env || true; \
	set +a; \
	BOOKINFO_HTTP_TIMEOUT_SEC=$(TIMEOUT_SEC) $(BIN) "$(TITLE)" --format text $(ARGS)

run-full: build
	@if [[ -z "$(TITLE)" ]]; then \
		echo "ERROR: TITLE 변수를 지정하세요. 예) make run-full TITLE='Clean Code'"; \
		exit 1; \
	fi
	@set -a; \
	[[ -f .env ]] && source .env || true; \
	set +a; \
	BOOKINFO_HTTP_TIMEOUT_SEC=$(TIMEOUT_SEC) $(BIN) "$(TITLE)" --full $(ARGS)

run-isbn: build
	@if [[ -z "$(ISBN)" ]]; then \
		echo "ERROR: ISBN 변수를 지정하세요. 예) make run-isbn ISBN='978-0132350884'"; \
		exit 1; \
	fi
	@set -a; \
	[[ -f .env ]] && source .env || true; \
	set +a; \
	BOOKINFO_HTTP_TIMEOUT_SEC=$(TIMEOUT_SEC) $(BIN) --isbn "$(ISBN)" $(ARGS)

clean:
	rm -f $(BIN)
