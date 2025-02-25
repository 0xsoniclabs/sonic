package basestreamseeder

type Config struct {
	SenderThreads           int
	MaxSenderTasks          int
	MaxPendingResponsesSize int64
	MaxResponsePayloadNum   uint32
	MaxResponsePayloadSize  uint64
	MaxResponseChunks       uint32
}
