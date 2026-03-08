package constants

const (
	AWS          = "aws"
	DigitalOcean = "digitalocean"
)

var SupportedInfra = map[string]bool{
	"aws":          true,
	"digitalocean": true,
}
