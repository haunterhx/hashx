package scan

type Result struct {
	Path    string  `json:"path"`
	Size    int64   `json:"size"`
	SHA256  string  `json:"sha256"`
	Entropy float64 `json:"entropy"`
	IOCHit  bool    `json:"ioc_hit"`
	HighEnt bool    `json:"high_entropy"`
	Err     string  `json:"error,omitempty"`
}
