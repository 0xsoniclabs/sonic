# syntax=docker/dockerfile:1
# Running Sonic in Docker is Experimental - not recommended for production use!

FROM golang:1.26.0 AS base

# Development stage — extends base so it does not require a full
# production build. The workspace is mounted by the dev container at runtime.
# Used by the dev container (devcontainer.json target: devcontainer).
# Not used in the production image.
#
# Example of manual usage:
#   docker build --target devcontainer --build-arg REMOTE_UID=$(id -u) --build-arg REMOTE_GID=$(id -g) -t sonic-dev-manual .
#   docker run --rm -it -v $(pwd):/workspace -w /workspace sonic-dev-manual bash
FROM base AS devcontainer

ARG REMOTE_USER=vscode
# These are just placeholders that get overwritten by devcontainer runtime
# (updateRemoteUserUID). For manual builds, pass --build-arg REMOTE_UID/GID.
ARG REMOTE_UID=1000
ARG REMOTE_GID=1000
RUN addgroup --gid ${REMOTE_GID} ${REMOTE_USER} \
    && adduser --disabled-password --gecos "" --uid ${REMOTE_UID} --gid ${REMOTE_GID} ${REMOTE_USER}

# Install system packages and system-level binaries as root, before the USER switch.
# Solidity compiler v0.8.30 (pinned binary — apt installs a different version)
RUN apt-get update && apt-get install -y --no-install-recommends curl \
    && curl -fsSL \
    https://github.com/ethereum/solc-bin/raw/gh-pages/linux-amd64/solc-linux-amd64-v0.8.30+commit.73712a01 \
    -o /usr/local/bin/solc \
    && chmod +x /usr/local/bin/solc

# Redirect GOPATH to the user's home so that go install writes to a
# user-writable location instead of /go (which is owned by root).
ENV HOME=/home/${REMOTE_USER}
ENV GOPATH=${HOME}/go
ENV PATH=${HOME}/go/bin:/usr/local/go/bin:${PATH}
USER ${REMOTE_USER}

# Go dev tools — installed to $GOPATH/bin = $HOME/go/bin, owned by the user.
RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.1 \
    && go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest

# Go file generators (versions pinned to match go.mod and the Makefile)
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6 \
    && go install github.com/ethereum/go-ethereum/cmd/abigen@v1.17.1 \
    && go install go.uber.org/mock/mockgen@v0.6.0

# Production stage
FROM base AS builder

WORKDIR /go/Sonic
COPY . .

RUN apt-get update && apt-get install -y git musl-dev make

ARG GOPROXY
RUN go mod download
RUN make all

# Final production image

# Example of usage:
#   docker build --target production -t sonic .
#   docker run --name sonic1 --entrypoint sonictool sonic --datadir=/var/sonic genesis fake 1
#   docker run --volumes-from sonic1 -p 5050:5050 -p 5050:5050/udp -p 18545:18545 sonic --fakenet 1/1 --http --http.addr=0.0.0.0
FROM base AS production

COPY --from=builder /go/Sonic/build/sonicd /usr/local/bin/
COPY --from=builder /go/Sonic/build/sonictool /usr/local/bin/

EXPOSE 18545 18546 5050 5050/udp

VOLUME /var/sonic

ENTRYPOINT ["sonicd", "--datadir=/var/sonic"]
