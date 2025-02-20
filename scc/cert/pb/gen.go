package pb

//go:generate protoc -I=./ --go_out=./ ./signature.proto
//go:generate protoc -I=./ --go_out=./ ./block_certificate.proto
//go:generate protoc -I=./ --go_out=./ ./committee_certificate.proto
