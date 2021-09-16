package event

const (
	TypePut Type = iota + 1
	TypeDelete
)

type Type int

type Event struct {
	Type      Type
	Namespace string
	Address   string
}

func (e *Event) IsNamespaceEvent() bool {
	return e.Address == ""
}
