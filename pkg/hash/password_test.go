package hash

import "testing"

func TestValidateNewPassword(t *testing.T) {
	tests := []struct {
		name    string
		pw      string
		wantErr bool
	}{
		{"ok", "Abcd1234!", false},
		{"too short", "Ab1!", true},
		{"no special", "Abcd12345", true},
		{"no upper", "abcd1234!", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNewPassword(tt.pw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateNewPassword(%q) err=%v wantErr=%v", tt.pw, err, tt.wantErr)
			}
		})
	}
}

func TestHashOTPDeterministic(t *testing.T) {
	a := HashOTP("123456", "pepper")
	b := HashOTP("123456", "pepper")
	if a != b {
		t.Fatal("HashOTP should be deterministic")
	}
	if a == "123456" {
		t.Fatal("HashOTP must not return plaintext")
	}
	if len(a) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(a))
	}
}

func TestDummyPasswordHashValid(t *testing.T) {
	if !CheckPassword("dummy-password", DummyPasswordHash) {
		t.Fatal("DummyPasswordHash must be a valid bcrypt of dummy-password")
	}
}
