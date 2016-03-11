package auth

import (
	backends "github.com/cchamplin/deployd/backends/auth"
	"github.com/dgrijalva/jwt-go"
	"strings"
	"time"
)

type Auth struct {
	AuthToken    string
	AuthEndpoint string
	Backend      AuthenticationBackend
}

func CreatToken(accountID int, roles Roles) string {
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims["account"] = accountID
	token.Claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	tokenString, err := token.SignedString("temp")
	if err != nil {

	}
	return tokenString
}

type AuthenticationBackend interface {
	Authenticate(authParams map[string]string) (interface{}, error)
	GetRoles() Roles
	GetRole(id string) Role
	GetUsers() interface{}
	GetUser(id string) interface{}
	DeleteUser(id string) error
	UpdateUser(id string, userData map[string]string) (interface{}, error)
	CreateUser(id string, userData map[string]string) (interface{}, error)
	CreateRole(roleData map[string]string) (Role, error)
	UpdateRole(roleData map[string]string) (Role, error)
}

func AuthFromConfig(config map[string]interface{}) Auth {
	t, ok := config["type"]
	var backendType string
	if !ok {
		backendType = "default"
	} else {
		backendType = t.(string)
	}

	t, ok = config["authtoken"]
	var token string
	if !ok {
		// TODO error
	} else {
		token = t.(string)
	}

	var auth Auth
	auth = Auth{AuthToken: token}
	switch strings.ToLower(backendType) {
	case "default":
		defaultAuth := backends.DefaultAuth{}
		defaultAuth.Init(auth.AuthEndpoint)
		auth.Backend = etcdAuth
	case "etcd":
		etcdAuth := backends.EtcdAuth{}
		etcdAuth.Init(auth.AuthEndpoint)
		auth.Backend = etcdAuth
	}

	return j
}
