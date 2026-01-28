package embedder

// e5ReadyMsg is the initial message sent by the Python worker.
type e5ReadyMsg struct {
	Status     string `json:"status"`
	Dimensions int    `json:"dimensions"`
}

// e5Request is sent to the Python worker via stdin.
type e5Request struct {
	ID        string   `json:"id"`
	Texts     []string `json:"texts"`
	InputType string   `json:"input_type"`
}

// e5Response is received from the Python worker via stdout.
type e5Response struct {
	ID         string      `json:"id"`
	Embeddings [][]float64 `json:"embeddings"`
	Dimensions int         `json:"dimensions"`
	Error      string      `json:"error,omitempty"`
}
