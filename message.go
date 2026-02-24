package notify

// Message is a provider-agnostic notification payload.
// Providers can map only the fields they support.
type Message struct {
	Subject string
	Text    string
	HTML    string
	Meta    map[string]string
}
