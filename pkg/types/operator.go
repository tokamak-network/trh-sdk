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
	Address    string
	PrivateKey string
}

type OperatorMap map[Operator]*IndexAccount

type Operators struct {
	AdminPrivateKey      string
	SequencerPrivateKey  string
	BatcherPrivateKey    string
	ProposerPrivateKey   string
	ChallengerPrivateKey string
}
