package apg

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const (
	// 7 bits to represent a letter index
	letterIdxBits = 7
	// All 1-bits, as many as letterIdxBits
	letterIdxMask = 1<<letterIdxBits - 1
	// # of letter indices fitting in 63 bits)
	letterIdxMax = 63 / letterIdxBits
)

// maxInt32 is the maximum positive value for a int32 number type
const maxInt32 = 2147483647

var (
	// ErrInvalidLength is returned if the provided maximum number is equal or less than zero
	ErrInvalidLength = errors.New("provided length value cannot be zero or less")
	// ErrLengthMismatch is returned if the number of generated bytes does not match the expected length
	ErrLengthMismatch = errors.New("number of generated random bytes does not match the expected length")
	// ErrInvalidCharRange is returned if the given range of characters is not valid
	ErrInvalidCharRange = errors.New("provided character range is not valid or empty")
)

// CoinFlip performs a simple coinflip based on the rand library and returns 1 or 0
func (g *Generator) CoinFlip() int64 {
	cf, _ := g.RandNum(2)
	return cf
}

// CoinFlipBool performs a simple coinflip based on the rand library and returns true or false
func (g *Generator) CoinFlipBool() bool {
	return g.CoinFlip() == 1
}

// Generate generates a password based on all the different config flags and returns
// it as string type. If the generation fails, an error will be thrown
func (g *Generator) Generate() (string, error) {
	switch g.config.Algorithm {
	case AlgoCoinFlip:
		return g.generateCoinFlip()
	case AlgoRandom:
		return g.generateRandom()
	case AlgoUnsupported:
		return "", fmt.Errorf("unsupported algorithm")
	}
	return "", nil
}

// GetPasswordLength returns the password length based on the given config
// parameters
func (g *Generator) GetPasswordLength() (int64, error) {
	if g.config.FixedLength > 0 {
		return g.config.FixedLength, nil
	}
	minLength := g.config.MinLength
	maxLength := g.config.MaxLength
	if minLength > maxLength {
		maxLength = minLength
	}
	diff := maxLength - minLength + 1
	randNum, err := g.RandNum(diff)
	if err != nil {
		return 0, err
	}
	length := minLength + randNum
	if length <= 0 {
		return 1, nil
	}
	return length, nil
}

