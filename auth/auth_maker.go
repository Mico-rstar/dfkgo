package auth


type AuthMaker interface {
	MakeToken(userId int64) (string, error)
	VerifyToken(token string) (int64, error)
}

