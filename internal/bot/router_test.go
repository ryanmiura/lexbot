package bot

import "testing"

func TestClassify(t *testing.T) {
	for _, test := range []struct {
		input, name, args string
	}{
		{" /LiSt\tNEW ", "list", "NEW"},
		{"/ajuda\n /status", "ajuda", "/status"},
		{"/VOCABULÁRIO", "vocabulario", ""},
		{"/", "", ""},
	} {
		t.Run(test.input, func(t *testing.T) {
			kind, command := Classify(test.input)
			if kind != MessageTypeCommand || command == nil || command.Name != test.name || command.Args != test.args {
				t.Fatalf("Classify(%q) = %v, %+v", test.input, kind, command)
			}
		})
	}
	if kind, command := Classify("look forward to"); kind != MessageTypeWord || command != nil {
		t.Fatalf("plain vocabulary classified as command: %v, %+v", kind, command)
	}
}
