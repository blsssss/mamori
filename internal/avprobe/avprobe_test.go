package avprobe

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"syscall"
	"testing"

	"github.com/blsssss/mamori/internal/check"
)

const amsiSHA256 = "50a3a914f7ce11e2ec39815e1d3880e2d38a7917645c60b87224094d5267501c"

// only hashes are compared and printed, the decoded samples stay in memory
func TestSamples(t *testing.T) {
	cases := []struct {
		name, enc, want string
		size            int
	}{
		{"eicar", eicarEnc, eicarSHA256, 68},
		{"amsi", amsiEnc, amsiSHA256, 54},
	}
	for _, c := range cases {
		b := reveal(c.enc)
		sum := sha256.Sum256(b)
		if got := hex.EncodeToString(sum[:]); got != c.want || len(b) != c.size {
			t.Errorf("%s: sha256 %s, %d bytes, want %s, %d bytes", c.name, got, len(b), c.want, c.size)
		}
	}
}

func pathErr(errno syscall.Errno) error {
	return &fs.PathError{Op: "open", Path: `C:\tmp\eicar-test.txt`, Err: errno}
}

func TestBlockedByAV(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain", errors.New("disk full"), false},
		{"virus infected", pathErr(225), true},
		{"virus deleted", pathErr(226), true},
		{"access denied", pathErr(5), true},
		{"wrapped", fmt.Errorf("write: %w", pathErr(225)), true},
		{"sharing violation", pathErr(32), false},
		{"file not found", pathErr(2), false},
	}
	for _, c := range cases {
		if got := blockedByAV(c.err); got != c.want {
			t.Errorf("%s: blockedByAV = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestReadState(t *testing.T) {
	cases := []struct {
		name    string
		content []byte
		err     error
		want    fileState
	}{
		{"intact", reveal(eicarEnc), nil, fileIntact},
		{"emptied", []byte{}, nil, fileAltered},
		{"replaced", []byte("removed by antivirus"), nil, fileAltered},
		{"removed", nil, &fs.PathError{Op: "open", Err: fs.ErrNotExist}, fileRemoved},
		{"virus infected", nil, pathErr(225), fileBlocked},
		{"access denied", nil, pathErr(5), fileBlocked},
		{"sharing violation", nil, pathErr(32), fileUnreadable},
		{"other", nil, errors.New("unexpected"), fileUnreadable},
	}
	for _, c := range cases {
		if got := readState(c.content, c.err); got != c.want {
			t.Errorf("%s: readState = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestClassifyAMSI(t *testing.T) {
	cases := []struct {
		result uint32
		want   outcome
	}{
		{0, notDetected},     // AMSI_RESULT_CLEAN
		{1, notDetected},     // AMSI_RESULT_NOT_DETECTED
		{16384, notDetected}, // AMSI_RESULT_BLOCKED_BY_ADMIN_START
		{20479, notDetected}, // AMSI_RESULT_BLOCKED_BY_ADMIN_END
		{32767, notDetected},
		{32768, detected}, // AMSI_RESULT_DETECTED
		{40000, detected},
	}
	for _, c := range cases {
		if got := classifyAMSI(c.result); got != c.want {
			t.Errorf("classifyAMSI(%d) = %d, want %d", c.result, got, c.want)
		}
	}
}

func TestVerdict(t *testing.T) {
	cases := []struct {
		eicar, amsi outcome
		status      check.Status
		code        string
	}{
		{detected, detected, check.Pass, "av.verdict.works"},
		{detected, notDetected, check.Warn, "av.verdict.partial"},
		{detected, notRun, check.Warn, "av.verdict.partial"},
		{notDetected, detected, check.Warn, "av.verdict.partial"},
		{notDetected, notDetected, check.Fail, "av.verdict.not_working"},
		{notDetected, notRun, check.Fail, "av.verdict.not_working"},
		{notRun, detected, check.Error, "av.verdict.error"},
		{notRun, notDetected, check.Error, "av.verdict.error"},
		{notRun, notRun, check.Error, "av.verdict.error"},
	}
	for _, c := range cases {
		status, code := verdict(c.eicar, c.amsi)
		if status != c.status || code != c.code {
			t.Errorf("verdict(%d, %d) = %s %s, want %s %s", c.eicar, c.amsi, status, code, c.status, c.code)
		}
	}
}
