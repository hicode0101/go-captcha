<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { fetchFonts, generateCaptcha, verifyCaptcha } from './api'
import { buildGoCode } from './goCode'
import type { DigitState, AudioState, GoCodeStates } from './goCode'
import type { DriverKey, GenerateParams, ImageDriverState } from './types'
import { AUDIO_LANGUAGES, DRIVER_TABS, isNoColor, sumLineOptions } from './types'
import CommonImageForm from './components/CommonImageForm.vue'
import CodeBox from './components/CodeBox.vue'
import ParamSlider from './components/ParamSlider.vue'

const message = useMessage()

// ---------- 各 driver 的参数状态（默认值与官方示例一致） ----------
const digit = reactive<DigitState>({ length: 5, width: 240, height: 80, maxSkew: 0.7, dotCount: 80 })
const stringState = reactive<ImageDriverState & { length: number; source: string }>({
  length: 6,
  source: '23456789qwertyuipkjhgfdsazxcvbnm',
  width: 240,
  height: 60,
  noiseCount: 0,
  showLineOptions: [],
  fonts: ['actionj.ttf'],
  bgColor: { r: 255, g: 255, b: 255, a: 255 },
})
const mathState = reactive<ImageDriverState>({
  width: 240,
  height: 60,
  noiseCount: 0,
  showLineOptions: [],
  fonts: ['actionj.ttf'],
  bgColor: { r: 255, g: 255, b: 255, a: 255 },
})
const chineseState = reactive<ImageDriverState & { length: number; source: string }>({
  length: 2,
  source: '大道,之行,天下,为公,选,贤,与,能,讲,信,修,睦,亲其,子其,老终,壮,用,幼,长,有养,有分,有归,藏己,谋闭,外户,大同',
  width: 320,
  height: 60,
  noiseCount: 0,
  showLineOptions: [],
  fonts: ['chinese.ttf'],
  bgColor: { r: 255, g: 255, b: 255, a: 255 },
})
const audio = reactive<AudioState>({ length: 6, language: 'zh' })

// ---------- 字体列表（服务端 /api/fonts 实时下发，随 fonts 目录自动更新） ----------
const fontList = ref<string[]>([])

/**
 * 从实际字体列表中挑选默认勾选：
 * prefer 依次取第一个仍存在的名字；全部缺失时中文驱动优先挑中文字体
 * （文件名含 chinese/pinyin 或 .ttc），最后兜底列表第一个。
 */
function pickDefaultFonts(list: string[], prefer: string[], chinese = false): string[] {
  if (list.length === 0) return []
  for (const name of prefer) {
    if (list.includes(name)) return [name]
  }
  if (chinese) {
    const zh = list.find((n) => /chinese|pinyin|\.ttc$/i.test(n))
    if (zh) return [zh]
  }
  const first = list[0]
  return first ? [first] : []
}

// ---------- 验证码状态 ----------
const activeDriver = ref<DriverKey>('digit')
const captchaId = ref('')
const captchaB64 = ref('')
const loading = ref(false)
const verifying = ref(false)
const answer = ref('')
const verifyResult = ref<boolean | null>(null)

const isImage = computed(() => captchaB64.value.startsWith('data:image'))

function toNullableColor(c: { r: number; g: number; b: number; a: number }) {
  return isNoColor(c) ? null : { ...c }
}

