package netcheck

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/blsssss/mamori/internal/check"
)

func Run(ctx context.Context, _ check.Options, emit check.Emit) check.Result {
	return run(ctx, emit, []probe{
		adapters(listAdapters),
		lookup("www.msftconnecttest.com"),
		ncsiLookup("dns.msftncsi.com", netip.MustParseAddr("131.107.255.255")),
		// three operators, as one of them may be filtered or throttled on the way
		ping(echo, netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("77.88.8.8")),
		dial(new(net.Dialer).DialContext, "1.1.1.1:443", "8.8.8.8:443", "77.88.8.8:443"),
		fetch("http://www.msftconnecttest.com/connecttest.txt"),
	})
}

func listAdapters() ([]adapter, error) {
	const flags = windows.GAA_FLAG_INCLUDE_GATEWAYS | windows.GAA_FLAG_SKIP_ANYCAST |
		windows.GAA_FLAG_SKIP_MULTICAST | windows.GAA_FLAG_SKIP_DNS_SERVER
	// 15 KB is the first buffer size microsoft recommends, it avoids a second call on most machines
	size := uint32(15 << 10)
	for {
		buf := make([]byte, size)
		first := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err := windows.GetAdaptersAddresses(windows.AF_INET, flags, 0, first, &size)
		if errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) && size > uint32(len(buf)) {
			continue
		}
		// ERROR_NO_DATA is the answer when no adapter has ipv4
		if errors.Is(err, windows.ERROR_NO_DATA) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("GetAdaptersAddresses: %w", err)
		}
		var list []adapter
		for aa := first; aa != nil; aa = aa.Next {
			a := adapter{
				name:     windows.UTF16PtrToString(aa.FriendlyName),
				up:       aa.OperStatus == windows.IfOperStatusUp,
				loopback: aa.IfType == windows.IF_TYPE_SOFTWARE_LOOPBACK,
			}
			for u := aa.FirstUnicastAddress; u != nil; u = u.Next {
				if ip, ok := netip.AddrFromSlice(u.Address.IP()); ok {
					a.ipv4 = append(a.ipv4, ip.Unmap())
				}
			}
			for g := aa.FirstGatewayAddress; g != nil; g = g.Next {
				if ip, ok := netip.AddrFromSlice(g.Address.IP()); ok {
					a.gateways = append(a.gateways, ip.Unmap())
				}
			}
			list = append(list, a)
		}
		return list, nil
	}
}

var (
	iphlpapi            = windows.NewLazySystemDLL("iphlpapi.dll")
	procIcmpCreateFile  = iphlpapi.NewProc("IcmpCreateFile")
	procIcmpSendEcho2   = iphlpapi.NewProc("IcmpSendEcho2")
	procIcmpCloseHandle = iphlpapi.NewProc("IcmpCloseHandle")
)

// the 32 bytes that ping.exe sends, so the request looks like an ordinary ping
var echoPayload = []byte("abcdefghijklmnopqrstuvwabcdefghi")

// echo sends one echo request through the ICMP helper of iphlpapi. Unlike a raw socket it needs no
// administrator rights. The call is synchronous and bounded by timeout
func echo(target netip.Addr, timeout time.Duration) (echoReply, error) {
	if !target.Is4() {
		return echoReply{}, fmt.Errorf("%s is not an ipv4 address", target)
	}
	// LazyProc.Call panics when the function is missing, Find turns that into an error
	for _, p := range []*windows.LazyProc{procIcmpCreateFile, procIcmpSendEcho2, procIcmpCloseHandle} {
		if err := p.Find(); err != nil {
			return echoReply{}, err
		}
	}
	h, _, err := procIcmpCreateFile.Call()
	if windows.Handle(h) == windows.InvalidHandle {
		return echoReply{}, fmt.Errorf("IcmpCreateFile: %w", err)
	}
	defer func() { _, _, _ = procIcmpCloseHandle.Call(h) }()

	ptrSize := int(unsafe.Sizeof(uintptr(0)))
	// one reply with the echoed data, plus the 8 byte ICMP error and the IO_STATUS_BLOCK that the
	// IcmpSendEcho2 documentation asks room for
	reply := make([]byte, echoReplySize(ptrSize)+len(echoPayload)+8+2*ptrSize)
	// IPAddr holds the address bytes in network order, read as a little-endian dword
	dst := target.As4()
	n, _, err := procIcmpSendEcho2.Call(h, 0, 0, 0,
		uintptr(binary.LittleEndian.Uint32(dst[:])),
		uintptr(unsafe.Pointer(&echoPayload[0])), uintptr(len(echoPayload)),
		0,
		uintptr(unsafe.Pointer(&reply[0])), uintptr(len(reply)),
		uintptr(max(timeout.Milliseconds(), 1)))
	// the result is a DWORD, the upper half of the register is not part of it
	if uint32(n) == 0 {
		return echoReply{}, icmpError(err)
	}
	return decodeEchoReply(reply, ptrSize)
}
