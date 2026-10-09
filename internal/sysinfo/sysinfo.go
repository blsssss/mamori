// Package sysinfo describes the machine the checks run on.
package sysinfo

type Info struct {
	Product string `json:"product"`
	Version string `json:"version"`
	Build   string `json:"build"`
	Arch    string `json:"arch"`
	Host    string `json:"host"`
	User    string `json:"user"`
}
