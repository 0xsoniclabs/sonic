package basestream

type Request struct {
	Session        Session
	Type           RequestType
	MaxPayloadNum  uint32
	MaxPayloadSize uint64
	MaxChunks      uint32
}

type Response struct {
	SessionID uint32
	Done      bool
	Payload   Payload
}

type Session struct {
	ID    uint32
	Start Locator
	Stop  Locator
}

type Locator interface {
	Compare(b Locator) int
	Inc() Locator
}

type Payload interface {
	Len() int
	TotalSize() uint64
	TotalMemSize() int
}

type RequestType uint8
