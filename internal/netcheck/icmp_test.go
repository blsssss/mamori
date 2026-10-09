package netcheck

import (
	"encoding/binary"
	"errors"
	"net/netip"
	"syscall"
	"testing"
)

// echoReplyBytes builds an ICMP_ECHO_REPLY as Windows lays it out for a process with the given
// pointer size. Pointers, padding and unused fields are filled with 0xEE so a wrong offset shows up
func echoReplyBytes(ptrSize int, addr [4]byte, status, rtt uint32, ttl uint8) []byte {
	b := make([]byte, echoReplySize(ptrSize)+32)
	for i := range b {
		b[i] = 0xEE
	}
	copy(b[0:4], addr[:])
	binary.LittleEndian.PutUint32(b[4:8], status)
	binary.LittleEndian.PutUint32(b[8:12], rtt)
	binary.LittleEndian.PutUint16(b[12:14], 32)
	// Options.Ttl follows the Data pointer: offset 20 on x86, 24 on x64 and arm64
	b[map[int]int{4: 20, 8: 24}[ptrSize]] = ttl
	return b
}

func TestEchoReplySize(t *testing.T) {
	// sizeof(ICMP_ECHO_REPLY) on x86 and on x64/arm64
	if got := echoReplySize(4); got != 28 {
		t.Errorf("32-bit size = %d, want 28", got)
	}
	if got := echoReplySize(8); got != 40 {
		t.Errorf("64-bit size = %d, want 40", got)
	}
}

func TestDecodeEchoReply(t *testing.T) {
	cases := []struct {
		name    string
		ptrSize int
		addr    [4]byte
		status  uint32
		rtt     uint32
		ttl     uint8
	}{
		{"64-bit reply", 8, [4]byte{1, 1, 1, 1}, ipSuccess, 23, 57},
		{"32-bit reply", 4, [4]byte{8, 8, 8, 8}, ipSuccess, 140, 116},
		{"64-bit unreachable", 8, [4]byte{192, 168, 0, 1}, 11003, 0, 64},
		{"32-bit ttl expired", 4, [4]byte{10, 0, 0, 1}, 11013, 3, 255},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := decodeEchoReply(echoReplyBytes(c.ptrSize, c.addr, c.status, c.rtt, c.ttl), c.ptrSize)
			if err != nil {
				t.Fatal(err)
			}
			want := echoReply{addr: netip.AddrFrom4(c.addr), status: c.status, rtt: c.rtt, ttl: c.ttl}
			if r != want {
				t.Errorf("got %+v, want %+v", r, want)
			}
		})
	}
}

func TestDecodeEchoReplyShort(t *testing.T) {
	for _, ptrSize := range []int{4, 8} {
		if _, err := decodeEchoReply(make([]byte, echoReplySize(ptrSize)-1), ptrSize); err == nil {
			t.Errorf("ptrSize %d: no error for a short buffer", ptrSize)
		}
	}
}

func TestEchoReplyErr(t *testing.T) {
	target := netip.MustParseAddr("1.1.1.1")
	ok := echoReply{addr: target, status: ipSuccess, ttl: 57}
	if err := ok.err(target); err != nil {
		t.Errorf("success reply: %v", err)
	}
	unreachable := echoReply{addr: netip.MustParseAddr("192.168.0.1"), status: 11003}
	err := unreachable.err(target)
	if err == nil || err.Error() != "IP_DEST_HOST_UNREACHABLE (11003) from 192.168.0.1" {
		t.Errorf("unreachable reply: %v", err)
	}
	if !errors.Is(err, ipStatus(11003)) {
		t.Errorf("%v does not wrap the status", err)
	}
	// an unfilled reply buffer decodes as a success from 0.0.0.0
	unfilled, err := decodeEchoReply(make([]byte, echoReplySize(8)), 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := unfilled.err(target); err == nil || err.Error() != "echo reply from 0.0.0.0" {
		t.Errorf("unfilled buffer: %v", err)
	}
}

func TestEchoReplyLocal(t *testing.T) {
	cases := []struct {
		ttl  uint8
		want bool
	}{
		{255, true},
		{254, false},
		{128, false},
		{57, false},
		{1, false},
	}
	for _, c := range cases {
		if got := (echoReply{ttl: c.ttl}).local(); got != c.want {
			t.Errorf("ttl %d: local = %t, want %t", c.ttl, got, c.want)
		}
	}
}

func TestIPStatusText(t *testing.T) {
	cases := []struct {
		s    ipStatus
		want string
	}{
		{11010, "IP_REQ_TIMED_OUT (11010)"},
		{11001, "IP_BUF_TOO_SMALL (11001)"},
		{11050, "IP_GENERAL_FAILURE (11050)"},
		{11040, "IP_STATUS 11040"},
	}
	for _, c := range cases {
		if got := c.s.Error(); got != c.want {
			t.Errorf("%d: %q, want %q", uint32(c.s), got, c.want)
		}
	}
}

func TestICMPError(t *testing.T) {
	cases := []struct {
		err  error
		want error
	}{
		{syscall.Errno(11010), ipStatus(11010)},
		{syscall.Errno(11003), ipStatus(11003)},
		{syscall.Errno(11050), ipStatus(11050)},
		{syscall.Errno(87), syscall.Errno(87)},
		{syscall.Errno(11051), syscall.Errno(11051)},
	}
	for _, c := range cases {
		got := icmpError(c.err)
		if !errors.Is(got, c.want) {
			t.Errorf("icmpError(%d) = %v, want %v", c.err, got, c.want)
		}
		var status ipStatus
		if !errors.As(c.want, &status) && got.Error() != "IcmpSendEcho2: "+c.err.Error() {
			t.Errorf("icmpError(%d) = %q, want the call name in front", c.err, got)
		}
	}
}
