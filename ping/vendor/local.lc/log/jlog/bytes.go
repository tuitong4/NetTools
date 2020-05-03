package jlog

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	Byte     = 1.0
	Kilobyte = 1024 * Byte
	Megabyte = 1024 * Kilobyte
	Gigabyte = 1024 * Megabyte
	Terabyte = 1024 * Gigabyte
)

var bytesPattern *regexp.Regexp = regexp.MustCompile(`(?i)^(-?\d+)([KMGT]B?|B)$`)

var invalidByteQuantityError = errors.New("Byte quantity must be a positive integer with a unit of measurement like M, MB, G, or GB")

// ByteSize returns a human readable byte string, of the format 10M, 12.5K, etc.  The following units are available:
//	T Terabyte
//	G Gigabyte
//	M Megabyte
//	K Kilobyte
// the unit that would result in printing the smallest whole number is always chosen
func Itob(b uint64) string {
	u := ""
	v := float32(b)

	switch {
	case b >= Terabyte:
		u = "T"
		v = v / Terabyte
	case b >= Gigabyte:
		u = "G"
		v = v / Gigabyte
	case b >= Megabyte:
		u = "M"
		v = v / Megabyte
	case b >= Kilobyte:
		u = "K"
		v = v / Kilobyte
	case b == 0:
		return "0"
	}

	s := fmt.Sprintf("%.1f", v)
	s = strings.TrimSuffix(s, ".0")
	s = fmt.Sprintf("%s%s", s, u)
	return s
}

// ToMegabyte parses a string formatted by ByteSize as megabytes
func Btoi(s string) (uint64, error) {
	var i uint64
	parts := bytesPattern.FindStringSubmatch(strings.TrimSpace(s))
	if len(parts) == 0 {
		p, _ := strconv.Atoi(s)
		i = uint64(p)
		return i, nil
	}
	if len(parts) < 3 {
		return 0, invalidByteQuantityError
	}

	v, err := strconv.ParseUint(parts[1], 10, 0)
	if err != nil || v < 1 {
		return 0, invalidByteQuantityError
	}

	u := strings.ToUpper(parts[2])
	switch u[:1] {
	case "T":
		i = v * Terabyte
	case "G":
		i = v * Gigabyte
	case "M":
		i = v * Megabyte
	case "K":
		i = v * Kilobyte
	}
	return i, nil
}
