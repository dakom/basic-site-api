package cipher

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/scrypt"
)

const CBC_IV_SIZE uint64 = 16

const GCM_IV_SIZE uint64 = 12
const GCM_ADATA_SIZE uint64 = 16

const SALT_LENGTH = 16
const PASSWORD_KEY_LENGTH = 32

func RandomBytes(byteLength uint64) ([]byte, error) {
	bytes := make([]byte, byteLength)

	_, err := rand.Read(bytes)

	return bytes, err
}

//nil salt is for new salt
func NewPWHash(plaintext string, salt []byte) (string, error) {
	var err error

	if salt == nil {
		salt, err = RandomBytes(SALT_LENGTH)
		if err != nil {
			return "", err
		}
	} else if len(salt) != SALT_LENGTH {
		return "", fmt.Errorf("BAD SALT LENGTH!")
	}

	// Generate key
	key, err := scrypt.Key([]byte(plaintext), salt, 16384, 8, 1, PASSWORD_KEY_LENGTH)
	if err != nil {
		return "", fmt.Errorf("Error in deriving passphrase: %s\n", err)
	}

	// Appending the salt
	key = append(salt, key...)

	return hex.EncodeToString(key), nil

}

func ComparePWHash(plaintext string, hashWithSalt string) bool {
	//Get salt from provided hash
	if len(hashWithSalt) <= SALT_LENGTH {
		return false
	}

	hashWithSaltBytes, err := hex.DecodeString(hashWithSalt)
	if err != nil {
		return false
	}
	salt := hashWithSaltBytes[:SALT_LENGTH]

	testHashWithSalt, err := NewPWHash(plaintext, salt)
	if err != nil {
		return false
	}

	if subtle.ConstantTimeCompare([]byte(testHashWithSalt), []byte(hashWithSalt)) == 1 {
		return true
	} else {
		return false
	}
}

//Tortilla
func WrapTortilla(plaintext []byte, publicKey []byte, aesCharif []byte, aesTortilla []byte) ([]byte, error) {
	var nonce [24]byte

	if len(publicKey) < 32 {
		return nil, fmt.Errorf("Public key must be 32 bytes")
	}
	_, err := rand.Read(nonce[:])
	if err != nil {
		return nil, err
	}

	_, tempPrivateKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	//TODO: plaintext needs to be encrypted with aesCharif (to yield shwarma)
	shwarma := plaintext

	//this would be nice, but appengine doesn't allow unsafe: choppedPublicKey := (*[32]byte)(unsafe.Pointer(&publicKey[0]))

	var publicKeyArray [32]byte
	for i := 0; i < 32; i++ {
		publicKeyArray[i] = publicKey[i]
	}
	//shwarma gets wrapped with public key to yield tortilla
	tortilla := box.Seal(nil, shwarma, &nonce, &publicKeyArray, tempPrivateKey)

	//TODO: tortilla gets encrypted with aesTortilla to yield result
	finalResult := tortilla

	return finalResult, nil
}

//permute with mask
func PermuteWithMask(original []byte, mask []byte) ([]byte, error) {
	if len(original) != len(mask) {
		return nil, fmt.Errorf("DIFFERENT LENGTHS!")
	}

	ret := make([]byte, len(original))

	for idx, maskOp := range mask {
		ret[idx] = original[idx] ^ maskOp
	}

	return ret, nil

}

// encrypt string to base64 crypto using AES GCM Mode (ciphertext is iv->adata->ciphertext)
func EncryptAesGcm(key []byte, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	iv, err := RandomBytes(GCM_IV_SIZE)
	if err != nil {
		return "", err
	}

	adata, err := RandomBytes(GCM_ADATA_SIZE)
	if err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, iv, plaintext, adata)

	ciphertext = append(ciphertext, iv...)
	ciphertext = append(ciphertext, adata...)

	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// decrypt from base64 to decrypted string (strip iv->adata off front of ciphertext)
func DecryptAesGcm(key []byte, cryptoText string) ([]byte, error) {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	cipherLen := uint64(len(ciphertext))

	if cipherLen < (uint64(aes.BlockSize) + GCM_ADATA_SIZE + GCM_IV_SIZE) {
		return nil, fmt.Errorf("ciphertext too short: %v", cipherLen)
	}

	cipherBlock := ciphertext[:(cipherLen - (GCM_IV_SIZE + GCM_ADATA_SIZE))]
	iv := ciphertext[len(cipherBlock):(cipherLen - GCM_ADATA_SIZE)]
	adata := ciphertext[(cipherLen - GCM_ADATA_SIZE):]

	plaintext, err := gcm.Open(nil, iv, cipherBlock, adata)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func EncryptAesCbc(iv []byte, key []byte, data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		data = Pkcs7Pad(data, aes.BlockSize)
	}
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCEncrypter(c, iv)
	cbc.CryptBlocks(data, data)
	return data, nil
}

func DecryptAesCbc(iv, key, data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("bad blocksize(%v), aes.BlockSize = %v\n", len(data), aes.BlockSize)
	}
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCDecrypter(c, iv)
	cbc.CryptBlocks(data, data)
	out, err := Pkcs7Unpad(data, aes.BlockSize)
	if out == nil {
		return nil, err
	}
	return out, nil
}

// Appends padding.
func Pkcs7Pad(data []byte, blocklen int) []byte {

	padlen := 1
	for ((len(data) + padlen) % blocklen) != 0 {
		padlen = padlen + 1
	}

	pad := bytes.Repeat([]byte{byte(padlen)}, padlen)
	return append(data, pad...)
}

// Returns slice of the original data without padding.
func Pkcs7Unpad(data []byte, blocklen int) ([]byte, error) {
	if blocklen <= 0 {
		return nil, fmt.Errorf("invalid blocklen %d", blocklen)
	}
	if len(data)%blocklen != 0 || len(data) == 0 {
		return nil, fmt.Errorf("invalid data len %d", len(data))
	}
	padlen := int(data[len(data)-1])
	if padlen > blocklen || padlen == 0 {
		return nil, fmt.Errorf("invalid padding")
	}
	// check padding
	pad := data[len(data)-padlen:]
	for i := 0; i < padlen; i++ {
		if pad[i] != byte(padlen) {
			return nil, fmt.Errorf("invalid padding")
		}
	}

	return data[:len(data)-padlen], nil
}

func CreateHmac(message []byte, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)

	return mac.Sum(nil)
}

func ValidateHmac(sig []byte, message []byte, key []byte) bool {

	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(sig, expectedMAC)
}
