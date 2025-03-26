package producer

type Config struct {
	Topics              string `json:"topics"`
	SaveReturningStatus struct {
		// fetch error response in channel
		Errors bool `json:"errors"`
		// fetch successes response in channel
		Succeeded bool `json:"succeeded"`
	}
	// CompressionType must be equal [0,1,2,3,4]
	CompressionType int8 `json:"compression_type"`
	// RequiredAcks must be equal [-1,0,1]
	RequiredAcks int16 `json:"required_acks"`
}

func SetDefaultConnectionSettings()
