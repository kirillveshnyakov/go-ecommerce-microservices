package outbox

type Kind int

const (
	KindUndefined Kind = iota
	KindNotification
)

func (kind Kind) String() string {
	switch kind {
	case KindUndefined:
		return "undefined"
	case KindNotification:
		return "notification"
	default:
		return ""
	}
}

type Data struct {
	IdempotencyKey string
	Kind           Kind
	Data           []byte
}
