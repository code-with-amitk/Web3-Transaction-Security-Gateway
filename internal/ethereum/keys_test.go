package ethereum

import "testing"

func TestNormalizePrivateKeyHex(t *testing.T) {
	valid := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	got, err := normalizePrivateKeyHex(valid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(got))
	}

	_, err = normalizePrivateKeyHex("0xac0974bec39a17e36ba4a6a679b5249696125143991808455148b4754e44583")
	if err == nil {
		t.Fatal("expected error for truncated key")
	}
}
