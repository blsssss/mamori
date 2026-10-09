package check

import "testing"

func TestWorse(t *testing.T) {
	cases := []struct{ a, b, want Status }{
		{Pass, Pass, Pass},
		{Pass, Skip, Skip},
		{Skip, Warn, Warn},
		{Fail, Warn, Fail},
		{Warn, Error, Error},
		{Error, Pass, Error},
	}
	for _, c := range cases {
		if got := Worse(c.a, c.b); got != c.want {
			t.Errorf("Worse(%s, %s) = %s, want %s", c.a, c.b, got, c.want)
		}
	}
}

func TestRecorder(t *testing.T) {
	var emitted []Finding
	r := Start(Internet, func(id ID, f Finding) {
		if id != Internet {
			t.Errorf("emitted for %s", id)
		}
		emitted = append(emitted, f)
	})
	r.Add(Pass, "net.dns.ok", "host", "example.org", "dangling")
	r.Add(Warn, "net.icmp.timeout")
	if r.Worst() != Warn {
		t.Fatalf("worst = %s", r.Worst())
	}
	res := r.Finish(Pass, "net.verdict.online")
	if res.Status != Pass || res.Code != "net.verdict.online" || res.Check != Internet {
		t.Fatalf("result = %+v", res)
	}
	if len(res.Findings) != 2 || len(emitted) != 2 {
		t.Fatalf("findings %d, emitted %d", len(res.Findings), len(emitted))
	}
	if p := res.Findings[0].Params; len(p) != 1 || p["host"] != "example.org" {
		t.Fatalf("params = %v", p)
	}
	if res.Findings[1].Params != nil {
		t.Fatalf("empty params should be nil, got %v", res.Findings[1].Params)
	}
}
