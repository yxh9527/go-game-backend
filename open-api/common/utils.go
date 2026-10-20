package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var AesKey = "7z2q$l^Jw^DM3*Ol"
var AesIV = "tBu&NN$qQb9bi5L*"

func GetJsonObj(s string, data interface{}) interface{} {
	return &Response{
		S: s,
		M: "/channelHandle",
		D: data,
	}
}

func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

func Base64UrlEncode(source []byte) string {
	dist := base64.StdEncoding.EncodeToString(source)
	dist = strings.Replace(dist, "+", "-", -1)
	dist = strings.Replace(dist, "/", "_", -1)
	dist = strings.Replace(dist, "=", "", -1)
	return dist
}

func Base64UrlDEcode(source string) (string, int) {
	tmp := "===="
	dist := string(source)
	dist = strings.Replace(dist, "-", "+", -1)
	dist = strings.Replace(dist, "_", "/", -1)
	md4 := len(dist) % 4
	if md4 > 0 {
		dist = dist + tmp[:md4]
	}
	res, _ := base64.StdEncoding.DecodeString(dist)
	zap.L().Debug("base64解密结果", zap.String("res", string(res)))
	return string(res), len(res)
}

func HexEncode(source []byte) string {
	return hex.EncodeToString(source)
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}

func PKCS7Padding(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize        //需要padding的数目
	padtext := bytes.Repeat([]byte{byte(padding)}, padding) //生成填充的文本
	return append(ciphertext, padtext...)
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func AesEncrypt(aesKey, aesIv string, origData []byte) (string, error) {
	block, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		return "", err
	}
	blockSize := aes.BlockSize
	origData = PKCS5Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, []byte(aesIv)[:blockSize])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return Base64UrlEncode([]byte(crypted)), nil
}

func AesDecrypt(aesKey, aesIv string, param string) string {
	b, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		zap.L().Error("aes校验失败", zap.Any("err", err))
		return ""
	}
	decodeParam, _ := Base64UrlDEcode(param)
	plainTextBytes := make([]byte, len(decodeParam))
	cipher.NewCBCDecrypter(b, []byte(aesIv)).CryptBlocks(plainTextBytes, []byte(decodeParam))
	plainTextBytes = PKCS5UnPadding(plainTextBytes)
	return string(plainTextBytes)
}

func AesEncryptWithHex(aesKey, aesIv string, source string) string {
	b, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		zap.L().Error("aes校验失败", zap.Any("err", err))
		return ""
	}
	// 获取秘钥块的长度
	origData := PKCS5Padding([]byte(source), b.BlockSize())
	plainTextBytes := make([]byte, len(origData))
	cipher.NewCBCEncrypter(b, []byte(aesIv)).CryptBlocks(plainTextBytes, origData)
	return HexEncode(plainTextBytes)
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Header("Access-Control-Allow-Origin", "*") // 可将将 * 替换为指定的域名
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Cookie, X-CSRF-TOKEN, Accept, Authorization, X-XSRF-TOKEN")
		c.Header("Access-Control-Expose-Headers", "Authorization, authenticated,Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "false")
		c.Header("Server", "ApiServer")
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
		}
		c.Next()
	}
}
