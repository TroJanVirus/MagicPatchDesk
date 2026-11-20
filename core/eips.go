package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const encryptedIPSMagic = "JMIPSE1!"

func pbkdf2sha256(password, salt []byte, rounds int, keyLen int) []byte {
	out := make([]byte, 0, keyLen)
	for block := uint32(1); len(out) < keyLen; block++ {
		m := hmac.New(sha256.New, password)
		m.Write(salt)
		var num [4]byte
		binary.BigEndian.PutUint32(num[:], block)
		m.Write(num[:])
		u := m.Sum(nil)
		t := append([]byte{}, u...)
		for i := 1; i < rounds; i++ {
			m = hmac.New(sha256.New, password)
			m.Write(u)
			u = m.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}
func EncryptIPS(plain []byte, password string) ([]byte, error) {
	if len(password) < 8 {
		return nil, errors.New("passphrase needs at least 8 characters")
	}
	if _, err := DecodeIPS(plain); err != nil {
		return nil, fmt.Errorf("not a standard IPS patch: %w", err)
	}
	salt := make([]byte, 16)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	key := pbkdf2sha256([]byte(password), salt, 210000, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	b := append([]byte(encryptedIPSMagic), salt...)
	b = append(b, nonce...)
	return gcm.Seal(b, nonce, plain, nil), nil
}
func DecryptIPS(encrypted []byte, password string) ([]byte, error) {
	if len(encrypted) < 52 || string(encrypted[:8]) != encryptedIPSMagic {
		return nil, errors.New("not an encrypted Magic IPS patch (.eips)")
	}
	salt, nonce, body := encrypted[8:24], encrypted[24:36], encrypted[36:]
	key := pbkdf2sha256([]byte(password), salt, 210000, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	out, err := gcm.Open(nil, nonce, body, nil)
	if err != nil {
		return nil, errors.New("incorrect password or damaged encrypted IPS file")
	}
	return out, nil
}
