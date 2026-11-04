package main

import (
	"fmt"
	"image/color"

	captcha "github.com/hicode0101/go-captcha"
)

func main() {
	bg := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	fonts := []string{"pinyin-cn.ttc"}
	// 中文验证码：imageHeight, imageWidth, noiseCount, lineOptions, textLength, charset, bgColor, fontBox, fontNames
	driver := captcha.NewChineseEngine(60, 320, 0, 0, 2,
		"大道,之行,天下,为公,选,贤,与,能,讲,信,修,睦,亲其,子其,老终,壮,用,幼,长,有养,有分,有归,藏己,谋闭,外户,大同",
		&bg,
		nil,
		fonts,
	)

	c := captcha.NewService(driver, captcha.DefaultRepository)
	id, b64s, answer, err := c.Create()
	if err != nil {
		panic(err)
	}
	fmt.Println("id:", id)
	fmt.Println("b64s:", b64s)
	fmt.Println("answer:", answer) // 调试输出正确答案

	// 校验用户输入；consume=true 表示校验后从仓库删除该验证码
	ok := c.Check(id, "用户输入的验证码", true)
	fmt.Println("check:", ok)
}
