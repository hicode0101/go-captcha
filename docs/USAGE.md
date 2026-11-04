# go-captcha 参数使用文档

本文档基于当前代码库（`module github.com/hicode0101/go-captcha`，Go 1.21）逐一说明所有公开 API 的参数含义、
取值范围、边界行为与推荐配置，并给出可直接运行的示例。所有 Web 演示统一使用 **8090** 端口。

- 五种验证码引擎：数字点阵 / 字符串 / 算术 / 中文 / 语音
- 统一输出带 MIME 前缀的 base64（`data:image/png;base64,...` 或 `data:audio/wav;base64,...`），前端 `<img src>` / `<audio src>` 直接可用
- 字体内嵌（6 款，含中文字体 `chinese.ttf` / `pinyin-cn.ttc`），二进制零外部依赖

---

## 目录

1. [整体架构](#1-整体架构)
2. [安装与快速开始](#2-安装与快速开始)
3. [Service 服务层参数](#3-service-服务层参数)
4. [Repository 答案仓库参数](#4-repository-答案仓库参数)
5. [五大引擎构造参数详解](#5-五大引擎构造参数详解)
6. [公共参数详解](#6-公共参数详解)
7. [Artifact 产物与落盘](#7-artifact-产物与落盘)
8. [工具函数](#8-工具函数)
9. [Web 调试示例（8090 端口）](#9-web-调试示例8090-端口)
10. [HTTP 接口参数（example 后端）](#10-http-接口参数example-后端)
11. [完整接入示例（8090 端口）](#11-完整接入示例8090-端口)
12. [生产环境建议](#12-生产环境建议)

---

## 1. 整体架构

```
业务代码
  │
  ├─ Service（主入口）─── Create() 出题 + 渲染 + 存答案
  │      │                Check()  校验用户输入
  │      │
  │      ├─ Engine（引擎接口）        五种内置引擎均可替换/自扩展
  │      │     ├─ NewChallenge()  → id, question, answer
  │      │     └─ Render(question) → Artifact
  │      │
  │      └─ Repository（仓库接口）    id → answer 存取、一次性消费
  │            ├─ DefaultRepository（内存，默认）
  │            ├─ NewMemRepository / NewSyncRepository
  │            └─ 自定义实现（Redis 等）
  │
  └─ Artifact（产物接口）
        ├─ SaveTo(w)       写出 PNG / WAV 二进制
        └─ Base64String()  带 MIME 前缀的 base64 字符串
```

三大接口定义见 `contract.go`；接口参数的语义由各实现决定，下文逐个展开。

## 2. 安装与快速开始

```bash
go get github.com/hicode0101/go-captcha
```

```go
package main

import (
	"fmt"

	captcha "github.com/hicode0101/go-captcha"
)

func main() {
	// 数字引擎：画布高 80, 宽 240, 5 个数字, 最大倾斜 0.7, 背景圆点 80
	engine := captcha.NewDigitEngine(80, 240, 5, 0.7, 80)

	svc := captcha.NewService(engine, captcha.DefaultRepository)
	id, b64s, answer, err := svc.Create()
	if err != nil {
		panic(err)
	}
	fmt.Println(id)     // 20 位会话 id，随验证码一起下发给前端
	fmt.Println(b64s)   // data:image/png;base64,... 直接放入 <img src>
	fmt.Println(answer) // 调试用：正确答案，生产环境勿下发

	// 校验：consume=true 表示无论成败都消费掉该验证码（防重放，推荐）
	fmt.Println(svc.Check(id, "90509", true))
}
```

每个引擎都有对应的“推荐默认配置”预置变量：

| 预置变量 | 等价构造 | 说明 |
| --- | --- | --- |
| `captcha.DefaultDigitEngine` | `NewDigitEngine(80, 240, 5, 0.7, 80)` | 数字引擎推荐配置 |
| `captcha.DefaultAudioEngine` | `NewAudioEngine(6, "en")` | 语音引擎推荐配置 |

## 3. Service 服务层参数

`Service` 并发安全，建议全局持有一个实例（见 `service.go`）。

### NewService(engine Engine, repo Repository) *Service

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `engine` | `Engine` | 出题与渲染引擎，五种内置引擎任选，或自行实现接口扩展 |
| `repo` | `Repository` | 答案存储仓库；**传 `nil` 时自动使用 `DefaultRepository`**（进程内内存仓库，10 分钟过期） |

多实例共享同一个 `Repository` 即可实现跨服务的验证码校验（例如同一进程内多个 Service、
或多个后端实例共用 Redis 仓库）。

### Create() (id, encoded, answer string, err error)

无入参。每次调用都会向仓库写入一条答案记录，过期与回收策略由仓库决定。

| 返回值 | 说明 |
| --- | --- |
| `id` | 20 位随机会话 id（数字 + 字母，CSPRNG 生成），需与 `encoded` 一起下发前端，校验时原样传回 |
| `encoded` | 带前缀的 base64 产物：图片为 `data:image/png;base64,...`，语音为 `data:audio/wav;base64,...` |
| `answer` | 正确答案明文。**仅建议调试场景打印，生产环境勿下发给前端** |
| `err` | 出题或渲染失败时非 `nil`（此时其余返回值为空） |

### Check(id, answer string, consume bool) bool

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `string` | `Create` 时下发的会话 id |
| `answer` | `string` | 用户输入。两侧都会先去首尾空白再**忽略大小写**比较 |
| `consume` | `bool` | `true`：无论校验成败都删除该记录（防重放，**推荐**）；`false`：允许反复校验，直到仓库过期 |

返回 `true` 表示校验通过。`consume=true` 时第二次用同一 `id` 校验必然失败。

## 4. Repository 答案仓库参数

### 4.1 DefaultRepository（默认，零配置）

```go
DefaultRepository = NewMemRepository(DefaultCollectThreshold, DefaultTTL)
// DefaultCollectThreshold = 10240
// DefaultTTL = 10 * time.Minute
```

进程内共享的内存仓库：`map` 存答案 + 时间链表记录写入顺序，
**累计写入超过 10240 条时异步触发一次全量过期回收**（`repository_memory.go`）。
校验比对忽略大小写；id 或 answer 为空一律不匹配。

### 4.2 NewMemRepository(collectThreshold int, ttl time.Duration) Repository

| 参数 | 类型 | 说明 | 常用值 |
| --- | --- | --- | --- |
| `collectThreshold` | `int` | 累计写入多少条后触发一次过期回收（异步 goroutine） | `10240` |
| `ttl` | `time.Duration` | 验证码存活时长，超期记录在下一次回收时清除 | `10 * time.Minute` |

- 回收是**阈值触发**而非定时触发：流量小的服务中，过期记录会保留到下一次触发回收才删除，但 `Load`/`Match` 不受影响——过期记录只是占内存，不会被匹配成功（前提是业务侧配合 `consume` 或校验时间窗）。若需要严格的过期判定，请选 `SyncRepository` 或自定义实现。
- `Put` 永远返回 `nil`；并发安全（读写锁）。

### 4.3 NewSyncRepository(liveTime time.Duration) *SyncRepository

| 参数 | 类型 | 说明 | 常用值 |
| --- | --- | --- | --- |
| `liveTime` | `time.Duration` | 记录存活时长 | `10 * time.Minute` |

基于 `sync.Map`，**每次 `Put` 前先全表清扫一遍已超时记录**（惰性过期，无后台 goroutine）。
适合写入频率低、希望过期数据尽快释放的场景；写入频率极高时全表 Range 有开销，
此时建议用 `NewMemRepository`。

### 4.4 自定义仓库（接入 Redis）

实现三个方法即可：

```go
// Repository 接口
type Repository interface {
	Put(id string, answer string) error
	Load(id string, consume bool) string // consume=true 时读取后立即删除
	Match(id, answer string, consume bool) bool
}
```

Redis 版骨架（TTL 直接交给 Redis 的 `EXPIRE`）：

```go
type RedisRepo struct{ rdb *redis.Client; ttl time.Duration }

func (r *RedisRepo) Put(id, answer string) error {
	return r.rdb.Set(context.Background(), "captcha:"+id, answer, r.ttl).Err()
}

func (r *RedisRepo) Load(id string, consume bool) string {
	key := "captcha:" + id
	v, err := r.rdb.Get(context.Background(), key).Result()
	if err != nil {
		return ""
	}
	if consume {
		r.rdb.Del(context.Background(), key)
	}
	return v
}

func (r *RedisRepo) Match(id, answer string, consume bool) bool {
	return strings.EqualFold(r.Load(id, consume), strings.TrimSpace(answer))
}
```

## 5. 五大引擎构造参数详解

### 5.1 DigitEngine — 数字点阵验证码

数字由圆形“墨点”拼成，经过倾斜、波浪删除线与正弦扭曲，背景撒圆点噪声，输出 PNG。
**内部固定行为**（不受参数控制）：波浪删除线始终绘制、正弦波扭曲始终开启
（振幅 5~10px、周期 100~200px 随机）。

```go
func NewDigitEngine(imageHeight int, imageWidth int, digitCount int, maxSkew float64, noiseDots int) *DigitEngine
```

| 参数 | 类型 | 说明 | 常用值 | 边界行为 |
| --- | --- | --- | --- | --- |
| `imageHeight` | `int` | 画布高度（像素） | `80` | 与宽度共同决定墨点尺寸，过小会被钳制到最小 1px 点径 |
| `imageWidth` | `int` | 画布宽度（像素） | `240` | 同上 |
| `digitCount` | `int` | 数字个数 | `4`~`6` | `<=0` 时按 4 个出题 |
| `maxSkew` | `float64` | 单个数字的最大倾斜系数，逐字形逐行水平错切，取值范围 `[-maxSkew, +maxSkew]` | `0.7` | `0` 表示不倾斜；过大（>1）数字明显歪斜但仍可绘制 |
| `noiseDots` | `int` | 背景噪声圆点数量 | `80` | `0` 表示纯数字无噪点；兼任调色板颜色变体数（见下方说明） |

**`noiseDots` 的双重作用**（`painters_digit.go` 的 `buildPalette`）：

- `noiseDots > 0`：PNG 背景**透明**（0 号色 alpha=0），数字为主色，噪声圆点为主色的随机明度变体；
- `noiseDots = 0`：不画噪声圆点，且 0 号色改为主色，即输出**实底色**图片。

**字模尺寸自适应**：墨点直径按“画布减去边距（短边 1/4）后均分数字个数”推算，
宽度方向放不下自动改为按高度约束；极端参数（画布小、数字多）下定位坐标会被钳制到边界，不会 panic。
调试页推荐滑杆范围：长度 1~10、宽 80~600、高 40~200、倾斜 0~1、圆点 0~200。

示例：

```go
// 标准配置
e1 := captcha.NewDigitEngine(80, 240, 5, 0.7, 80)

// 无噪声、无倾斜的“干净”数字图（透明背景变为实底色）
e2 := captcha.NewDigitEngine(80, 240, 4, 0, 0)

// 高难度：6 位数字 + 强倾斜 + 大量噪声
e3 := captcha.NewDigitEngine(90, 260, 6, 0.9, 150)
```

### 5.2 StringEngine — 字符串验证码

从候选字符集随机取 N 个字符，逐字符随机字号/深色/纵向抖动绘制，支持干扰线、
背景噪点文字与自定义背景色，输出 PNG。

```go
func NewStringEngine(imageHeight int, imageWidth int, noiseCount int, lineOptions int,
	textLength int, charset string, bgColor *color.RGBA, fontBox FontBox, fontNames []string) *StringEngine
```

| 参数 | 类型 | 说明 | 常用值 |
| --- | --- | --- | --- |
| `imageHeight` | `int` | 画布高度（像素）。主体字号为高度的 7/16 ~ 13/16 随机 | `60` |
| `imageWidth` | `int` | 画布宽度（像素）。每个字符均分宽度 | `240` |
| `noiseCount` | `int` | 背景噪点文字个数，从 `数字+字母+,.[]<>` 中随机取，浅色、约半高字号 | `0`~`5` |
| `lineOptions` | `int` | 干扰线位组合，见 [6.1](#61-干扰线参数-lineoptions) | `0` 或按位或组合 |
| `textLength` | `int` | 验证码字符个数 | `4`~`6` |
| `charset` | `string` | 候选字符集，**按 rune 采样**，可直接用预置 `Charset*` 常量或自定义 | `CharsetDigits + CharsetLetters` |
| `bgColor` | `*color.RGBA` | 背景色指针；**`nil` 表示每次随机浅色背景** | `nil` |
| `fontBox` | `FontBox` | 字体仓库；`nil` 使用内嵌仓库 `BuiltinFontBox` | `nil` |
| `fontNames` | `[]string` | 字体文件名列表；**空/nil 使用全部内嵌字体** | `nil` 或 `[]string{"actionj.ttf"}` |

绘制顺序：干扰线（可选）→ 背景噪点文字（可选）→ 主体文字。
主体文字为空串时渲染返回 error。

### 5.3 MathEngine — 算术验证码

随机一道加/减/乘口算题，题目绘制到图片上，**答案为计算结果**（question ≠ answer）。

```go
func NewMathEngine(imageHeight int, imageWidth int, noiseCount int, lineOptions int,
	bgColor *color.RGBA, fontBox FontBox, fontNames []string) *MathEngine
```

参数含义与 `StringEngine` 完全一致，只是**没有 `textLength` 和 `charset`**（题目长度与内容由出题逻辑决定）。

出题规则（`challenge_math.go`）：

| 运算 | 操作数范围 | 示例 question | answer |
| --- | --- | --- | --- |
| 加法 `+` | 两数均取自 `[0, 20)` | `12+5=?` | `17` |
| 乘法 `x` | 两数均取自 `[0, 10)` | `3x4=?` | `12` |
| 减法 `-` | 被减数 `rand.Intn(80)+rand.Intn(20)` ∈ `[0,100)`，减数 ∈ `[0,80)`，**保证结果非负** | `97-35=?` | `62` |

三种运算等概率随机。背景噪点文字固定使用数字字符集。校验时用户输入计算结果即可
（`Check` 忽略大小写并去首尾空白，纯数字不受影响）。

### 5.4 ChineseEngine — 中文验证码

参数与 `StringEngine` 完全一致。`charset` 支持**两种写法**：

1. **纯字符集（无逗号）**：从字符集中按 rune 随机取 `textLength` 个字：

   ```go
   e := captcha.NewChineseEngine(60, 320, 0, 0, 2, captcha.CharsetChinese, nil, nil, nil)
   ```

2. **逗号分隔的候选词组**：每次逐位从词组列表随机抽一个词组拼接出题，
   更贴近真实语序、便于朗读（语音播报场景友好）：

   ```go
   charset := "北京,上海,广州,深圳,杭州,成都"
   e := captcha.NewChineseEngine(60, 400, 0, 0, 2, charset, nil, nil, nil)
   // 出题示例：杭州北京 / 上海深圳 ……
   ```

   **注意**：词组数 `<= textLength` 时会退化为从 `CharsetDigits+CharsetLetters` 随机取字符，
   请保证候选词组数量大于 `textLength`。

**字体要求**：渲染中文必须使用中文字体。内嵌的 `chinese.ttf` 与 `pinyin-cn.ttc`（汉字带拼音标注）
均已加入默认随机池：显式指定 `fontNames: []string{"chinese.ttf"}`（演示页默认）可正常渲染中文；
留空则从全部内嵌字体随机挑选，西文字体被抽中时会出豆腐块，纯中文场景建议显式指定。

### 5.5 AudioEngine — 语音验证码

输出 WAV 音频（8kHz、8bit、单声道）。音频结构：三声前置提示音 → 按随机间隔朗读各数字
→ 背景白噪声与倒置人声碎片 → 结尾提示音。每个数字发音都有随机变速（0.95~1.1 倍）与
音量抖动，使每次生成的音频都不完全相同，增加机器识别难度。

```go
func NewAudioEngine(digitCount int, language string) *AudioEngine
```

| 参数 | 类型 | 说明 | 常用值 | 边界行为 |
| --- | --- | --- | --- | --- |
| `digitCount` | `int` | 数字个数 | `6` | `<=0` 时按 6 个出题 |
| `language` | `string` | 数字发音语言：`"en"` 英语 / `"zh"` 中文 / `"ru"` 俄语 / `"ja"` 日语 | `"en"`、`"zh"` | 不支持或缺少样本的语言**回退英语** |

- 产物为 `data:audio/wav;base64,...`，前端 `<audio src="...">` 直接播放；
- 数字之间有 1~2 秒随机间隔，整段音频长度随数字个数增长（6 位约 15 秒量级），
  前端注意播放器布局；
- `Render(content)` 要求 content 全为 `'0'~'9'`，其他字符的行为未定义（请勿手工调用时传非法值）。

## 6. 公共参数详解

### 6.1 干扰线参数 lineOptions

`StringEngine` / `MathEngine` / `ChineseEngine` 共用，定义在 `assets.go`，**按位或组合**，`0` 表示不画线：

| 常量 | 值 | 效果 |
| --- | --- | --- |
| `LineOptionHollow` | `2` | 一条横贯画面的粗空心正弦线（线宽约高度 1/20，浅色，1/10 概率上下翻转） |
| `LineOptionSlime` | `4` | 3 条随机深色的斜细线，起止点取自画面左右两侧，奇偶交替形成交叉 |
| `LineOptionSine` | `8` | 多条小振幅正弦波浪线（随机相位、周期与颜色） |

```go
// 三种全开
opts := captcha.LineOptionHollow | captcha.LineOptionSlime | captcha.LineOptionSine

// 只要波浪线
opts := captcha.LineOptionSine

// 不画线
opts := 0
```

> 注意：`DigitEngine` 不使用该参数（它的干扰手段固定为删除线 + 波扭曲 + 圆点噪声）。

### 6.2 背景色参数 bgColor

类型 `*color.RGBA`（`image/color` 包），对 `StringEngine` / `MathEngine` / `ChineseEngine` 生效：

| 取值 | 行为 |
| --- | --- |
| `nil` | 每次生成验证码时**随机浅色背景**（`RandomLightTone`：三通道各在 200~254 之间） |
| 指定值 | 固定使用该颜色。示例：`&color.RGBA{R: 125, G: 125, B: 0, A: 118}` |

主体文字颜色始终为随机深色（`RandomDarkColor`），与浅色背景自动形成对比；
若自定义了较深的背景色，对比度会下降，建议背景保持浅色系。

> `DigitEngine` / `AudioEngine` 无背景色参数：前者背景透明或主色实底，后者与颜色无关。

### 6.3 字体参数 fontBox 与 fontNames

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `fontBox` | `FontBox` 接口 | 字体仓库抽象（`OpenFont(name)` / `OpenFonts(names)`），`nil` 使用内嵌仓库 `BuiltinFontBox`；从磁盘/网络加载自定义字体时实现该接口 |
| `fontNames` | `[]string` | 参与渲染的字体文件名列表；**空/nil 表示使用全部内嵌字体**（每次绘制随机挑一款） |

内嵌字体（`fonts/` 目录整体嵌入，`BuiltinFontNames()` 可枚举）：

```
随机池（fontNames 为空时从中随机挑选，共 6 款）：
3Dumb.ttf  ApothecaryFont.ttf  DENNEthree-dee.ttf
actionj.ttf  chinese.ttf  pinyin-cn.ttc   ← 后两款为中文字体
```

要点：

- `fontNames` 中只写文件名（如 `"actionj.ttf"`），不带 `fonts/` 前缀——库内部会自动拼接；
- **中文验证码务必使用 `chinese.ttf` 或 `pinyin-cn.ttc`**（显式传入），否则可能抽中不含汉字字形的西文字体；
- 字体加载/解析失败以 **panic** 暴露（属于部署错误，应在启动期暴露），自定义 `FontBox` 实现时保持同样语义即可；
- 以 struct 字面量方式构造引擎（绕过构造函数）时，字段赋值后需手动调用 `LoadFonts()` 重新加载字体（链式调用）：

  ```go
  e := &captcha.StringEngine{ImageHeight: 60, ImageWidth: 240, TextLength: 4,
      Charset: captcha.CharsetDigits, FontNames: []string{"actionj.ttf"}}
  e.LoadFonts()
  ```

- 自定义字体仓库示例：

  ```go
  //go:embed myfonts/*.ttf
  var myFS embed.FS

  box := captcha.NewEmbedFontBox(myFS) // 已实现 FontBox 接口
  e := captcha.NewStringEngine(60, 240, 0, 0, 4, captcha.CharsetDigits, nil, box, []string{"myfonts/custom.ttf"})
  ```

### 6.4 字符集常量 charset

`assets.go` 预置字符集，可单独或拼接使用，也可完全自定义：

| 常量 | 内容 | 长度 | 说明 |
| --- | --- | --- | --- |
| `CharsetDigits` | `012346789` | 9 | 纯数字，**剔除了易混淆的 5**，与点阵绘制字形保持一致 |
| `CharsetLetters` | `ABCDEFGHJKMNOQRSTUVXYZabcdefghjkmnoqrstuvxyz` | 44 | 英文字母，已剔除易混淆字母（不含 I/l 等） |
| `CharsetSimple` | `13467ertyiadfhjkxcvbnERTYADFGHJKXCVBN` | 38 | 数字字母混合精简集，人工辨识友好 |
| `CharsetChinese` | 常用汉字 500 个 | 500 | 中文验证码默认候选字集 |
| `CharsetEmoji` | `😀😃💯😄🤖…` 等 emoji | — | 可作为 `StringEngine` 的候选字集；**实际渲染取决于所选字体是否覆盖 emoji 字形**，建议先在调试页（8090）确认效果再用 |

自定义示例：

```go
// 只用大写字母 + 数字
e := captcha.NewStringEngine(60, 240, 0, 0, 4, "012346789ABCDEFGHJKMNPQRSTUVWXYZ", nil, nil, nil)

// 剔除易混淆的字母后拼接
e := captcha.NewStringEngine(60, 240, 2, captcha.LineOptionSine, 5,
	captcha.CharsetDigits+captcha.CharsetLetters, nil, nil, nil)
```

采样规则（`RandRunes`）：按 rune 随机取 `size` 个；候选集为空或 size 为 0 返回空串；
size 大于字符集长度时自动重复扩展字符集保证采样空间。

## 7. Artifact 产物与落盘

`Render` 返回的 `Artifact` 有两个图片实现（`DotMatrixImage` / `CharImage`）和一个音频实现（`VoiceClip`），
统一接口：

| 方法 | 说明 |
| --- | --- |
| `SaveTo(w io.Writer) (int64, error)` | 以二进制（PNG 或 WAV）写入 `w`，返回字节数 |
| `Base64String() string` | 返回 `data:image/png;base64,...` 或 `data:audio/wav;base64,...` 完整字符串 |

HTTP 直接输出二进制（省流量）：

```go
art, err := engine.Render("90509")
w.Header().Set("Content-Type", captcha.MimeImagePng) // 或 captcha.MimeAudioWav
art.SaveTo(w)
```

调试落盘（目录不存在自动创建）：

```go
art, err := engine.Render("90509")
if err != nil {
	log.Fatal(err)
}
captcha.SaveArtifactFile(art, "./output", "demo", "png") // ./output/demo.png
```

## 8. 工具函数

| 函数 | 签名 | 说明 |
| --- | --- | --- |
| `RandRunes` | `func RandRunes(size int, charset string) string` | 从字符集按 rune 随机取 size 个字符；size ≥ 字符集长度时自动重复扩展 |
| `NewSessionID` | `func NewSessionID() string` | 生成 20 位数字字母会话 id（CSPRNG + 模偏差消除） |
| `RandomDarkColor` | `func RandomDarkColor() color.RGBA` | 随机深色（主体文字用） |
| `RandomLightTone` | `func RandomLightTone() color.RGBA` | 随机浅色（背景/浅色干扰用） |
| `RandomVividColor` | `func RandomVividColor() color.RGBA` | 随机高饱和色（配色种子） |
| `SaveArtifactFile` | `func SaveArtifactFile(art Artifact, outputDir, fileName, fileExt string) error` | 产物落盘，目录不存在自动创建 |
| `BuiltinFontNames` | `func BuiltinFontNames() []string` | 枚举内嵌字体文件名（不含 `fonts/` 前缀），可直接用于前端勾选列表或 `fontNames` 参数 |

## 9. Web 调试示例（8090 端口）

`example/web` 是一个可实时调参的可视化调试页（Vue3 + Naive UI + TypeScript），
前端展示五种引擎的全部参数滑杆，并生成等价的 Go 接入代码。

```bash
# 1. 构建前端
cd example/web/frontend
npm install
npm run build

# 2. 启动服务（默认监听 :8090，自带静态文件服务）
cd ../server
go run .
# 浏览器打开 http://localhost:8090
```

- 后端默认端口为 **`:8090`**，可用 `-addr` 覆盖：`go run . -addr :9000`；
- 静态文件目录默认 `./static`，可用 `-static` 覆盖；
- **字体列表自动跟随 `fonts/` 目录**：服务启动时检测字体目录（默认 `../../../fonts`，可用 `-fonts` 覆盖），
  目录可读则进入磁盘模式——`/api/fonts` 实时枚举目录内的 `.ttf`/`.ttc` 文件，
  **新增或删除字体文件后无需重新编译、无需重启**，刷新页面即自动更新所有字体勾选列表，
  新字体也可直接参与渲染（未勾选任何字体时全部磁盘字体进入随机池）；
  目录不可读时回退编译期内嵌字体；
- 服务开启 CORS（`*`），前端开发模式可直接跨域调用。

前端开发模式（可选）：

```bash
cd example/web/frontend
npm run dev        # Vite 启动在 5173，/api 已代理到 http://localhost:8090
```

## 10. HTTP 接口参数（example 后端）

`example/web/server/main.go` 暴露 3 个接口：

### POST /api/generate — 生成验证码

请求体（JSON，所有字段可选，缺省时按 `applyDefaults` 补齐）：

| 字段 | 类型 | 适用 driver | 说明 | 缺省值 |
| --- | --- | --- | --- | --- |
| `driver` | `string` | 全部 | 引擎类型：`digit` / `string` / `math` / `chinese` / `audio`（其他值按 `string` 处理） | `string` |
| `height` | `int` | 图片类 | 画布高度（px） | `80` |
| `width` | `int` | 图片类 | 画布宽度（px） | `240` |
| `length` | `int` | 全部 | 字符/数字个数（digit/string/chinese/audio） | `4` |
| `maxSkew` | `float64` | digit | 数字最大倾斜系数 | `0.7` |
| `dotCount` | `int` | digit | 背景噪声圆点数量 | `0` |
| `noiseCount` | `int` | string/math/chinese | 背景噪点文字数量 | `0` |
| `showLineOptions` | `int` | string/math/chinese | 干扰线位组合（2/4/8 按位或） | `0` |
| `source` | `string` | string/chinese | 候选字符集（chinese 支持逗号分隔词组） | 空 |
| `language` | `string` | audio | 发音语言 `en`/`zh`/`ru`/`ja` | 空（回退 `en`） |
| `bgColor` | `object\|null` | string/math/chinese | 背景色 `{r,g,b,a}`，**四通道全 0 视为 nil（随机浅色）** | `null`（演示页默认四通道 255，白色不透明） |
| `fonts` | `[]string` | string/math/chinese | 字体文件名列表（空使用全部内嵌字体） | 空 |

响应：

```json
{ "code": 0, "data": { "id": "a1B2c3...", "b64s": "data:image/png;base64,..." } }
```

失败时 `code` 非 0，`message` 为错误信息。

### POST /api/verify — 校验验证码

```json
// 请求
{ "id": "a1B2c3...", "answer": "90509" }
// 响应（consume=true，一次性）
{ "code": 0, "data": { "ok": true } }
```

### GET /api/fonts — 枚举可用字体

返回 `data` 为字体文件名数组，前端用它渲染字体勾选列表。
磁盘模式下实时枚举 `fonts/` 目录（随目录增删自动更新，无需重编译），
目录不可读时回退内嵌字体清单（`BuiltinFontNames()`）。

### curl 示例（默认 8090 端口）

```bash
# 数字验证码
curl -X POST http://localhost:8090/api/generate \
  -H "Content-Type: application/json" \
  -d '{"driver":"digit","height":80,"width":240,"length":5,"maxSkew":0.7,"dotCount":80}'

# 字符串验证码（干扰线全开 + 指定字体）
curl -X POST http://localhost:8090/api/generate \
  -H "Content-Type: application/json" \
  -d '{"driver":"string","height":60,"width":240,"length":4,
       "noiseCount":2,"showLineOptions":14,
       "source":"012346789ABCDEFGHJKMNOQRSTUVXYZ","fonts":["actionj.ttf"]}'

# 中文验证码（逗号分隔词组写法）
curl -X POST http://localhost:8090/api/generate \
  -H "Content-Type: application/json" \
  -d '{"driver":"chinese","height":60,"width":400,"length":2,
       "source":"北京,上海,广州,深圳,杭州,成都","fonts":["chinese.ttf"]}'

# 语音验证码（中文发音）
curl -X POST http://localhost:8090/api/generate \
  -H "Content-Type: application/json" \
  -d '{"driver":"audio","length":6,"language":"zh"}'

# 校验（id 换成上一步返回值）
curl -X POST http://localhost:8090/api/verify \
  -H "Content-Type: application/json" \
  -d '{"id":"a1B2c3...","answer":"90509"}'

# 内嵌字体列表
curl http://localhost:8090/api/fonts
```

## 11. 完整接入示例（8090 端口）

一个自包含的 `main.go`，演示"出题 + 下发 + 校验"的完整 Web 链路，默认监听 `:8090`：

```go
package main

import (
	"encoding/json"
	"flag"
	"image/color"
	"log"
	"net/http"
	"strings"

	captcha "github.com/hicode0101/go-captcha"
)

// svc 全局单例：Service 并发安全
var svc *captcha.Service

// 服务端配置全局唯一引擎；需要按请求调参时参考 example/web/server 的做法
func init() {
	engine := captcha.NewStringEngine(60, 240, 2,
		captcha.LineOptionHollow|captcha.LineOptionSine, 4,
		captcha.CharsetDigits+captcha.CharsetLetters, nil, nil, nil)
	svc = captcha.NewService(engine, captcha.DefaultRepository)
}

type resp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, r resp) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(r)
}

// GET /api/captcha/get → {"id": "...", "b64s": "data:image/png;base64,..."}
func handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, resp{Code: 1, Msg: "method not allowed"})
		return
	}
	id, b64s, _, err := svc.Create()
	if err != nil {
		writeJSON(w, resp{Code: 1, Msg: err.Error()})
		return
	}
	writeJSON(w, resp{Code: 0, Data: map[string]string{"id": id, "b64s": b64s}})
}

// POST /api/captcha/verify {"id":"...","answer":"..."} → {"ok": true|false}
func handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, resp{Code: 1, Msg: "method not allowed"})
		return
	}
	var in struct{ ID, Answer string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, resp{Code: 1, Msg: "bad json"})
		return
	}
	// consume=true：一次性校验，防重放
	ok := svc.Check(strings.TrimSpace(in.ID), in.Answer, true)
	writeJSON(w, resp{Code: 0, Data: map[string]bool{"ok": ok}})
}

func main() {
	addr := flag.String("addr", ":8090", "http listen address")
	flag.Parse()

	// 使用不带方法的路径模式，兼容所有 Go 版本
	//（"GET /path" 方法模式路由需要 go.mod 声明 go 1.22+，否则会静默 404）
	mux := http.NewServeMux()
	mux.HandleFunc("/api/captcha/get", handleGet)
	mux.HandleFunc("/api/captcha/verify", handleVerify)

	log.Printf("captcha api listening on http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

// 显式引用 color 包的示例：固定背景色用法
var _ = color.RGBA{R: 240, G: 240, B: 240, A: 255}
```

运行与验证：

```bash
go run .                        # 监听 http://localhost:8090
curl http://localhost:8090/api/captcha/get
curl -X POST http://localhost:8090/api/captcha/verify \
  -d '{"id":"<上一步的id>","answer":"<用户输入>"}'
```

前端配合：

```html
<!-- 图片验证码 -->
<img id="cap" alt="验证码" />
<script>
  fetch('/api/captcha/get').then(r => r.json()).then(({ data }) => {
    document.getElementById('cap').src = data.b64s
    sessionStorage.setItem('captchaId', data.id)
  })
</script>

<!-- 语音验证码（AudioEngine） -->
<!-- <audio controls src={data.b64s} /> -->
```

## 12. 生产环境建议

1. **防重放**：校验一律 `consume=true`；一次通过后立即失效，失败同样销毁，防止爆破枚举。
2. **答案不下发**：`Create` 返回的 `answer` 只用于调试，生产响应中不要包含。
3. **控制有效时长**：验证码 TTL 建议 5~10 分钟（`NewMemRepository` 的 `ttl` / `NewSyncRepository` 的 `liveTime`）；接入 Redis 时用 `EXPIRE` 管理。
4. **多实例部署**：`DefaultRepository` 是进程内的，多个后端实例无法共享校验状态；水平扩容时必须实现 `Repository` 接口接入 Redis 等集中存储。
5. **并发安全**：`Service` 与两个内置仓库均并发安全，全局持有一个 `Service` 即可；引擎本身出题/渲染无共享可变状态，也可并发使用。
6. **中文场景**：`fontNames` 显式传 `["chinese.ttf"]` 或 `["pinyin-cn.ttc"]`，避免随机抽中西文字体渲染失败/乱码。
7. **参数上限**：`DigitEngine` 对极端参数（画布过小、数字过多）已做钳制不会 panic，但可读性会崩坏；建议长度 ≤ 10、宽高比参考默认 80×240。
8. **内存回收**：内存仓库的回收由写入量触发（默认 10240 条）；低流量服务可调小 `collectThreshold`（如 1024）让过期数据更早释放，或改用 `NewSyncRepository`。
