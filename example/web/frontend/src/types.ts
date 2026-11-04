export type DriverKey = 'digit' | 'string' | 'math' | 'chinese' | 'audio'

export interface ColorState {
  r: number
  g: number
  b: number
  a: number
}

export interface GenerateParams {
  driver: DriverKey
  height: number
  width: number
  length: number
  maxSkew: number
  dotCount: number
  noiseCount: number
  showLineOptions: number
  source: string
  language: string
  bgColor: ColorState | null
  fonts: string[]
}

export interface GenerateResult {
  id: string
  b64s: string
}

/** 除 digit/audio 外的图片类 driver 共有参数 */
export interface ImageDriverState {
  width: number
  height: number
  noiseCount: number
  showLineOptions: number[]
  fonts: string[]
  bgColor: ColorState
}

export const LINE_OPTIONS = [
  { value: 2, label: 'OptionShowHollowLine' },
  { value: 4, label: 'OptionShowSlimeLine' },
  { value: 8, label: 'OptionShowSineLine' },
]

export const AUDIO_LANGUAGES = ['en', 'zh', 'ru', 'ja']

export const DRIVER_TABS: { key: DriverKey; label: string }[] = [
  { key: 'digit', label: 'DriverDigit' },
  { key: 'string', label: 'DriverString' },
  { key: 'math', label: 'DriverMath' },
  { key: 'chinese', label: 'DriverChinese' },
  { key: 'audio', label: 'DriverAudio' },
]

/** 与后端约定：RGBA 全 0 视为 nil，即随机浅色背景 */
export function isNoColor(c: ColorState): boolean {
  return c.r === 0 && c.g === 0 && c.b === 0 && c.a === 0
}

export function sumLineOptions(values: number[]): number {
  return values.reduce((acc, v) => acc + v, 0)
}
