package constants

const (
	AWS   = "aws"
	Local = "local"
)

var SupportedInfra = map[string]bool{
	"aws":   true,
	"local": true,
}
