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

type DeleteUserReq struct {
	Reason   string
	Feedback string
	Email    string
}
