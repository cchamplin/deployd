package auth

type Role struct {
	Id          string        `json:"id"`
	Name        string        `json:"name"`
	Permissions []Permissions `json:"permissions"`
}
