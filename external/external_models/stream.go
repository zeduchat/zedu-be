package external_models

type StreamChunk struct {
	Data  []byte
	Done  bool
	Error error
}
