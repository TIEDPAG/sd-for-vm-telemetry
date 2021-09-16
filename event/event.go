package event

const (
	TypePut Type = iota + 1
	TypeDelete
)

type Type int

func (t Type) String() string {
	switch t {
	case TypePut:
		return "PUT"
	case TypeDelete:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

type Event struct {
	Type      Type
	Namespace string
	Name      string
	Address   string
}

func (e *Event) IsNamespaceEvent() bool {
	return e.Address == ""
}
