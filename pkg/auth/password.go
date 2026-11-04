package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// PasswordConfig is a password configuration for argon2.
type PasswordConfig struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func NewPasswordConfig(time int, memory int, threads int, keyLen int) *PasswordConfig {
	return &PasswordConfig{uint32(time), uint32(memory), uint8(threads), uint32(keyLen)}
}

// One hash runs at a time. Each takes tens of milliseconds and 64 MB, and the
// API hosts have two vCPUs, so running more at once would only slow down
// everything else on the host.
var hasher = make(chan struct{}, 1)

// hasherWait is how long a password check waits behind others before giving
// up. Without a limit, a burst of login attempts queues up in front of every
// real login.
const hasherWait = 2 * time.Second

// ErrHasherBusy means too many password checks were queued ahead of this one.
var ErrHasherBusy = errors.New("the server is busy, please try again in a moment")

func argonKey(password, salt []byte, c *PasswordConfig) ([]byte, error) {
	timer := time.NewTimer(hasherWait)
	defer timer.Stop()
	select {
	case hasher <- struct{}{}:
	case <-timer.C:
		return nil, ErrHasherBusy
	}
	defer func() { <-hasher }()
	return argon2.IDKey(password, salt, c.time, c.memory, c.threads, c.keyLen), nil
}

// GeneratePassword is used to generate a new password hash for storing and
// comparing at a later date.
func GeneratePassword(c *PasswordConfig, password string) (string, error) {

	// Generate a Salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash, err := argonKey([]byte(password), salt, c)
	if err != nil {
		return "", err
	}

	// Base64 encode the salt and hashed password.
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	format := "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"
	full := fmt.Sprintf(format, argon2.Version, c.memory, c.time, c.threads, b64Salt, b64Hash)
	return full, nil
}

// ComparePassword is used to compare a user-inputted password to a hash to see
// if the password matches or not.
func ComparePassword(password, hash string) (bool, error) {
	parts := strings.Split(hash, "$")

	c := &PasswordConfig{}
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &c.memory, &c.time, &c.threads)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	c.keyLen = uint32(len(decodedHash))

	comparisonHash, err := argonKey([]byte(password), salt, c)
	if err != nil {
		return false, err
	}

	return (subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1), nil
}

// NeedsRehash reports whether a stored hash was made with settings other than
// c, so it can be replaced after the next successful login.
func NeedsRehash(c *PasswordConfig, hash string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}
	return parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", c.memory, c.time, c.threads)
}
