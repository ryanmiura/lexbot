package whatsmeow

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestResolveSenderPhone(t *testing.T) {
	tests := []struct {
		name string
		info types.MessageInfo
		want string
	}{
		{
			name: "sender addressed by phone number JID",
			info: types.MessageInfo{
				MessageSource: types.MessageSource{
					Sender: types.NewJID("554399777658", types.DefaultUserServer),
				},
			},
			want: "554399777658",
		},
		{
			name: "sender addressed by LID with phone-number counterpart",
			info: types.MessageInfo{
				MessageSource: types.MessageSource{
					Sender:    types.NewJID("217449966350582", types.HiddenUserServer),
					SenderAlt: types.NewJID("554399777658", types.DefaultUserServer),
				},
			},
			want: "554399777658",
		},
		{
			name: "sender addressed by LID with no counterpart available",
			info: types.MessageInfo{
				MessageSource: types.MessageSource{
					Sender: types.NewJID("217449966350582", types.HiddenUserServer),
				},
			},
			want: "217449966350582",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveSenderPhone(tt.info); got != tt.want {
				t.Errorf("ResolveSenderPhone() = %q, want %q", got, tt.want)
			}
		})
	}
}
