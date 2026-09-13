package config

import "testing"

func TestLoadNormalizesWhatsAppPhone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"already digits only", "5543936180556", "5543936180556"},
		{"with plus, spaces and dash", "+55 43 93618-0556", "5543936180556"},
		{"unset", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WHATSAPP_PHONE", tt.input)
			cfg := Load()
			if cfg.WhatsAppPhone != tt.want {
				t.Errorf("got WhatsAppPhone=%q, want %q", cfg.WhatsAppPhone, tt.want)
			}
		})
	}
}
