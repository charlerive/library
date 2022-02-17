package authenticator

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"github.com/charlerive/library/encrypt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	DefaultAuthSecret = "ZLO2F5BCYBBUEXKQH5KGJQWJ6UNV3H47"
	AESEncryptSecret  = "Ln28N29B7C4emC6p"
	AESEncryptIv      = "4374912809341387"
)

var (
	googleAuth     *GoogleAuth
	googleAuthOnce sync.Once
)

// GetGoogleAuthService 获取单例
func GetGoogleAuthService() *GoogleAuth {
	googleAuthOnce.Do(func() {
		googleAuth = &GoogleAuth{}
		googleAuth.start()
	})
	return googleAuth
}

type GoogleAuth struct {
	ctx       context.Context
	secret    string
	gaFile    string
	offsetLen int64
}

func (ga *GoogleAuth) start() {
	ga.ctx = context.Background()
	ga.secret = DefaultAuthSecret
	if execPath, err := os.Executable(); err == nil {
		ga.gaFile = execPath[:strings.LastIndexByte(execPath[:strings.LastIndexByte(execPath, '/')], '/')+1] + "ga"
	}
	ga.offsetLen = 1
}

func (ga *GoogleAuth) writeSecretToFile() {
	file, err := os.OpenFile(ga.gaFile, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err == nil {
		if encryptStr, err := encrypt.AesEncrypt(ga.secret, AESEncryptSecret, AESEncryptIv); err == nil {
			_, _ = file.WriteString(encryptStr)
			_ = file.Close()
		}
	}
}

func (ga *GoogleAuth) SetSecret(secret string) {
	ga.secret = secret
}

func (ga *GoogleAuth) Auth(code int) bool {
	if b, err := ioutil.ReadFile(ga.gaFile); err == nil {
		if decryptStr, err := encrypt.AesDecrypt(string(b), AESEncryptSecret, AESEncryptIv); err == nil && string(decryptStr) == ga.secret {
			return true
		}
	}

	t := time.Now().Unix() / 30
	minT := t - ga.offsetLen
	maxT := t + ga.offsetLen
	for minT <= maxT {
		expect := ga.getCode(ga.secret, minT)
		if expect == code {
			ga.writeSecretToFile()
			return true
		}
		minT++
	}

	return false
}

func (ga *GoogleAuth) Quit() {
	file, err := os.OpenFile(ga.gaFile, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err == nil {
		_ = file.Truncate(0)
		_ = file.Close()
	}
}

func (ga *GoogleAuth) getCode(secret string, value int64) int {

	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return -1
	}

	hash := hmac.New(sha1.New, key)
	err = binary.Write(hash, binary.BigEndian, value)
	if err != nil {
		return -1
	}
	h := hash.Sum(nil)

	offset := h[19] & 0x0f

	truncated := binary.BigEndian.Uint32(h[offset : offset+4])

	truncated &= 0x7fffffff
	code := truncated % 1000000

	return int(code)
}

func (ga *GoogleAuth) GenSecretKey() (string, error) {
	buf := bytes.Buffer{}
	err := binary.Write(&buf, binary.BigEndian, time.Now().Unix())
	if err != nil {
		return "", err
	}
	h := hmac.New(sha1.New, buf.Bytes())

	hSum := h.Sum(nil)
	secKey := base32.StdEncoding.EncodeToString(hSum)
	return secKey, nil
}
