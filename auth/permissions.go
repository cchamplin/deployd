package auth

type Permission struct {
	Name  string `json:"name"`
	Flags int    `json:"flags"`
}

type Permissions []Permission

const (
	READ      = 1 << iota
	UPDATE    = 1 << iota
	CREATE    = 1 << iota
	DELETE    = 1 << iota
	RESERVED  = 1 << iota
	RESERVED2 = 1 << iota
	RESERVED3 = 1 << iota
	RESERVED4 = 1 << iota
)

func (p *Permission) CanRead() bool {
	return p.Flags&READ != 0
}

func (p *Permission) CanUpdate() bool {
	return p.Flags&UPDATE != 0
}

func (p *Permission) CanCreate() bool {
	return p.Flags&CREATE != 0
}

func (p *Permission) CanDelete() bool {
	return p.Flags&DELETE != 0
}
