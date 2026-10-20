package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

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
	// zap.L().Debug("base64解密结果", zap.String("res", string(res)))
	return string(res), len(res)
}

func HexEncode(source []byte) string {
	return hex.EncodeToString(source)
}

func HexDecode(source string) []byte {
	d, _ := hex.DecodeString(source)
	return d
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

// HmacSha256 计算HmacSha256
// key 是加密所使用的key
// data 是加密的内容
func HmacSha256(key string, data []byte) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

// HmacSha256ToHex 将加密后的二进制转16进制字符串
func HmacSha256ToHex(key string, data []byte) string {
	return hex.EncodeToString(HmacSha256(key, data))
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

func HttpPostAction(url string, br interface{}) ([]byte, error) {
	str, _ := jsoniter.MarshalToString(br)
	req, e := http.NewRequest("POST", url, strings.NewReader(str))
	if e != nil {
		zap.L().Error("请求失败,记录请求参数", zap.String("Url", url), zap.Any("数据内容", str), zap.Any("响应", e))
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	client := &http.Client{}
	if resp, ee := client.Do(req); ee != nil {
		zap.L().Error("请求失败,记录请求参数", zap.String("Url", url), zap.Any("数据内容", str), zap.Any("响应", ee))
		return nil, ee
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			if data, respErr := io.ReadAll(resp.Body); respErr != nil {
				zap.L().Error("请求失败,记录请求参数", zap.String("Url", url), zap.Any("数据内容", str), zap.Any("响应", respErr))
				return nil, respErr
			} else {
				return data, nil
			}
		}
		zap.L().Error("请求失败,记录请求参数", zap.String("Url", url), zap.Any("数据内容", str), zap.Any("响应", resp.StatusCode))
		return nil, errors.New("请求失败,http响应状态码异常")
	}
}

func HttpGetAction(url, headerType string, br interface{}) ([]byte, error) {
	str, _ := jsoniter.MarshalToString(br)
	req, e := http.NewRequest("GET", url, strings.NewReader(str))
	if e != nil {
		zap.L().Error("请求失败,记录请求参数", zap.String("Url", url), zap.Any("数据内容", str), zap.Any("响应", e))
		return nil, e
	}
	req.Header.Set("Content-Type", headerType)
	client := &http.Client{}
	if resp, ee := client.Do(req); ee != nil {
		zap.L().Error("请求失败,记录请求参数", zap.String("Url", url), zap.Any("数据内容", str), zap.Any("响应", ee))
		return nil, ee
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			if data, respErr := io.ReadAll(resp.Body); respErr != nil {
				zap.L().Error("请求失败,记录请求参数", zap.String("Url", url), zap.Any("数据内容", str), zap.Any("响应", respErr))
				return nil, respErr
			} else {
				return data, nil
			}
		}
		zap.L().Error("请求失败,记录请求参数", zap.String("Url", url), zap.Any("数据内容", str), zap.Any("响应", resp.StatusCode))
		return nil, errors.New("请求失败,http响应状态码异常")
	}
}

// 签名
func Sign(params map[string]string, key string) string {
	keys := make([]string, 0)
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	str := ""
	for _, v := range keys {
		str += v + "=" + params[v]
	}
	str += "key=" + key
	sig := fmt.Sprintf("%x", md5.Sum([]byte(str)))
	zap.L().Debug("签名结果", zap.Any("str", str), zap.Any("sig", sig))
	return sig
}

func CountPage(total, size int64) int64 {
	if total == 0 || size == 0 {
		return 0
	}
	if total%size == 0 {
		return total / size
	} else {
		return total/size + 1
	}
}

func GenRandString(size int) string {
	str := "1234567890QWERTYUIOPASDFGHJKLZXCVBNM"
	tmp := ""
	for i := 0; i < size; i++ {
		tmp += string(str[rand.Intn(len(str))])
	}
	return tmp
}
