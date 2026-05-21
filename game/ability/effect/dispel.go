package effect

type DispelLevel string

const (
	DispelAll      = "all"
	DispelPositive = "positive"
	DispelNegative = "negative"
)

type Dispel struct {
	Level DispelLevel `json:"level"`
}