// RandomBytes returns a byte slice of random bytes with given length that got generated by
// the crypto/rand generator
func (g *Generator) RandomBytes(length int64) ([]byte, error) {
	if length < 1 {
		return nil, ErrInvalidLength
	}
	bytes := make([]byte, length)
	numBytes, err := rand.Read(bytes)
	if int64(numBytes) != length {
		return nil, ErrLengthMismatch
	}
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// RandNum generates a random, non-negative number with given maximum value
func (g *Generator) RandNum(max int64) (int64, error) {
	if max < 1 {
		return 0, ErrInvalidLength
	}
	max64 := big.NewInt(max)
	randNum, err := rand.Int(rand.Reader, max64)
	if err != nil {
		return 0, fmt.Errorf("random number generation failed: %w", err)
	}
	return randNum.Int64(), nil
}

// RandomStringFromCharRange returns a random string of length l based of the range of characters given.
// The method makes use of the crypto/random package and therfore is
// cryptographically secure
func (g *Generator) RandomStringFromCharRange(length int64, charRange string) (string, error) {
	if length < 1 {
		return "", ErrInvalidLength
	}
	if len(charRange) < 1 {
		return "", ErrInvalidCharRange
	}
	rs := strings.Builder{}

	// As long as the length is smaller than the max. int32 value let's grow
	// the string builder to the actual size, so we need less allocations
	if length <= maxInt32 {
		rs.Grow(int(length))
	}

	charRangeLength := len(charRange)

	rp := make([]byte, 8)
	_, err := rand.Read(rp)
	if err != nil {
		return rs.String(), err
	}
	for i, c, r := length-1, binary.BigEndian.Uint64(rp), letterIdxMax; i >= 0; {
		if r == 0 {
			_, err = rand.Read(rp)
			if err != nil {
				return rs.String(), err
			}
			c, r = binary.BigEndian.Uint64(rp), letterIdxMax
		}
		if idx := int(c & letterIdxMask); idx < charRangeLength {
			rs.WriteByte(charRange[idx])
			i--
		}
		c >>= letterIdxBits
		r--
	}

	return rs.String(), nil
}

// GetCharRangeFromConfig checks the Mode from the Config and returns a
// list of all possible characters that are supported by these Mode
func (g *Generator) GetCharRangeFromConfig() string {
	cr := strings.Builder{}
	if MaskHasMode(g.config.Mode, ModeLowerCase) {
		switch MaskHasMode(g.config.Mode, ModeHumanReadable) {
		case true:
			cr.WriteString(CharRangeAlphaLowerHuman)
		default:
			cr.WriteString(CharRangeAlphaLower)
		}
	}
	if MaskHasMode(g.config.Mode, ModeNumeric) {
		switch MaskHasMode(g.config.Mode, ModeHumanReadable) {
		case true:
			cr.WriteString(CharRangeNumericHuman)
		default:
			cr.WriteString(CharRangeNumeric)
		}
	}
	if MaskHasMode(g.config.Mode, ModeSpecial) {
		switch MaskHasMode(g.config.Mode, ModeHumanReadable) {
		case true:
			cr.WriteString(CharRangeSpecialHuman)
		default:
			cr.WriteString(CharRangeSpecial)
		}
	}
	if MaskHasMode(g.config.Mode, ModeUpperCase) {
		switch MaskHasMode(g.config.Mode, ModeHumanReadable) {
		case true:
			cr.WriteString(CharRangeAlphaUpperHuman)
		default:
			cr.WriteString(CharRangeAlphaUpper)
		}
	}
	return cr.String()
}

func (g *Generator) checkMinimumRequirements(pw string) bool {
	ok := true
	if g.config.MinLowerCase > 0 {
		var cr string
		switch MaskHasMode(g.config.Mode, ModeHumanReadable) {
		case true:
			cr = CharRangeAlphaLowerHuman
		default:
			cr = CharRangeAlphaLower
		}

		m := 0
		for _, c := range cr {
			m += strings.Count(pw, string(c))
		}
		if int64(m) < g.config.MinLowerCase {
			ok = false
		}
	}
	if g.config.MinNumeric > 0 {
		var cr string
		switch MaskHasMode(g.config.Mode, ModeHumanReadable) {
		case true:
			cr = CharRangeNumericHuman
		default:
			cr = CharRangeNumeric
		}

		m := 0
		for _, c := range cr {
			m += strings.Count(pw, string(c))
		}
		if int64(m) < g.config.MinNumeric {
			ok = false
		}
	}
	if g.config.MinSpecial > 0 {
		var cr string
		switch MaskHasMode(g.config.Mode, ModeHumanReadable) {
		case true:
			cr = CharRangeSpecialHuman
		default:
			cr = CharRangeSpecial
		}

		m := 0
		for _, c := range cr {
			m += strings.Count(pw, string(c))
		}
		if int64(m) < g.config.MinSpecial {
			ok = false
		}
	}
	if g.config.MinUpperCase > 0 {
		var cr string
		switch MaskHasMode(g.config.Mode, ModeHumanReadable) {
		case true:
			cr = CharRangeAlphaUpperHuman
		default:
			cr = CharRangeAlphaUpper
		}

		m := 0
		for _, c := range cr {
			m += strings.Count(pw, string(c))
		}
		if int64(m) < g.config.MinUpperCase {
			ok = false
		}
	}
	return ok
}

// generateCoinFlip is executed when Generate() is called with Algorithm set
// to AlgoCoinFlip
func (g *Generator) generateCoinFlip() (string, error) {
	switch g.CoinFlipBool() {
	case true:
		return "Heads", nil
	default:
		return "Tails", nil
	}
}

// generateRandom is executed when Generate() is called with Algorithm set
// to AlgoRandmom
func (g *Generator) generateRandom() (string, error) {
	l, err := g.GetPasswordLength()
	if err != nil {
		return "", fmt.Errorf("failed to calculate password length: %w", err)
	}
	cr := g.GetCharRangeFromConfig()
	var pw string
	var ok bool
	for !ok {
		pw, err = g.RandomStringFromCharRange(l, cr)
		if err != nil {
			return "", err
		}
		ok = g.checkMinimumRequirements(pw)
	}

	return pw, nil
}
