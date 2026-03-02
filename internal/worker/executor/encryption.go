package executor

import "fmt"

// GenerateEncryptionOverride produces an OpenTofu override file that enables
// native state and plan encryption (AES-GCM via PBKDF2 key derivation).
// This is an OpenTofu 1.7+ feature.
func GenerateEncryptionOverride(passphrase string) string {
	return fmt.Sprintf(`terraform {
  encryption {
    key_provider "pbkdf2" "state_key" {
      passphrase = %q
    }
    method "aes_gcm" "state_enc" {
      keys = key_provider.pbkdf2.state_key
    }
    state {
      method = method.aes_gcm.state_enc
    }
    plan {
      method = method.aes_gcm.state_enc
    }
  }
}
`, passphrase)
}
