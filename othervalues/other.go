package othervalues

type Other struct {
	Values []string `json:"values"`
	// Truth checks whether this is true or not.
	Truth bool `json:"truth"`
}

// Number is an amount.
type Number int
