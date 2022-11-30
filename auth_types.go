package proton

type AuthInfoReq struct {
	Username string
}

type AuthInfo struct {
	Version         int
	Modulus         string
	ServerEphemeral string
	Salt            string
	SRPSession      string
	TwoFA           TwoFAInfo `json:"2FA"`
}

type U2FReq struct {
	KeyHandle     string
	ClientData    string
	SignatureData string
}

type AuthReq struct {
	Username        string
	ClientEphemeral string
	ClientProof     string
	SRPSession      string
	U2F             U2FReq
}

type Auth struct {
	UserID string

	UID          string
	AccessToken  string
	RefreshToken string
	ServerProof  string

	Scope        string
	TwoFA        TwoFAInfo `json:"2FA"`
	PasswordMode PasswordMode
}

type RegisteredKey struct {
	Version   string
	KeyHandle string
}

type U2FInfo struct {
	Challenge      string
	RegisteredKeys []RegisteredKey
}

type TwoFAInfo struct {
	Enabled TwoFAStatus
	U2F     U2FInfo
}

type TwoFAStatus int

const (
	TwoFADisabled TwoFAStatus = iota
	TOTPEnabled
)

type PasswordMode int

const (
	OnePasswordMode PasswordMode = iota + 1
	TwoPasswordMode
)

type Auth2FAReq struct {
	TwoFactorCode string
}

type AuthRefreshReq struct {
	UID          string
	RefreshToken string
	ResponseType string
	GrantType    string
	RedirectURI  string
	State        string
}

type AuthSession struct {
	UID        string
	CreateTime int64

	ClientID  string
	MemberID  string
	Revocable Bool

	LocalizedClientName string
}
