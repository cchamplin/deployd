package auth

import "time"

func CreatToken(int accountID, Permissions perms) {
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims[""]
	token.Claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	tokenString, err := token.SignedString()
}
