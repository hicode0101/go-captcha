<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string
    min: number
    max: number
    step?: number
    precision?: number
    modelValue: number
  }>(),
  { step: 1, precision: 0 },
)

const emit = defineEmits<{ (e: 'update:modelValue', v: number): void }>()
</script>

<template>
  <div class="param-row">
    <div class="param-label">{{ label }}</div>
    <n-slider
      class="param-slider"
      :value="modelValue"
      :min="min"
      :max="max"
      :step="step"
      :precision="precision"
      :format-tooltip="(v: number) => v.toFixed(precision)"
      @update:value="(v: number) => emit('update:modelValue', v)"
    />
    <n-input-number
      class="param-number"
      size="small"
      :value="modelValue"
      :min="min"
      :max="max"
      :step="step"
      :precision="precision"
      @update:value="(v: number | null) => emit('update:modelValue', v ?? min)"
    />
  </div>
</template>
