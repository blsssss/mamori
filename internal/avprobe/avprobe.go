// Package avprobe checks that the antivirus reacts to the EICAR test file and the AMSI test sample.
package avprobe

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io/fs"
	"syscall"

	"github.com/blsssss/mamori/internal/check"
)

// both samples are kept xor-ed and base64 encoded: in plain form a scanner would quarantine the
// source tree and the program itself
const (
	sampleKey = "mamori"
	eicarEnc  = "NVQiTiJMLSA9NEY1PTs1WkZBPT9EWDEqRFYQSzcgLiA/QiE9LC8pLiAtQCAjOzs/JDM4PF89KDI5QjQgISRMSzpCJUs="
	amsiEnc   = "LCw+JlI9CBIZTyEIABEBCkhJWgRaXRFaDgRAV0RYD0xZXEFQQFlaW0JEXQAOXkZRWQJcXEpf"
)

const eicarSHA256 = "275a021bbfb6489e54d471899f7db9d1663fc695ec2fe2a2c4538aabf651fd0f"

// AMSI_RESULT_DETECTED, AmsiResultIsMalware is a header macro and not an amsi.dll export
const amsiResultDetected = 32768

// win32 errors a file operation fails with when an antivirus refuses the content
const (
	errorAccessDenied  syscall.Errno = 5
	errorVirusInfected syscall.Errno = 225
	errorVirusDeleted  syscall.Errno = 226
)

type outcome int

const (
	notDetected outcome = iota
	detected
	notRun
)

type fileState int

const (
	fileIntact fileState = iota
	// the read failed for a reason that says nothing about the antivirus, e.g. the scanner still
	// holds the file, so the next poll decides
	fileUnreadable
	fileRemoved
	fileBlocked
	fileAltered
)

func reveal(enc string) []byte {
	b, _ := base64.StdEncoding.DecodeString(enc)
	for i := range b {
		b[i] ^= sampleKey[i%len(sampleKey)]
	}
	return b
}

func isEicar(b []byte) bool {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]) == eicarSHA256
}

func blockedByAV(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	return errno == errorAccessDenied || errno == errorVirusInfected || errno == errorVirusDeleted
}

// readState classifies one read of the written test file
func readState(content []byte, err error) fileState {
	switch {
	case err == nil && isEicar(content):
		return fileIntact
	case err == nil:
		return fileAltered
	case errors.Is(err, fs.ErrNotExist):
		return fileRemoved
	case blockedByAV(err):
		return fileBlocked
	}
	return fileUnreadable
}

func classifyAMSI(result uint32) outcome {
	if result >= amsiResultDetected {
		return detected
	}
	return notDetected
}

// verdict combines the EICAR and AMSI outcomes, notRun for AMSI means it was unavailable
func verdict(eicar, amsi outcome) (check.Status, string) {
	switch {
	case eicar == notRun:
		return check.Error, "av.verdict.error"
	case eicar == detected && amsi == detected:
		return check.Pass, "av.verdict.works"
	case eicar == detected || amsi == detected:
		return check.Warn, "av.verdict.partial"
	}
	return check.Fail, "av.verdict.not_working"
}
