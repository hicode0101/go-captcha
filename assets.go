package captcha

// codeLen 是生成的验证码会话 id 的长度。
const codeLen = 20

// codeAlphabet 是验证码会话 id 允许出现的字符（数字 + 字母）。
var codeAlphabet = []byte(CharsetDigits + CharsetLetters)

const (
	// renderDPI 是绘制图片验证码时 freetype 使用的 DPI。
	renderDPI = 72.0

	// CharsetDigits 是纯数字字符集（剔除易混淆的 5，保持与绘制字形一致）。
	CharsetDigits = "012346789"
	// CharsetLetters 是英文字母字符集（剔除易混淆的 I/l/O 等）。
	CharsetLetters = "ABCDEFGHJKMNOQRSTUVXYZabcdefghjkmnoqrstuvxyz"
	// CharsetSimple 是数字字母混合的精简字符集。
	CharsetSimple = "13467ertyiadfhjkxcvbnERTYADFGHJKXCVBN"
	// CharsetChinese 是常用汉字字符集，可直接作为中文验证码的候选字集。
	CharsetChinese = "的一是在不了有和人这中大为上个国我以要他" +
		"时来用们生到作地于出就分对成会可主发年动" +
		"同工也能下过子说产种面而方后多定行学法所" +
		"民得经十三之进着等部度家电力里如水化高自" +
		"二理起小物现实加量都两体制机当使点从业本" +
		"去把性好应开它合还因由其些然前外天政四日" +
		"那社义事平形相全表间样与关各重新线内数正" +
		"心反你明看原又么利比或但质气第向道命此变" +
		"条只没结解问意建月公无系军很情者最立代想" +
		"已通并提直题党程展五果料象员革位入常文总" +
		"次品式活设及管特件长求老头基资边流路级少" +
		"图山统接知较将组见计别她手角期根论运农指" +
		"几九区强放决西被干做必战先回则任取据处队" +
		"南给色光门即保治北造百规热领七海口东导器" +
		"压志世金增争济阶油思术极交受联什认六共权" +
		"收证改清己美再采转更单风切打白教速花带安" +
		"场身车例真务具万每目至达走积示议声报斗完" +
		"类八离华名确才科张信马节话米整空元况今集" +
		"温传土许步群广石记需段研界拉林律叫且究观" +
		"越织装影算低持音众书布复容儿须际商非验连" +
		"断深难近矿千周委素技备半办青省列习响约支" +
		"般史感劳便团往酸历市克何除消构府称太准精" +
		"值号率族维划选标写存候毛亲快效斯院查江型" +
		"眼王按格养易置派层片始却专状育厂京识适属" +
		"圆包火住调满县局照参红细引听该铁价严龙飞"

	// MimeImagePng 是图片验证码输出 base64 时的 MIME 前缀类型。
	MimeImagePng = "image/png"
	// MimeAudioWav 是语音验证码输出 base64 时的 MIME 前缀类型。
	MimeAudioWav = "audio/wav"
	// CharsetEmoji 是 emoji 表情字符集，可作为 StringEngine 的候选字集。
	CharsetEmoji = "😀😃💯😄🤖😻😅🤣😂🧑🙃😉😊😇😍👴🤩😘😗☺👽♀😙♂😋😛🎨😜🤪😝🤑🤗🤭🤫🤔🤐🤨😐🙉😶😏💗🙄😬🤥😌😪🤤😷🤢🤮🤯😵🤠😎🧐😨😰😱😭😖😡🤬👿☠💀💥💢"
)

// 干扰线选项，按位或组合后赋给引擎的 LineOptions 字段；0 表示不绘制干扰线。
const (
	// LineOptionHollow 绘制一条贯穿画面的粗空心正弦线。
	LineOptionHollow = 2
	// LineOptionSlime 绘制数条随机颜色的斜细线。
	LineOptionSlime = 4
	// LineOptionSine 绘制多条小振幅正弦波浪线。
	LineOptionSine = 8
)
