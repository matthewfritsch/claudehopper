package profile

import (
	"testing"
)

func TestValidateProfileName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid lowercase", "myprofile", false},
		{"valid with hyphen", "my-profile", false},
		{"valid with underscore", "my_profile", false},
		{"valid with numbers", "profile123", false},
		{"mixed case normalized", "My-Profile", false},
		{"empty string", "", true},
		{"starts with hyphen", "-bad", true},
		{"has slash", "has/slash", true},
		{"has space", "has space", true},
		{"only whitespace", "   ", true},
		{"starts with underscore", "_bad", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfileName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProfileName(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeProfileName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"MyProfile", "myprofile"},
		{"  MyProfile  ", "myprofile"},
		{"my-profile", "my-profile"},
		{"MY_PROFILE", "my_profile"},
		{"  mixed CASE  ", "mixed case"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeProfileName(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeProfileName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
