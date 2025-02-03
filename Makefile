.PHONY: all
all: sonicd sonictool

GOPROXY ?= "https://proxy.golang.org,direct"
.PHONY: sonicd sonictool
sonicd:
	GIT_COMMIT=`git rev-list -1 HEAD 2>/dev/null || echo ""` && \
	GIT_DATE=`git log -1 --date=short --pretty=format:%ct 2>/dev/null || echo ""` && \
	GIT_TAG=`echo $(call get_git_tag)` && \
	GOPROXY=$(GOPROXY) \
	go build \
	    -ldflags "-s -w -X github.com/0xsoniclabs/sonic/config.GitCommit=$${GIT_COMMIT} \
				        -X github.com/0xsoniclabs/sonic/config.GitDate=$${GIT_DATE} \
						-X github.com/0xsoniclabs/sonic/version.Version=$${GIT_TAG}" \
	    -o build/sonicd \
	    ./cmd/sonicd && \
	    ./build/sonicd version

sonictool:
	GIT_COMMIT=`git rev-list -1 HEAD 2>/dev/null || echo ""` && \
	GIT_DATE=`git log -1 --date=short --pretty=format:%ct 2>/dev/null || echo ""` && \
	GIT_TAG=`echo  $(call get_git_tag)` && \
	GOPROXY=$(GOPROXY) \
	go build \
	    -ldflags "-s -w -X github.com/0xsoniclabs/sonic/config.GitCommit=$${GIT_COMMIT} \
				        -X github.com/0xsoniclabs/sonic/config.GitDate=$${GIT_DATE} \
						-X github.com/0xsoniclabs/sonic/version.Version=$${GIT_TAG}" \
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

STATICCHECK_VERSION = 2024.1.1
.PHONY: staticcheck
staticcheck: 
	@go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	staticcheck ./...

ERRCHECK_VERSION = v1.7.0
.PHONY: errcheck
errorcheck:
	@go install github.com/kisielk/errcheck@$(ERRCHECK_VERSION)
	errcheck ./...

.PHONY: deadcode
deadcode:
	@go install golang.org/x/tools/cmd/deadcode@latest
	deadcode -test ./...

.PHONY: lint
lint: vet staticcheck deadcode # errorcheck

# Define a function to compute the tag
define get_git_tag
    $(shell \
        GIT_TAG=$$(git describe --tags --abbrev=0 2>/dev/null); \
        COMMITS_SINCE=$$(git log $${GIT_TAG}..HEAD --oneline 2>/dev/null | wc -l); \
        DIRTY_STATE=$$(git status --porcelain 2>/dev/null); \
        FINAL_TAG=$${GIT_TAG}; \
        if [ "$${COMMITS_SINCE}" -ne 0 ]; then \
            FINAL_TAG=$${FINAL_TAG}-dev; \
        fi; \
        if [ -n "$${DIRTY_STATE}" ]; then \
            FINAL_TAG=$${FINAL_TAG}-dirty; \
        fi; \
        echo "$${FINAL_TAG}"
    )
endef
