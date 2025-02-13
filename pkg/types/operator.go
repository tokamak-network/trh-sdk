package types

type Operator int

const (
	Admin Operator = iota
	Sequencer
	Batcher
	Proposer
	Challenger
)

type IndexAccount struct {
	Index      int
	Address    string
	PrivateKey string
}

type OperatorMap map[Operator]IndexAccount
