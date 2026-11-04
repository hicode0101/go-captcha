package captcha

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// digitsToString 把随机数字字节序列（每个元素 0-9）转换成对应的数字字符串。
func digitsToString(bytes []byte) string {
	stringB := make([]byte, len(bytes))
	for idx, by := range bytes {
		stringB[idx] = by + '0'
	}
	return string(stringB)
}

// stringToDigitBytes 把数字字符串还原成字节切片（每个元素 0-9），
// 供点阵字模绘制使用。
func stringToDigitBytes(content string) []byte {
	digits := make([]byte, len(content))
	for idx, cc := range content {
		digits[idx] = byte(cc - '0')
	}
	return digits
}

// randomDigitBytes 生成长度为 length、每个元素都在 0-9 范围内的随机数字切片。
func randomDigitBytes(length int) []byte {
	return cryptoRandomBytesMod(length, 10)
}

// cryptoRandomBytes 从 CSPRNG 读取 length 字节的真随机数据，读取失败 panic。
func cryptoRandomBytes(length int) (b []byte) {
	b = make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("captcha: error reading random source: " + err.Error())
	}
	return
}

// cryptoRandomBytesMod 生成长度为 length、每个字节都对 mod 取模的随机切片。
// 通过丢弃超过 255-(256%mod) 的样本消除模偏差，保证各取值等概率。
func cryptoRandomBytesMod(length int, mod byte) (b []byte) {
	if length == 0 {
		return nil
	}
	if mod == 0 {
		panic("captcha: bad mod argument for randomBytesMod")
	}
	maxrb := 255 - byte(256%int(mod))
	b = make([]byte, length)
	i := 0
	for {
		r := cryptoRandomBytes(length + (length / 4))
		for _, c := range r {
			if c > maxrb {
				// 丢弃会引入模偏差的样本
				continue
			}
			b[i] = c % mod
			i++
			if i == length {
				return
			}
		}
	}
}

// SaveArtifactFile 把验证码产物落盘为文件，便于调试与离线查看。
// outputDir 不存在时会自动创建；文件名为 fileName.fileExt。
func SaveArtifactFile(art Artifact, outputDir, fileName, fileExt string) error {
	filePath := filepath.Join(outputDir, fileName+"."+fileExt)
	if !pathExists(outputDir) {
		_ = os.MkdirAll(outputDir, os.ModePerm)
	}
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("%s is invalid path.error:%v", filePath, err)
		return err
	}
	defer file.Close()
	_, err = art.SaveTo(file)
	return err
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
