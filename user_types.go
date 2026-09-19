package proton

type User struct {
	ID          string
	Name        string
	DisplayName string
	Email       string
	Keys        Keys

	UsedSpace uint64
	MaxSpace  uint64
	MaxUpload uint64

	Credit   int64
	Currency string

	ProductUsedSpace ProductUsedSpace
}

type DeleteUserReq struct {
	Reason   string
	Feedback string
	Email    string
}

type ProductUsedSpace struct {
	Calendar uint64
	Contact  uint64
	Drive    uint64
	Mail     uint64
	Pass     uint64
}
