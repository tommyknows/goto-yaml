package values

// Config defines the configuration for an object.
type Config struct {
	// X is cool
	X string `json:"x"`
	// Y is not.
	Y bool `json:"y"`

	M map[string]string `json:"m"`
}
