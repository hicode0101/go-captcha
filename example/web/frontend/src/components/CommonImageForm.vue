<script setup lang="ts">
import type { ColorState, ImageDriverState } from '../types'
import { LINE_OPTIONS, isNoColor } from '../types'
import ParamSlider from './ParamSlider.vue'

const props = defineProps<{
  state: ImageDriverState
  fontList: string[]
  labelPrefix: string
}>()

const colorChannels: { key: keyof ColorState; name: string }[] = [
  { key: 'r', name: 'R' },
  { key: 'g', name: 'G' },
  { key: 'b', name: 'B' },
  { key: 'a', name: 'A' },
]
</script>

<template>
  <ParamSlider
    :label="`${labelPrefix}.Width`"
    v-model="props.state.width"
    :min="80"
    :max="600"
  />
  <ParamSlider
    :label="`${labelPrefix}.Height`"
    v-model="props.state.height"
    :min="40"
    :max="200"
  />
  <ParamSlider
    :label="`${labelPrefix}.NoiseCount`"
    v-model="props.state.noiseCount"
    :min="0"
    :max="20"
  />

  <div class="param-row">
    <div class="param-label">{{ labelPrefix }}.ShowLineOptions</div>
    <n-checkbox-group v-model:value="props.state.showLineOptions" class="param-wide">
      <n-checkbox v-for="o in LINE_OPTIONS" :key="o.value" :value="o.value" :label="o.label" />
    </n-checkbox-group>
  </div>

  <div class="param-row">
    <div class="param-label">{{ labelPrefix }}.Fonts</div>
    <n-checkbox-group v-model:value="props.state.fonts" class="param-wide fonts-group">
      <n-checkbox v-for="f in fontList" :key="f" :value="f" :label="f" />
    </n-checkbox-group>
  </div>

  <div class="param-row" v-for="ch in colorChannels" :key="ch.key">
    <div class="param-label">{{ labelPrefix }}.BgColor.{{ ch.name }}</div>
    <n-slider
      class="param-slider"
      v-model:value="props.state.bgColor[ch.key]"
      :min="0"
      :max="255"
      :step="1"
    />
    <n-input-number
      class="param-number"
      size="small"
      v-model:value="props.state.bgColor[ch.key]"
      :min="0"
      :max="255"
    />
  </div>

  <div class="param-row" v-if="!isNoColor(props.state.bgColor)">
    <div class="param-label">背景色预览</div>
    <span
      class="color-preview"
      :style="{ background: `rgba(${props.state.bgColor.r}, ${props.state.bgColor.g}, ${props.state.bgColor.b}, ${props.state.bgColor.a / 255})` }"
    ></span>
    <span class="color-tip">RGBA 全 0 时后端视为 nil（随机浅色背景）</span>
  </div>
</template>

<style scoped>
.color-preview {
  display: inline-block;
  width: 48px;
  height: 24px;
  border: 1px solid #ccc;
  border-radius: 4px;
  flex: none;
}

.color-tip {
  font-size: 12px;
  color: #999;
}
</style>
