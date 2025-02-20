.PHONY: all
all: sonicd sonictool

GOPROXY ?= "https://proxy.golang.org,direct"
.PHONY: sonicd sonictool
sonicd:
	GIT_COMMIT=`git rev-list -1 HEAD 2>/dev/null || echo ""` && \
	GIT_DATE=`git log -1 --date=short --pretty=format:%ct 2>/dev/null || echo ""` && \
	GOPROXY=$(GOPROXY) \
	go build \
	    -ldflags "-s -w -X github.com/0xsoniclabs/sonic/config.GitCommit=$${GIT_COMMIT} -X github.com/0xsoniclabs/sonic/config.GitDate=$${GIT_DATE}" \
	    -o build/sonicd \
	    ./cmd/sonicd

sonictool:
	GIT_COMMIT=`git rev-list -1 HEAD 2>/dev/null || echo ""` && \
	GIT_DATE=`git log -1 --date=short --pretty=format:%ct 2>/dev/null || echo ""` && \
	GOPROXY=$(GOPROXY) \
	go build \
	    -ldflags "-s -w -X github.com/0xsoniclabs/sonic/config.GitCommit=$${GIT_COMMIT} -X github.com/0xsoniclabs/sonic/config.GitDate=$${GIT_DATE}" \
	    -o build/sonictool \
	    ./cmd/sonictool

TAG ?= "latest"
.PHONY: sonic-image
sonic-image:
	docker build \
    	    --network=host \
    	    -f ./docker/Dockerfile.opera -t "sonic:$(TAG)" .

.PHONY: test
test:
	go test -cover ./...

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

.PHONY: lint
lint: vet staticcheck # errorcheck  
