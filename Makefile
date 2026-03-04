SHELL := /bin/bash

BIN := bin/bookinfo
TIMEOUT_SEC ?= 120

# ── 옵션 변수 ────────────────────────────────────────
# TITLE        책 제목              (필수 for run 계열)
# ISBN         ISBN-13             (필수 for isbn 계열)
# L            언어 ko|en|ja|zh-tw  (선택)  ※ LANG은 시스템 로케일과 충돌하므로 L 사용
# AUTHOR       저자                 (선택)
# MODEL        모델 ID             (선택, 기본 qwen3.5-flash)
# OUTPUT       출력 경로            (선택)
# FORMAT       json|text           (선택, 기본 json)
# BATCH        배치 파일 경로       (필수 for batch)
# NOCACHE      1 로 설정시 --no-cache (선택)
# FULL         1 로 설정시 --full   (선택)
# ARGS         추가 플래그          (선택)

# 공통 플래그 자동 조합
_FLAGS :=
ifdef L
_FLAGS += --lang $(L)
endif
ifdef AUTHOR
_FLAGS += --author "$(AUTHOR)"
endif
ifdef MODEL
_FLAGS += --model $(MODEL)
endif
ifdef OUTPUT
_FLAGS += --output "$(OUTPUT)"
endif
ifdef FORMAT
_FLAGS += --format $(FORMAT)
endif
ifeq ($(NOCACHE),1)
_FLAGS += --no-cache
endif
ifeq ($(FULL),1)
_FLAGS += --full
endif

# .env 로딩 + 실행을 한 쉘에서 처리하는 매크로
# $(1) = 바이너리 인자 전체
define exec
	@set -a; [[ -f .env ]] && source .env || true; set +a; \
	BOOKINFO_HTTP_TIMEOUT_SEC=$(TIMEOUT_SEC) $(1)
endef

.PHONY: help build test clean \
	run run-en run-kr run-full run-full-en run-full-kr run-text \
	run-isbn run-isbn-en run-batch \
	run-nocache run-full-nocache \
	compare compare-dashscope compare-cli

# ── 도움말 ───────────────────────────────────────────
help:
	@echo "bookinfo CLI - Makefile 사용법"
	@echo ""
	@echo "기본:"
	@echo "  make build                          바이너리 빌드"
	@echo "  make test                           전체 테스트"
	@echo "  make clean                          바이너리 삭제"
	@echo ""
	@echo "실행 (TITLE 필수):"
	@echo "  make run TITLE='Clean Code'                      기본 실행 (json)"
	@echo "  make run-en TITLE='Clean Code'                   영문 모드"
	@echo "  make run-kr TITLE='클린 코드'                     한국어 모드"
	@echo "  make run-text TITLE='Clean Code'                 텍스트 출력"
	@echo "  make run-full TITLE='Clean Code'                 풀모드 (메타+TOC+리뷰+강의+유사서)"
	@echo "  make run-full-en TITLE='Clean Code'              풀모드 + 영문"
	@echo "  make run-full-kr TITLE='클린 코드'                풀모드 + 한국어"
	@echo "  make run-nocache TITLE='Clean Code'              캐시 무시"
	@echo "  make run-full-nocache TITLE='Clean Code'         풀모드 + 캐시 무시"
	@echo ""
	@echo "ISBN 실행:"
	@echo "  make run-isbn ISBN='978-0132350884'              ISBN 조회"
	@echo "  make run-isbn-en ISBN='978-0132350884'           ISBN + 영문"
	@echo ""
	@echo "배치:"
	@echo "  make run-batch BATCH='titles.txt'                배치 실행"
	@echo ""
	@echo "비교 스크립트:"
	@echo "  make compare                                     OpenRouter 라이브 비교"
	@echo "  make compare-dashscope                           DashScope 라이브 비교"
	@echo ""
	@echo "조합 변수 (모든 run-* 타겟에 추가 가능):"
	@echo "  L=en|ko|ja|zh-tw       언어 지정"
	@echo "  AUTHOR='Robert Martin' 저자 필터"
	@echo "  MODEL='qwen-plus'      모델 변경"
	@echo "  OUTPUT='out.json'      출력 경로"
	@echo "  FORMAT=json|text       출력 형식"
	@echo "  NOCACHE=1              캐시 무시"
	@echo "  FULL=1                 풀모드"
	@echo "  TIMEOUT_SEC=180        타임아웃 (기본 120)"
	@echo "  ARGS='--extra-flag'    추가 플래그"
	@echo ""
	@echo "조합 예시:"
	@echo "  make run TITLE='Clean Code' L=en AUTHOR='Robert C. Martin' NOCACHE=1"
	@echo "  make run TITLE='클린 코드' L=ko FORMAT=text FULL=1"

