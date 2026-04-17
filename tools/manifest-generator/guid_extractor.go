package main

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"strings"
)

// ProductCodeFor computes the Inno Setup ProductCode for a Mendix Studio Pro version.
// Inno Setup uses SHA1(UTF-16LE(AppId)) as the registry key, where
// AppId = "Mendix Studio Pro " + full version string.
func ProductCodeFor(versionFull string) string {
	appID := "Mendix Studio Pro " + versionFull
	utf16le := utf16LEEncode(appID)
	hash := sha1.Sum(utf16le)
	return fmt.Sprintf("{%x_is1}", hash)
}

func utf16LEEncode(s string) []byte {
	buf := make([]byte, len(s)*2)
	for i, r := range s {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(r))
	}
	return buf
}

func GUIDPlaceholder(version string) string {
	return fmt.Sprintf("{MENDIX-STUDIO-PRO-%s-PLACEHOLDER}", strings.ReplaceAll(version, ".", "-"))
}
