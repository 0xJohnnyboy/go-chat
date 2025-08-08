package http

type UserRegisterInput struct {
	Name     string  `json:"name"`
	Password *string `json:"password"`
}

type UserLoginInput struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}
