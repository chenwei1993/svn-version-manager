package secure

import "testing"

func TestEncryptDecrypt(t *testing.T) {
	const secret = "p@ss'word\"测试"
	enc, err := Encrypt(secret)
	if err != nil {
		t.Fatal(err)
	}
	if enc == "" {
		t.Fatal("encrypted value is empty")
	}
	dec, err := Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != secret {
		t.Fatalf("decrypted = %q, want %q", dec, secret)
	}
}

func TestEncryptEmpty(t *testing.T) {
	enc, err := Encrypt("")
	if err != nil {
		t.Fatal(err)
	}
	if enc != "" {
		t.Fatalf("empty encrypt = %q, want empty", enc)
	}
	dec, err := Decrypt("")
	if err != nil {
		t.Fatal(err)
	}
	if dec != "" {
		t.Fatalf("empty decrypt = %q, want empty", dec)
	}
}
