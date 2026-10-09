package netcheck

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"syscall"
)

const (
	ipSuccess        = 0
	ipStatusBase     = 11000
	ipGeneralFailure = 11050
)

type echoReply struct {
	addr   netip.Addr
	status uint32
	rtt    uint32
	ttl    uint8
}

// err is the error of a reply to an echo request sent to target
func (r echoReply) err(target netip.Addr) error {
	if r.status != ipSuccess {
		return fmt.Errorf("%w from %s", ipStatus(r.status), r.addr)
	}
	// an echo reply comes from the pinged address, any other one, 0.0.0.0 of an unfilled buffer
	// included, does not answer this request
	if r.addr != target {
		return fmt.Errorf("echo reply from %s", r.addr)
	}
	return nil
}

// local tells a reply that crossed no router: each router decrements the ttl, so a public address
// that answers with the maximum ttl was answered for by this machine, as the TUN adapter of a VPN or
// proxy client does, or by a device on the same link
func (r echoReply) local() bool {
	return r.ttl == math.MaxUint8
}

// echoReplySize is sizeof(ICMP_ECHO_REPLY): 16 bytes of fixed fields, the Data pointer and
// IP_OPTION_INFORMATION, whose 4 bytes are padded to the pointer size before OptionsData
func echoReplySize(ptrSize int) int {
	return 16 + 3*ptrSize
}

// decodeEchoReply reads the first ICMP_ECHO_REPLY (ipexport.h) from the reply buffer of
// IcmpSendEcho2. Pointer fields make the layout depend on the bitness of the process:
//
//	0      Address        IPAddr, network byte order
//	4      Status         IP_STATUS
//	8      RoundTripTime  milliseconds
//	12     DataSize, Reserved
//	16     Data           pointer
//	16+p   Options.Ttl, Tos, Flags, OptionsSize, OptionsData pointer
func decodeEchoReply(b []byte, ptrSize int) (echoReply, error) {
	if len(b) < echoReplySize(ptrSize) {
		return echoReply{}, fmt.Errorf("reply of %d bytes is shorter than ICMP_ECHO_REPLY", len(b))
	}
	return echoReply{
		addr:   netip.AddrFrom4([4]byte(b[0:4])),
		status: binary.LittleEndian.Uint32(b[4:8]),
		rtt:    binary.LittleEndian.Uint32(b[8:12]),
		ttl:    b[16+ptrSize],
	}, nil
}

// ipStatus is an IP_STATUS code. It has its own text because 11001-11004 are also winsock codes,
// and syscall.Errno would describe them as DNS errors
type ipStatus uint32

var ipStatusNames = map[ipStatus]string{
	11001: "IP_BUF_TOO_SMALL",
	11002: "IP_DEST_NET_UNREACHABLE",
	11003: "IP_DEST_HOST_UNREACHABLE",
	11004: "IP_DEST_PROT_UNREACHABLE",
	11005: "IP_DEST_PORT_UNREACHABLE",
	11006: "IP_NO_RESOURCES",
	11007: "IP_BAD_OPTION",
	11008: "IP_HW_ERROR",
	11009: "IP_PACKET_TOO_BIG",
	11010: "IP_REQ_TIMED_OUT",
	11011: "IP_BAD_REQ",
	11012: "IP_BAD_ROUTE",
	11013: "IP_TTL_EXPIRED_TRANSIT",
	11014: "IP_TTL_EXPIRED_REASSEM",
	11015: "IP_PARAM_PROB",
	11016: "IP_SOURCE_QUENCH",
	11017: "IP_OPTION_TOO_BIG",
	11018: "IP_BAD_DESTINATION",
	11050: "IP_GENERAL_FAILURE",
}

func (s ipStatus) Error() string {
	if name, ok := ipStatusNames[s]; ok {
		return fmt.Sprintf("%s (%d)", name, uint32(s))
	}
	return fmt.Sprintf("IP_STATUS %d", uint32(s))
}

// icmpError converts the last error of a failed IcmpSendEcho2, which is an IP_STATUS code when the
// request went out and got no reply
func icmpError(err error) error {
	if errno, ok := errors.AsType[syscall.Errno](err); ok && errno >= ipStatusBase && errno <= ipGeneralFailure {
		return ipStatus(errno)
	}
	return fmt.Errorf("IcmpSendEcho2: %w", err)
}
