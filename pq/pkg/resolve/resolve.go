package resolve

import "context"

type Outcome int

const (
	Unknown Outcome = iota
	Yes
	No
)

func (o Outcome) String() string {
	switch o {
	case Yes:
		return "Yes"
	case No:
		return "No"
	default:
		return "Unknown"
	}
}

type Market struct {
	ID               string
	Question         string
	ConditionID      string
	EndDate          string
	ResolutionSource string
}

type Result struct {
	Outcome    Outcome
	Confidence float64
	Reasoning  string
	Sources    []string
}

type Resolver interface {
	Name() string
	CanResolve(m Market) bool
	Resolve(ctx context.Context, m Market) (Result, error)
}
