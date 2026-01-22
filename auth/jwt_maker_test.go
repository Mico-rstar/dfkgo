package auth

func TestJwtMakeToken(t *testing.T) {
	maker := NewJwtMaker()
	maker.MakeToken()
}