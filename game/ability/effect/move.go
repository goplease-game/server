package effect

type MoveType string

const (
	MovePush MoveType = "push"
	MoveTo   MoveType = "to"
	MoveSwap MoveType = "swap"
)

type Move struct {
	Type     MoveType `json:"type"`
	Distance int      `json:"distance"`
}
