package values

import "github.com/tommyknows/goto-yaml/othervalues"

type Values struct {
	// Config defines the configuration for this Chart.
	Config Config `json:"config"`
	// Count defines the number of things.
	Count  othervalues.Number `json:"count"`
	Unused string             `json:"unused"`
	Image  string             `json:"image"`
	Other  othervalues.Other  `json:"other"`
}

var (
	// DefaultValues defines the default values for the Helm chart
	DefaultValues = Values{
		// 8 is the best.
		Count: 8,
		// this is a test comment
		Config: Config{
			X: "hello",
			// we set Y to false because it's better that way.
			Y: false,
			// map m defines greetings and goodbyes
			M: map[string]string{
				"hello": "world",
				// sleep well little moon
				"goodbye": "moon",
			},
		},
		// y
		Image: "hi",
		Other: othervalues.Other{
			// We are not lying.
			Truth: true,
			// Values are cool.
			Values: []string{
				// does it?
				"hello",
				// does this automagically work?
				"abc",
			},
		},
	}
)
