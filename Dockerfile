# Running Sonic in Docker is Experimental - not recommended for production use!

# Example of usage:
# docker build -t sonic .
# docker run --name sonic1 --entrypoint sonictool sonic --datadir=/var/sonic genesis fake 1
# docker run --volumes-from sonic1 -p 5050:5050 -p 5050:5050/udp -p 18545:18545 sonic --fakenet 1/1 --http --http.addr=0.0.0.0

FROM golang:1.24 AS builder

RUN apt-get update && apt-get install -y git musl-dev make

WORKDIR /go/Sonic
COPY . .

ARG GOPROXY
RUN go mod download
RUN make all


FROM golang:1.24

COPY --from=builder /go/Sonic/build/sonicd /usr/local/bin/
COPY --from=builder /go/Sonic/build/sonictool /usr/local/bin/

EXPOSE 18545 18546 5050 5050/udp

VOLUME /var/sonic

ENTRYPOINT ["sonicd", "--datadir=/var/sonic"]
