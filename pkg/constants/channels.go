package constants

// Channel types for alert notifications
const (
	ChannelEmail    = "email"
	ChannelTelegram = "telegram"
)

// ChannelType represents the type of notification channel
type ChannelType string

// Valid channel types
const (
	EmailChannel    ChannelType = "email"
	TelegramChannel ChannelType = "telegram"
)

// IsValidChannelType checks if the given channel type is valid
func IsValidChannelType(channelType string) bool {
	switch channelType {
	case ChannelEmail, ChannelTelegram:
		return true
	default:
		return false
	}
}

// GetValidChannelTypes returns a list of valid channel types
func GetValidChannelTypes() []string {
	return []string{ChannelEmail, ChannelTelegram}
}
