.PHONY: all
all: sonicd sonictool

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
		-f ./Dockerfile -t "sonic:$(TAG)" .

.PHONY: test
test:
	go test --timeout 30m ./...

.PHONY: coverage
coverage:
	@mkdir -p build ;\
	go test -coverpkg=./... --timeout=30m -coverprofile=build/coverage.cov ./... && \
	go tool cover -html build/coverage.cov -o build/coverage.html &&\
	echo "Coverage report generated in build/coverage.html"

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

.PHONY: license-check
license-check:
	go run ./scripts/license/add_license_header.go --check -dir ./

.PHONY: license-add
license-add:
	go run ./scripts/license/add_license_header.go -dir ./