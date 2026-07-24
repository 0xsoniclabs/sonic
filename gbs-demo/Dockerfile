# Transaction-priorities demo image: a single container that runs a full
# multi-validator Sonic fake-net with the priorities feature enabled and a set
# of pre-funded demo accounts.
#
# Build from the repository root:
#   docker build -f gbs-demo/Dockerfile -t sonic-priorities-demo .
# Run:
#   docker run --rm -p 18545:18545 sonic-priorities-demo

FROM golang:1.26 AS builder

RUN apt-get update && apt-get install -y git make && rm -rf /var/lib/apt/lists/*

WORKDIR /go/sonic
COPY . .

ARG GOPROXY
RUN go mod download
RUN make all
RUN go build -o /go/bin/genesisgen ./gbs-demo/genesisgen


FROM golang:1.26

RUN apt-get update && apt-get install -y --no-install-recommends curl jq \
    && rm -rf /var/lib/apt/lists/*

ENV PATH="/root/.foundry/bin:${PATH}"
RUN curl -L https://foundry.paradigm.xyz | bash && foundryup \
    && ln -s /root/.foundry/bin/* /usr/local/bin/

COPY --from=builder /go/sonic/build/sonicd /usr/local/bin/
COPY --from=builder /go/sonic/build/sonictool /usr/local/bin/
COPY --from=builder /go/bin/genesisgen /usr/local/bin/
COPY gbs-demo/entrypoint.sh /usr/local/bin/entrypoint.sh
COPY gbs-demo/USAGE.md /USAGE.md
RUN chmod +x /usr/local/bin/entrypoint.sh

# HTTP RPC for validators 1..5 and the WebSocket endpoint of validator 1.
EXPOSE 18545 18546 18547 18548 18549 18645

# No VOLUME for /data: the chain lives in the container's writable layer so it
# is recreated fresh on every start and env-var overrides always take effect.
# Opt into persistence with the named volume in docker-compose.yml instead.

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
