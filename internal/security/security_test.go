package security

import "testing"

func TestPasswordHashAndSecretDerivation(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Fatal("expected wrong password to fail")
	}

	a := DeriveSecretKey("team-123", "correct horse battery staple")
	b := DeriveSecretKey("team-123", "correct horse battery staple")
	c := DeriveSecretKey("team-456", "correct horse battery staple")
	if a == "" || a != b {
		t.Fatal("secret key derivation must be stable")
	}
	if a == c {
		t.Fatal("different group IDs must derive different keys")
	}
}

func TestTokenHash(t *testing.T) {
	token, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	hash := HashToken(token)
	if err := RequireToken(token, hash); err != nil {
		t.Fatal(err)
	}
	if err := RequireToken("bad", hash); err == nil {
		t.Fatal("expected invalid token to fail")
	}
}
