package pb

//go:generate protoc -I=./ --go_out=./ ./transaction.proto
//go:generate protoc -I=./ --go_out=./ ./proposal.proto
//go:generate protoc -I=./ --go_out=./ ./payload.proto