/** 当前参数对应的生成请求 */
const currentRequest = computed<GenerateParams>(() => {
  switch (activeDriver.value) {
    case 'digit':
      return {
        driver: 'digit',
        height: digit.height,
        width: digit.width,
        length: digit.length,
        maxSkew: digit.maxSkew,
        dotCount: digit.dotCount,
        noiseCount: 0,
        showLineOptions: 0,
        source: '',
        language: '',
        bgColor: null,
        fonts: [],
      }
    case 'string':
      return {
        driver: 'string',
        height: stringState.height,
        width: stringState.width,
        length: stringState.length,
        maxSkew: 0,
        dotCount: 0,
        noiseCount: stringState.noiseCount,
        showLineOptions: sumLineOptions(stringState.showLineOptions),
        source: stringState.source,
        language: '',
        bgColor: toNullableColor(stringState.bgColor),
        fonts: [...stringState.fonts],
      }
    case 'math':
      return {
        driver: 'math',
        height: mathState.height,
        width: mathState.width,
        length: 0,
        maxSkew: 0,
        dotCount: 0,
        noiseCount: mathState.noiseCount,
        showLineOptions: sumLineOptions(mathState.showLineOptions),
        source: '',
        language: '',
        bgColor: toNullableColor(mathState.bgColor),
        fonts: [...mathState.fonts],
      }
    case 'chinese':
      return {
        driver: 'chinese',
        height: chineseState.height,
        width: chineseState.width,
        length: chineseState.length,
        maxSkew: 0,
        dotCount: 0,
        noiseCount: chineseState.noiseCount,
        showLineOptions: sumLineOptions(chineseState.showLineOptions),
        source: chineseState.source,
        language: '',
        bgColor: toNullableColor(chineseState.bgColor),
        fonts: [...chineseState.fonts],
      }
    case 'audio':
      return {
        driver: 'audio',
        height: 0,
        width: 0,
        length: audio.length,
        maxSkew: 0,
        dotCount: 0,
        noiseCount: 0,
        showLineOptions: 0,
        source: '',
        language: audio.language,
        bgColor: null,
        fonts: [],
      }
  }
})

const goCode = computed(() => buildGoCode(activeDriver.value, { digit, string: stringState, math: mathState, chinese: chineseState, audio }))

// ---------- 生成 / 校验 ----------
async function refresh() {
  loading.value = true
  try {
    const data = await generateCaptcha(currentRequest.value)
    captchaId.value = data.id
    captchaB64.value = data.b64s
    answer.value = ''
    verifyResult.value = null
  } catch (e) {
    message.error(e instanceof Error ? e.message : '生成验证码失败')
  } finally {
    loading.value = false
  }
}

async function doVerify() {
  if (!captchaId.value) {
    message.warning('请先生成验证码')
    return
  }
  if (!answer.value.trim()) {
    message.warning('请输入验证码')
    return
  }
  verifying.value = true
  try {
    const data = await verifyCaptcha(captchaId.value, answer.value)
    verifyResult.value = data.ok
    if (data.ok) {
      message.success('验证成功')
      setTimeout(refresh, 900)
    } else {
      message.error('验证失败')
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : '校验请求失败')
  } finally {
    verifying.value = false
  }
}

// 参数变化后自动刷新（防抖）
let refreshTimer: ReturnType<typeof setTimeout> | null = null
watch(currentRequest, () => {
  if (refreshTimer) clearTimeout(refreshTimer)
  refreshTimer = setTimeout(refresh, 500)
})

onMounted(async () => {
  refresh()
  try {
    const list = await fetchFonts()
    fontList.value = list
    // 默认勾选随实际字体列表自动校正，避免勾到已从 fonts 目录删除的字体
    stringState.fonts = pickDefaultFonts(list, ['actionj.ttf'])
    mathState.fonts = pickDefaultFonts(list, ['actionj.ttf'])
    chineseState.fonts = pickDefaultFonts(list, ['chinese.ttf', 'pinyin-cn.ttc'], true)
  } catch {
    // 获取失败时保留当前勾选，由用户手动选择
  }
})
</script>

