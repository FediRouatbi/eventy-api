package users

import "errors"

var ErrUserNotFound = errors.New("user not found")
var ErrInvalidName = errors.New("invalid name")
var ErrInvalidEmail = errors.New("invalid email")
var ErrEmailAlreadyExists = errors.New("email already exists")
var ErrCurrentPasswordWrong = errors.New("current password is incorrect")