# ── 빌드/테스트 ──────────────────────────────────────
build:
	@mkdir -p bin
	go build -o $(BIN) ./cmd/bookinfo

test:
	go test ./...

clean:
	rm -f $(BIN)

# ── 실행: 타이틀 기반 ────────────────────────────────

run: build
	@if [[ -z "$(TITLE)" ]]; then echo "ERROR: TITLE 필수. 예) make run TITLE='Clean Code'"; exit 1; fi
	$(call exec,$(BIN) "$(TITLE)" $(_FLAGS) $(ARGS))

run-en: build
	@if [[ -z "$(TITLE)" ]]; then echo "ERROR: TITLE 필수. 예) make run-en TITLE='Clean Code'"; exit 1; fi
	$(call exec,$(BIN) "$(TITLE)" --lang en $(_FLAGS) $(ARGS))

run-kr: build
	@if [[ -z "$(TITLE)" ]]; then echo "ERROR: TITLE 필수. 예) make run-kr TITLE='클린 코드'"; exit 1; fi
	$(call exec,$(BIN) "$(TITLE)" --lang ko $(_FLAGS) $(ARGS))

run-text: build
	@if [[ -z "$(TITLE)" ]]; then echo "ERROR: TITLE 필수. 예) make run-text TITLE='Clean Code'"; exit 1; fi
	$(call exec,$(BIN) "$(TITLE)" --format text $(_FLAGS) $(ARGS))

run-full: build
	@if [[ -z "$(TITLE)" ]]; then echo "ERROR: TITLE 필수. 예) make run-full TITLE='Clean Code'"; exit 1; fi
	$(call exec,$(BIN) "$(TITLE)" --full $(_FLAGS) $(ARGS))

run-full-en: build
	@if [[ -z "$(TITLE)" ]]; then echo "ERROR: TITLE 필수. 예) make run-full-en TITLE='Clean Code'"; exit 1; fi
	$(call exec,$(BIN) "$(TITLE)" --full --lang en $(_FLAGS) $(ARGS))

run-full-kr: build
	@if [[ -z "$(TITLE)" ]]; then echo "ERROR: TITLE 필수. 예) make run-full-kr TITLE='클린 코드'"; exit 1; fi
	$(call exec,$(BIN) "$(TITLE)" --full --lang ko $(_FLAGS) $(ARGS))

run-nocache: build
	@if [[ -z "$(TITLE)" ]]; then echo "ERROR: TITLE 필수. 예) make run-nocache TITLE='Clean Code'"; exit 1; fi
	$(call exec,$(BIN) "$(TITLE)" --no-cache $(_FLAGS) $(ARGS))

run-full-nocache: build
	@if [[ -z "$(TITLE)" ]]; then echo "ERROR: TITLE 필수. 예) make run-full-nocache TITLE='Clean Code'"; exit 1; fi
	$(call exec,$(BIN) "$(TITLE)" --full --no-cache $(_FLAGS) $(ARGS))

# ── 실행: ISBN 기반 ──────────────────────────────────

run-isbn: build
	@if [[ -z "$(ISBN)" ]]; then echo "ERROR: ISBN 필수. 예) make run-isbn ISBN='978-0132350884'"; exit 1; fi
	$(call exec,$(BIN) --isbn "$(ISBN)" $(_FLAGS) $(ARGS))

run-isbn-en: build
	@if [[ -z "$(ISBN)" ]]; then echo "ERROR: ISBN 필수. 예) make run-isbn-en ISBN='978-0132350884'"; exit 1; fi
	$(call exec,$(BIN) --isbn "$(ISBN)" --lang en $(_FLAGS) $(ARGS))

# ── 실행: 배치 ───────────────────────────────────────

run-batch: build
	@if [[ -z "$(BATCH)" ]]; then echo "ERROR: BATCH 필수. 예) make run-batch BATCH='titles.txt'"; exit 1; fi
	$(call exec,$(BIN) --batch "$(BATCH)" $(_FLAGS) $(ARGS))

# ── 비교 스크립트 ────────────────────────────────────

compare:
	$(call exec,python3 scripts/live_compare_openrouter.py $(ARGS))

compare-dashscope:
	$(call exec,python3 scripts/dashscope_compare.py $(ARGS))

compare-cli:
	$(call exec,python3 scripts/dashscope_compare.py --cli $(ARGS))
