.PHONY: all
all: sonicd sonictool

# build

GOPROXY ?= "https://proxy.golang.org,direct"
.PHONY: sonicd sonictool
sonicd:
	GIT_COMMIT=`git rev-list -1 HEAD 2>/dev/null || echo ""` && \
	GIT_DATE=`git log -1 --date=short --pretty=format:%ct 2>/dev/null || echo ""` && \
	GOPROXY=$(GOPROXY) \
	go build \
	    -ldflags "-s -w -X github.com/0xsoniclabs/sonic/version.gitCommit=$${GIT_COMMIT} \
	                    -X github.com/0xsoniclabs/sonic/version.gitDate=$${GIT_DATE}" \
	    -o build/sonicd \
	    ./cmd/sonicd && \
	    ./build/sonicd version

sonictool:
	GIT_COMMIT=`git rev-list -1 HEAD 2>/dev/null || echo ""` && \
	GIT_DATE=`git log -1 --date=short --pretty=format:%ct 2>/dev/null || echo ""` && \
	GOPROXY=$(GOPROXY) \
	go build \
	    -ldflags "-s -w -X github.com/0xsoniclabs/sonic/version.gitCommit=$${GIT_COMMIT} \
	                    -X github.com/0xsoniclabs/sonic/version.gitDate=$${GIT_DATE}" \
	    -o build/sonictool \
	    ./cmd/sonictool && \
	    ./build/sonictool --version

TAG ?= "latest"
.PHONY: sonic-image
sonic-image:
	docker build \
    	    --network=host \
    	    -f ./docker/Dockerfile.opera -t "sonic:$(TAG)" .

# test

.PHONY: test
test:
	go test --timeout 30m ./...

.PHONY: coverage
coverage:
	@mkdir -p build ;\
	go test -coverpkg=./... --timeout=30m -coverprofile=build/coverage.cov ./... && \
	go tool cover -html build/coverage.cov -o build/coverage.html &&\
	echo "Coverage report generated in build/coverage.html"

# Fuzzing

.PHONY: fuzz
fuzz:
	CGO_ENABLED=1 \
	mkdir -p ./fuzzing && \
	go run github.com/dvyukov/go-fuzz/go-fuzz-build -o=./fuzzing/gossip-fuzz.zip ./gossip && \
	go run github.com/dvyukov/go-fuzz/go-fuzz -workdir=./fuzzing -bin=./fuzzing/gossip-fuzz.zip


.PHONY: fuzz-txpool-validatetx-cover
fuzz-txpool-validatetx-cover: PACKAGES=./...,github.com/ethereum/go-ethereum/core/...
fuzz-txpool-validatetx-cover: DATE=$(shell date +"%Y-%m-%d-%T")
fuzz-txpool-validatetx-cover: export GOCOVERDIR=./build/coverage/fuzz-validate/${DATE}
fuzz-txpool-validatetx-cover: SEEDDIR=$$(go env GOCACHE)/fuzz/github.com/0xsoniclabs/sonic/evmcore/FuzzValidateTransaction/
fuzz-txpool-validatetx-cover: TEMPSEEDDIR=./evmcore/testdata/fuzz/FuzzValidateTransaction/
fuzz-txpool-validatetx-cover:
	@mkdir -p ${GOCOVERDIR} ;\
     mkdir -p ${TEMPSEEDDIR} ;\
	 go test -fuzz=FuzzValidateTransaction -fuzztime=2m ./evmcore/ ;\
     cp -r ${SEEDDIR}* ${TEMPSEEDDIR} ;\
     go test -v -run FuzzValidateTransaction -coverprofile=${GOCOVERDIR}/fuzz-txpool-validatetx-cover.out -coverpkg=${PACKAGES} ./evmcore/ ;\
     go tool cover -html ${GOCOVERDIR}/fuzz-txpool-validatetx-cover.out -o ${GOCOVERDIR}/fuzz-txpool-validatetx-coverage.html ;\
     rm -rf ${TEMPSEEDDIR} ;\

.PHONY: clean
clean:
	rm -fr ./build/*

# Linting

.PHONY: lint
lint: 
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6
	@golangci-lint run ./...

.PHONY: generated-check
generated-check:
	@ go generate ./... > /dev/null 2>&1 ;\
	 make license-add > /dev/null 2>&1
	@ git diff --exit-code > /dev/null 2>&1 && \
	 (echo "Generated files are up to date." && exit 0) || \
	 (echo "Generated files are not up to date. Please update them." && exit 1)

.PHONY: generators-version-check
generators-version-check:
	@echo "Checking tool versions..."
	@check_version() { \
		cmd=$$1; required=$$2; version_cmd=$$3; \
		version=$$($$version_cmd 2>&1 | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+' | head -n1); \
		if [ -z "$$version" ]; then \
			echo "$$cmd not found or version not detectable"; exit 1; \
		fi; \
		echo "$$cmd version: $$version"; \
		if [ "$$(printf "%s\n%s" "$$required" "$$version" | sort -V | head -n1)" != "$$required" ]; then \
			echo "$$cmd version must be >= $$required"; exit 1; \
		fi; \
	} && \
	check_version "mockgen" "0.5.0" "mockgen -version" && \
	check_version "protoc-gen-go" "1.30.0" "protoc-gen-go --version" && \
	check_version "solc" "0.8.20" "solc --version" && \
	echo "Generators meet version requirements."

.PHONY: generators-update
generators-update:
	@echo "Updating generators..."
	@go install go.uber.org/mock/mockgen@v0.5.0
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.30.0
	@echo "run manually: sudo apt-get install solc=1:0.8.30-0ubuntu1~noble"
	@echo "Generators updated."


# License checks

.PHONY: license-check
license-check:
	go run ./scripts/license/add_license_header.go --check -dir ./

.PHONY: license-add
license-add:
	go run ./scripts/license/add_license_header.go -dir ./