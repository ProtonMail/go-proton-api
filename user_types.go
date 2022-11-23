package proton

type User struct {
	ID          string
	Name        string
	DisplayName string
	Email       string
	Keys        Keys

	UsedSpace int
	MaxSpace  int
	MaxUpload int

	Credit   int
	Currency string
}

type TokenType string

const (
	EmailTokenType TokenType = "email"
	SMSTokenType   TokenType = "sms"
)

type SendVerificationCodeReq struct {
	Username    string
	Type        TokenType
	Destination TokenDestination
}

type TokenDestination struct {
	Address string
	Phone   string
}

type UserType int

const (
	MailUserType UserType = iota + 1
	VPNUserType
)

type CreateUserReq struct {
	Username  string
	Email     string `json:",omitempty"`
	Phone     string `json:",omitempty"`
	Token     string
	TokenType TokenType
	Type      UserType
	Auth      CreateUserAuth
}

type CreateUserAuth struct {
	Version   int
	ModulusID string
	Salt      string
	Verifier  string
}
