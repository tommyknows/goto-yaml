package values

var (
	// DefaultValues defines the default values for the Helm chart
	DefaultValues = Values{
		// this is a test comment
		Config: Config{
			X: "hello",
			// we set Y to false because it's better that way.
			Y: false,
		},
		// y
		Image: "hi",
	}
)

type Values struct {
	// Config defines the configuration for this Chart.
	Config Config `json:"config"`
	Image  string `json:"image"`
}

type Config struct {
	// X is cool
	X string `json:"x"`
	// Y is not.
	Y bool `json:"y"`
}
