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
    	    -f ./docker/Dockerfile.opera -t "sonic:$(TAG)" .

.PHONY: test
test:
	go test -cover --timeout 30m ./...

.PHONY: coverage
coverage:
	go test -coverprofile=cover.prof $$(go list ./... | grep -v '/gossip/contract/' | grep -v '/gossip/emitter/mock' | xargs)
	go tool cover -func cover.prof | grep -e "^total:"

.PHONY: fuzz
fuzz:
	CGO_ENABLED=1 \
	mkdir -p ./fuzzing && \
	go run github.com/dvyukov/go-fuzz/go-fuzz-build -o=./fuzzing/gossip-fuzz.zip ./gossip && \
	go run github.com/dvyukov/go-fuzz/go-fuzz -workdir=./fuzzing -bin=./fuzzing/gossip-fuzz.zip

.PHONY: run-coverage
run-coverage: DATE=$(shell date +"%Y-%m-%d-%T")
run-coverage: export GOCOVERDIR=./build/coverage/${DATE}
run-coverage:
	@mkdir -p ${GOCOVERDIR} ;\
	go test -coverpkg=${COVERPACKAGES} --timeout=30m -coverprofile="${GOCOVERDIR}/${REPORTNAME}.out" ${TARGETPACKAGES} ;\
	go tool cover -html ${GOCOVERDIR}/${REPORTNAME}.out -o ${GOCOVERDIR}/${REPORTNAME}.html ;\
	echo "Coverage report generated in ${GOCOVERDIR}/${REPORTNAME}.html"

.PHONY: integration-cover-all
integration-cover-all: COVERPACKAGES=./...
integration-cover-all: TARGETPACKAGES=./tests
integration-cover-all: REPORTNAME="integration-cover"
integration-cover-all: run-coverage

.PHONY: unit-cover-all
unit-cover-all: COVERPACKAGES=`go list ./... | grep -v /tests`
unit-cover-all: TARGETPACKAGES=`go list ./... | grep -v /tests`
unit-cover-all: REPORTNAME="unit-cover"
unit-cover-all: run-coverage

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

.PHONY: vet
vet:
	go vet ./...

STATICCHECK_VERSION = 2025.1
.PHONY: staticcheck
staticcheck:
	@go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	staticcheck ./...

ERRCHECK_VERSION = v1.9.0
.PHONY: errcheck
errorcheck:
	@go install github.com/kisielk/errcheck@$(ERRCHECK_VERSION)
	errcheck ./...

.PHONY: deadcode
deadcode:
	@go install golang.org/x/tools/cmd/deadcode@latest
	deadcode -test ./...

.PHONY: lint
lint: vet staticcheck deadcode errorcheck
