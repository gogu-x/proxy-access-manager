<script setup lang="ts">
import { computed } from 'vue'
import type { UsageGranularity, UsagePoint } from '../api/keys'

const props = defineProps<{ points: UsagePoint[]; granularity: UsageGranularity }>()
const emit = defineEmits<{ change: [value: UsageGranularity] }>()
const ranges: { value: UsageGranularity; label: string }[] = [
  { value: 'minute', label: '分钟' },
  { value: 'hour', label: '小时' },
  { value: 'day', label: '天' },
]
const rangeLabel = computed(() => ({ minute: '最近 60 分钟', hour: '最近 24 小时', day: '最近 30 天' })[props.granularity])
const hasData = computed(() => props.points.some(point => point.bytes > 0))
const max = computed(() => Math.max(1, ...props.points.map(point => point.bytes)))
const x = (index: number) => 40 + index * 580 / Math.max(1, props.points.length - 1)
const y = (bytes: number) => 170 - bytes / max.value * 125
const line = computed(() => props.points.map((point, index) => `${index ? 'L' : 'M'}${x(index).toFixed(1)},${y(point.bytes).toFixed(1)}`).join(' '))
const area = computed(() => props.points.length ? `${line.value} L${x(props.points.length - 1)},170 L40,170 Z` : '')
const format = (bytes: number) => {
  if (bytes >= 1024 ** 3) return `${(bytes / 1024 ** 3).toFixed(2)} GB`
  if (bytes >= 1024 ** 2) return `${(bytes / 1024 ** 2).toFixed(2)} MB`
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${bytes} B`
}
const label = (value: string) => {
  if (!value) return ''
  const date = new Date(value)
  if (props.granularity === 'minute') return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false })
  if (props.granularity === 'hour') return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', hour12: false })
  return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
}
const midpoint = computed(() => props.points[Math.floor((props.points.length - 1) / 2)]?.date || '')
</script>

<template>
  <section class="usage-chart">
    <div class="chart-heading">
      <div><h3>流量使用趋势</h3><p>{{ rangeLabel }}新增流量 · 本地时间</p></div>
      <div class="range-switch" aria-label="统计粒度">
        <button v-for="range in ranges" :key="range.value" type="button" :class="{ active: granularity === range.value }" @click="emit('change', range.value)">{{ range.label }}</button>
      </div>
    </div>
    <div v-if="!hasData" class="chart-empty">暂无历史流量记录</div>
    <div v-else class="chart-body">
      <svg viewBox="0 0 660 215" role="img" :aria-label="`${rangeLabel}流量使用曲线`" preserveAspectRatio="none">
        <defs><linearGradient id="usage-fill" x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stop-color="#4389e6" stop-opacity=".24"/><stop offset="100%" stop-color="#4389e6" stop-opacity="0"/></linearGradient></defs>
        <line x1="40" y1="45" x2="620" y2="45" class="grid-line"/>
        <line x1="40" y1="107" x2="620" y2="107" class="grid-line"/>
        <line x1="40" y1="170" x2="620" y2="170" class="grid-line"/>
        <text x="35" y="49" class="axis-y" text-anchor="end">{{ format(max) }}</text>
        <text x="35" y="174" class="axis-y" text-anchor="end">0</text>
        <path :d="area" fill="url(#usage-fill)"/>
        <path :d="line" fill="none" stroke="#397ee0" stroke-width="3" stroke-linejoin="round" stroke-linecap="round" vector-effect="non-scaling-stroke"/>
        <circle v-for="(point, index) in points" v-show="point.bytes > 0" :key="point.date" :cx="x(index)" :cy="y(point.bytes)" r="3" class="chart-dot"><title>{{ new Date(point.date).toLocaleString('zh-CN') }}：{{ format(point.bytes) }}</title></circle>
        <text x="40" y="200" class="axis-x">{{ label(points[0]?.date || '') }}</text>
        <text x="330" y="200" class="axis-x" text-anchor="middle">{{ label(midpoint) }}</text>
        <text x="620" y="200" class="axis-x" text-anchor="end">{{ label(points[points.length - 1]?.date || '') }}</text>
      </svg>
    </div>
  </section>
</template>

<style scoped>
.usage-chart{padding:0 0 24px;margin-bottom:22px;border-bottom:1px solid #edf1f5}.chart-heading{display:flex;justify-content:space-between;align-items:flex-start;gap:12px;margin-bottom:12px}.chart-heading h3{font-size:14px;color:#223957;margin:0 0 5px}.chart-heading p{font-size:11px;color:#9aa8ba;margin:0}.range-switch{display:flex;gap:2px;padding:3px;background:#f0f4f9;border-radius:8px}.range-switch button{border:0;background:transparent;color:#8191a6;padding:5px 9px;border-radius:6px;font:inherit;font-size:11px;cursor:pointer}.range-switch button.active{background:white;color:#346fc9;box-shadow:0 1px 4px rgba(27,67,113,.12);font-weight:700}.chart-empty{height:185px;display:grid;place-items:center;color:#a3afbf;font-size:12px;background:#f8fafd;border:1px dashed #e4eaf3;border-radius:10px}.chart-body{width:100%;height:210px}.chart-body svg{width:100%;height:100%;overflow:visible}.grid-line{stroke:#e9eef5;stroke-dasharray:4 5}.axis-y,.axis-x{fill:#a0aec0;font:11px sans-serif}.chart-dot{fill:#fff;stroke:#397ee0;stroke-width:2;cursor:crosshair}
</style>
