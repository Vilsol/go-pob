package difftest

import "testing"

func TestNormaliseReadsCRLFAsLF(t *testing.T) {
	r := result{stdout: []byte("ROOT /x/sb\r\nok\r\n"), stderr: []byte("e\r\n")}.normalise("/x/sb")
	if string(r.stdout) != "ROOT "+rootToken+"\nok\n" || string(r.stderr) != "e\n" {
		t.Fatalf("stdout %q stderr %q", r.stdout, r.stderr)
	}
}
