package platform

import "strings"

type Os string
type Arch string
type Variant string

// Canonical? OS and Arch names.
// See:
// - https://go.dev/doc/install/source#environment
// - https://github.com/opencontainers/image-spec/blob/main/image-index.md#platform-variants
const (
	OsLinux   Os = "linux"
	OsWindows Os = "windows"
	OsDarwin  Os = "darwin"

	ArchArm   Arch = "arm"
	ArchArm64 Arch = "arm64"
	ArchAmd64 Arch = "amd64"

	VariantArmV6   Variant = "v6"
	VariantArmV7   Variant = "v7"
	VariantArmV8   Variant = "v8"
	VariantArmV8_1 Variant = "v8.1"
)

type Platform string

func (p Platform) String() string {
	return string(p)
}

func (p Platform) Split() (os string, arch string, variant string) {
	es := strings.SplitN(string(p), "/", 3)
	os = es[0]
	if len(es) > 1 {
		arch = es[1]
	}
	if len(es) > 2 {
		variant = es[2]
	}
	return
}

func (p Platform) Os() string {
	os, _, _ := p.Split()
	return os
}

func (p Platform) Arch() string {
	_, arch, _ := p.Split()
	return arch
}

func (p Platform) Variant() string {
	_, _, variant := p.Split()
	return variant
}