<template>
  <div>
    <!-- 顶部：验证码展示 + 校验 -->
    <header class="captcha-band">
      <div class="captcha-view" title="点击刷新验证码" @click="refresh">
        <n-spin :show="loading" size="small">
          <img v-if="isImage" class="captcha-img" :src="captchaB64" alt="captcha" />
          <audio v-else-if="captchaB64" class="captcha-audio" controls :src="captchaB64" @click.stop></audio>
          <div v-else class="captcha-empty">正在生成验证码…</div>
        </n-spin>
      </div>
      <div class="verify-col">
        <n-input
          v-model:value="answer"
          size="large"
          placeholder="input your captcha numbers"
          :disabled="!captchaId"
          clearable
          @keyup.enter="doVerify"
        />
        <n-button type="primary" size="large" :loading="verifying" @click="doVerify">Verify Captcha</n-button>
        <div class="verify-row">
          <n-tag v-if="verifyResult === true" type="success" size="small">验证成功</n-tag>
          <n-tag v-else-if="verifyResult === false" type="error" size="small">验证失败</n-tag>
          <n-button quaternary size="tiny" @click="refresh">刷新验证码</n-button>
        </div>
      </div>
    </header>

    <main class="page">
      <!-- 参数调试区 -->
      <n-card size="small">
        <n-tabs v-model:value="activeDriver" type="line" animated>
          <n-tab-pane name="digit" tab="DriverDigit">
            <div class="form">
              <ParamSlider label="DriverDigit.Length" v-model="digit.length" :min="1" :max="10" />
              <ParamSlider label="DriverDigit.Width" v-model="digit.width" :min="80" :max="600" />
              <ParamSlider label="DriverDigit.Height" v-model="digit.height" :min="40" :max="200" />
              <ParamSlider label="DriverDigit.MaxSkew" v-model="digit.maxSkew" :min="0" :max="1" :step="0.1" :precision="1" />
              <ParamSlider label="DriverDigit.DotCount" v-model="digit.dotCount" :min="0" :max="200" />
            </div>
          </n-tab-pane>

          <n-tab-pane name="string" tab="DriverString">
            <div class="form">
              <ParamSlider label="DriverString.Length" v-model="stringState.length" :min="1" :max="10" />
              <div class="param-row">
                <div class="param-label">DriverString.Source</div>
                <n-input
                  v-model:value="stringState.source"
                  class="param-wide"
                  type="textarea"
                  placeholder="候选字符集"
                  :autosize="{ minRows: 1, maxRows: 4 }"
                />
              </div>
              <CommonImageForm label-prefix="DriverString" :state="stringState" :font-list="fontList" />
            </div>
          </n-tab-pane>

          <n-tab-pane name="math" tab="DriverMath">
            <div class="form">
              <CommonImageForm label-prefix="DriverMath" :state="mathState" :font-list="fontList" />
            </div>
          </n-tab-pane>

          <n-tab-pane name="chinese" tab="DriverChinese">
            <div class="form">
              <ParamSlider label="DriverChinese.Length" v-model="chineseState.length" :min="1" :max="6" />
              <div class="param-row">
                <div class="param-label">DriverChinese.Source</div>
                <n-input
                  v-model:value="chineseState.source"
                  class="param-wide"
                  type="textarea"
                  placeholder="候选汉字，逗号分隔；无逗号则按字符集随机"
                  :autosize="{ minRows: 1, maxRows: 4 }"
                />
              </div>
              <CommonImageForm label-prefix="DriverChinese" :state="chineseState" :font-list="fontList" />
            </div>
          </n-tab-pane>

          <n-tab-pane name="audio" tab="DriverAudio">
            <div class="form">
              <ParamSlider label="DriverAudio.Length" v-model="audio.length" :min="1" :max="10" />
              <div class="param-row">
                <div class="param-label">DriverAudio.Language</div>
                <n-radio-group v-model:value="audio.language">
                  <n-radio-button v-for="lang in AUDIO_LANGUAGES" :key="lang" :value="lang" :label="lang" />
                </n-radio-group>
              </div>
            </div>
          </n-tab-pane>
        </n-tabs>
      </n-card>

      <!-- 代码预览 -->
      <div style="margin-top: 20px">
        <CodeBox :code="goCode" @copied="message.success('代码已复制到剪贴板')" />
      </div>
    </main>
  </div>
</template>

<style scoped>
:deep(.n-tabs-tab) {
  font-weight: 600;
}
</style>
