package utils

import (
	"strings"
	"testing"
)

func TestHashString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{
			name:    "valid password",
			input:   "password123",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: false,
		},
		{
			name:    "long password",
			input:   strings.Repeat("a", 1000),
			wantErr: true, // bcrypt has 72 byte limit
		},
		{
			name:    "special characters",
			input:   "!@#$%^&*()_+-={}[]|\\:;\"'<>?,./",
			wantErr: false,
		},
		{
			name:    "unicode characters",
			input:   "こんにちは🎉",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashString(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("HashString() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("HashString() unexpected error: %v", err)
				return
			}

			// Hash should not be empty
			if hash == "" {
				t.Errorf("HashString() returned empty hash")
			}

			// Hash should be different from original
			if hash == tt.input {
				t.Errorf("HashString() hash should be different from original string")
			}

			// Hash should start with bcrypt prefix
			if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
				t.Errorf("HashString() hash should have bcrypt prefix, got: %s", hash)
			}

			// Hash should be consistent length (around 60 chars for bcrypt)
			if len(hash) < 50 || len(hash) > 80 {
				t.Errorf("HashString() hash length unexpected: %d", len(hash))
			}
		})
	}
}

func TestVerifyHashedString(t *testing.T) {
	// Test with known good hash
	password := "testpassword123"
	hash, err := HashString(password)
	if err != nil {
		t.Fatalf("Failed to create test hash: %v", err)
	}

	tests := []struct {
		name           string
		originalString string
		hashedString   string
		expected       bool
	}{
		{
			name:           "correct password",
			originalString: password,
			hashedString:   hash,
			expected:       true,
		},
		{
			name:           "wrong password",
			originalString: "wrongpassword",
			hashedString:   hash,
			expected:       false,
		},
		{
			name:           "empty original string",
			originalString: "",
			hashedString:   hash,
			expected:       false,
		},
		{
			name:           "empty hash",
			originalString: password,
			hashedString:   "",
			expected:       false,
		},
		{
			name:           "invalid hash format",
			originalString: password,
			hashedString:   "invalid-hash",
			expected:       false,
		},
		{
			name:           "case sensitive check",
			originalString: strings.ToUpper(password),
			hashedString:   hash,
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyHashedString(tt.originalString, tt.hashedString)
			
			if result != tt.expected {
				t.Errorf("VerifyHashedString() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHashAndVerifyConsistency(t *testing.T) {
	testPasswords := []string{
		"simplepassword",
		"",
		"complex!P@ssw0rd$123",
		"normalpassword", // Remove the overly long password
		"🔐🌟password🎉",
	}

	for _, password := range testPasswords {
		t.Run("password_"+password, func(t *testing.T) {
			// Hash the password
			hash, err := HashString(password)
			if err != nil {
				t.Errorf("HashString() failed: %v", err)
				return
			}

			// Verify the password matches the hash
			if !VerifyHashedString(password, hash) {
				t.Errorf("VerifyHashedString() failed for password that was just hashed")
			}

			// Verify a different password doesn't match
			if password != "" { // Skip for empty string test
				wrongPassword := password + "wrong"
				if VerifyHashedString(wrongPassword, hash) {
					t.Errorf("VerifyHashedString() incorrectly verified wrong password")
				}
			}
		})
	}
}

func TestMultipleHashesAreDifferent(t *testing.T) {
	password := "testpassword"
	
	hash1, err := HashString(password)
	if err != nil {
		t.Fatalf("Failed to create first hash: %v", err)
	}

	hash2, err := HashString(password)
	if err != nil {
		t.Fatalf("Failed to create second hash: %v", err)
	}

	// Even with the same password, hashes should be different due to salt
	if hash1 == hash2 {
		t.Errorf("Multiple hashes of same password should be different due to salt")
	}

	// But both should verify correctly
	if !VerifyHashedString(password, hash1) {
		t.Errorf("First hash should verify correctly")
	}

	if !VerifyHashedString(password, hash2) {
		t.Errorf("Second hash should verify correctly")
	}
}