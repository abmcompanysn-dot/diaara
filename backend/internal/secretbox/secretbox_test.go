package secretbox

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func testKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(i)
	}
	return k
}

func TestRoundTrip(t *testing.T) {
	b, err := New(testKey())
	if err != nil {
		t.Fatal(err)
	}
	enc, err := b.Encrypt("EAAB-jeton-secret")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, "v1:") || strings.Contains(enc, "jeton") {
		t.Fatalf("valeur chiffrée inattendue: %s", enc)
	}
	enc2, _ := b.Encrypt("EAAB-jeton-secret")
	if enc == enc2 {
		t.Fatal("le nonce doit rendre chaque chiffrement unique")
	}
	plain, err := b.Decrypt(enc)
	if err != nil || plain != "EAAB-jeton-secret" {
		t.Fatalf("déchiffrement: %q %v", plain, err)
	}
}

func TestTamperAndWrongKey(t *testing.T) {
	b, _ := New(testKey())
	enc, _ := b.Encrypt("secret")
	raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(enc, "v1:"))
	raw[len(raw)-1] ^= 0x01
	if _, err := b.Decrypt("v1:" + base64.StdEncoding.EncodeToString(raw)); !errors.Is(err, ErrInvalidCipher) {
		t.Fatalf("altération non détectée: %v", err)
	}
	other := testKey()
	other[0] = 99
	b2, _ := New(other)
	if _, err := b2.Decrypt(enc); !errors.Is(err, ErrInvalidCipher) {
		t.Fatalf("mauvaise clé acceptée: %v", err)
	}
	for _, bad := range []string{"", "v1:", "v1:!!!", "secret", "v1:" + base64.StdEncoding.EncodeToString([]byte("court"))} {
		if _, err := b.Decrypt(bad); err == nil {
			t.Fatalf("valeur invalide acceptée: %q", bad)
		}
	}
}

func TestParseKey(t *testing.T) {
	k := testKey()
	for _, s := range []string{
		hex.EncodeToString(k),
		base64.StdEncoding.EncodeToString(k),
		base64.RawURLEncoding.EncodeToString(k),
		"  " + base64.StdEncoding.EncodeToString(k) + "\n",
	} {
		got, err := ParseKey(s)
		if err != nil || string(got) != string(k) {
			t.Fatalf("clé %q refusée: %v", s, err)
		}
	}
	for _, s := range []string{"", "trop-court", base64.StdEncoding.EncodeToString(make([]byte, 16)), strings.Repeat("z", 64)} {
		if _, err := ParseKey(s); err == nil {
			t.Fatalf("clé invalide acceptée: %q", s)
		}
	}
	if _, err := New(make([]byte, 16)); err == nil {
		t.Fatal("clé de 16 octets acceptée")
	}
	if _, err := NewFromString(hex.EncodeToString(k)); err != nil {
		t.Fatal(err)
	}
}
