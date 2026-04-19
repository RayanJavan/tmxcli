package cli

import "testing"

func TestParseStart(t *testing.T) {
	opts, err := ParseStart([]string{"/tmp/out.log"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if opts.LogPath != "/tmp/out.log" {
		t.Fatalf("bad path: %s", opts.LogPath)
	}
}

func TestParseProcAndTarget(t *testing.T) {
	opts, err := ParseProcAndTarget("attach", []string{"-p", "3", "-t", "1"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if opts.ProcessID == nil || *opts.ProcessID != 3 {
		t.Fatalf("bad process")
	}
	if opts.Target != "1" {
		t.Fatalf("bad target")
	}
}
