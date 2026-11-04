import type { ColorState, DriverKey, ImageDriverState } from './types'
import { isNoColor, sumLineOptions } from './types'

export interface DigitState {
  length: number
  width: number
  height: number
  maxSkew: number
  dotCount: number
}

export interface AudioState {
  length: number
  language: string
}

export interface GoCodeStates {
  digit: DigitState
  string: ImageDriverState & { length: number; source: string }
  math: ImageDriverState
  chinese: ImageDriverState & { length: number; source: string }
  audio: AudioState
}

function colorExpr(c: ColorState): { arg: string; stmt: string; needImport: boolean } {
  if (isNoColor(c)) {
    return { arg: 'nil', stmt: '', needImport: false }
  }
  return {
    arg: '&bg',
    stmt: `\tbg := color.RGBA{R: ${c.r}, G: ${c.g}, B: ${c.b}, A: ${c.a}}\n`,
    needImport: true,
  }
}

function fontsStmt(fonts: string[]): string {
  if (fonts.length === 0) {
    return '\tvar fonts []string // nil 表示使用全部内嵌字体\n'
  }
  return `\tfonts := []string{${fonts.map((f) => JSON.stringify(f)).join(', ')}}\n`
}

function lineOptionComment(values: number[]): string {
  const total = sumLineOptions(values)
  if (total === 0) return ''
  const names = values
    .slice()
    .sort((a, b) => a - b)
    .map((v) => (v === 2 ? 'LineOptionHollow(2)' : v === 4 ? 'LineOptionSlime(4)' : 'LineOptionSine(8)'))
  return `\t// lineOptions: ${names.join(' | ')} = ${total}\n`
}

function quote(s: string): string {
  // JSON.stringify 的转义规则与 Go 字符串字面量兼容
  return JSON.stringify(s)
}

/** 根据当前引擎与参数生成对应的 Go 使用代码 */
export function buildGoCode(driver: DriverKey, s: GoCodeStates): string {
  let needImport = false
  let body = ''

  switch (driver) {
    case 'digit': {
      const d = s.digit
      body = `\t// 数字验证码：imageHeight, imageWidth, digitCount, maxSkew, noiseDots
\tdriver := captcha.NewDigitEngine(${d.height}, ${d.width}, ${d.length}, ${d.maxSkew}, ${d.dotCount})`
      break
    }
    case 'string': {
      const st = s.string
      const color = colorExpr(st.bgColor)
      needImport = color.needImport
      body = `${color.stmt}${fontsStmt(st.fonts)}${lineOptionComment(st.showLineOptions)}\t// 字符串验证码：imageHeight, imageWidth, noiseCount, lineOptions, textLength, charset, bgColor, fontBox, fontNames
\tdriver := captcha.NewStringEngine(${st.height}, ${st.width}, ${st.noiseCount}, ${sumLineOptions(st.showLineOptions)}, ${st.length},
\t\t${quote(st.source)},
\t\t${color.arg},${color.needImport ? '' : ' // nil 表示随机浅色背景'}
\t\tnil, // nil 表示使用内嵌字体仓库
\t\tfonts,
\t)`
      break
    }
    case 'math': {
      const st = s.math
      const color = colorExpr(st.bgColor)
      needImport = color.needImport
      body = `${color.stmt}${fontsStmt(st.fonts)}${lineOptionComment(st.showLineOptions)}\t// 算术验证码：imageHeight, imageWidth, noiseCount, lineOptions, bgColor, fontBox, fontNames
\tdriver := captcha.NewMathEngine(${st.height}, ${st.width}, ${st.noiseCount}, ${sumLineOptions(st.showLineOptions)}, ${color.arg}, nil, fonts)`
      break
    }
    case 'chinese': {
      const st = s.chinese
      const color = colorExpr(st.bgColor)
      needImport = color.needImport
      body = `${color.stmt}${fontsStmt(st.fonts)}${lineOptionComment(st.showLineOptions)}\t// 中文验证码：imageHeight, imageWidth, noiseCount, lineOptions, textLength, charset, bgColor, fontBox, fontNames
\tdriver := captcha.NewChineseEngine(${st.height}, ${st.width}, ${st.noiseCount}, ${sumLineOptions(st.showLineOptions)}, ${st.length},
\t\t${quote(st.source)},
\t\t${color.arg},
\t\tnil,
\t\tfonts,
\t)`
      break
    }
    case 'audio': {
      const a = s.audio
      body = `\t// 语音验证码：digitCount, language(en|zh|ru|ja)
\tdriver := captcha.NewAudioEngine(${a.length}, ${quote(a.language)})`
      break
    }
  }

  const colorImport = needImport ? '\n\t"image/color"' : ''
  return `package main

import (
\t"fmt"${colorImport}

\tcaptcha "github.com/hicode0101/go-captcha"
)

func main() {
${body}

\tc := captcha.NewService(driver, captcha.DefaultRepository)
\tid, b64s, answer, err := c.Create()
\tif err != nil {
\t\tpanic(err)
\t}
\tfmt.Println("id:", id)
\tfmt.Println("b64s:", b64s)
\tfmt.Println("answer:", answer) // 调试输出正确答案

\t// 校验用户输入；consume=true 表示校验后从仓库删除该验证码
\tok := c.Check(id, "用户输入的验证码", true)
\tfmt.Println("check:", ok)
}
`
}
