package commands

import "testing"

func TestWithoutEnvKeyRemovesOnlyExactVariable(t *testing.T) {
	got := withoutEnvKey([]string{"PATH=/bin", "ENVO_TOKEN=secret", "ENVO_TOKEN_BACKUP=keep"}, "ENVO_TOKEN")
	if len(got) != 2 || got[0] != "PATH=/bin" || got[1] != "ENVO_TOKEN_BACKUP=keep" {
		t.Fatalf("withoutEnvKey() = %v", got)
	}
}
