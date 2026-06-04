<script setup lang="ts">
/**
 * 全平台统一标签组件
 * 所有页面的标签展示必须使用此组件，确保视觉一致性
 */
import { computed } from 'vue'
import { getTagStyle, getRecommendationStyle, getMacdSignalStyle } from '@/utils/tagColors'

const props = withDefaults(defineProps<{
  /** 标签类型 */
  type?: 'highlight' | 'recommendation' | 'macdSignal' | 'bollPosition' | 'signalType' | 'score'
  /** 标签文本/值 */
  text: string
  /** 可选的大小变体 */
  size?: 'sm' | 'md'
  /** 分数值（type='score'时使用） */
  score?: number
}>(), {
  type: 'highlight',
  size: 'sm',
  score: 0,
})

const styleClass = computed(() => {
  switch (props.type) {
    case 'recommendation': {
      const s = getRecommendationStyle(props.text)
      return `${s.bg} ${s.text} ${s.border}`
    }
    case 'macdSignal': {
      const s = getMacdSignalStyle(props.text)
      return `${s.bg} ${s.text} ${s.border}`
    }
    case 'bollPosition': {
      // BOLL位置颜色
      const bollMap: Record<string, string> = {
        '突破上轨': 'text-red-500 font-medium',
        '上轨区域': 'text-amber-600',
        '中轨上方': 'text-blue-600',
        '中轨下方': 'text-slate-500',
        '下轨区域': 'text-slate-500',
        '跌破下轨': 'text-green-600',
      }
      return bollMap[props.text] || 'text-slate-400'
    }
    case 'signalType': {
      const sigMap: Record<string, string> = {
        'sell': 'bg-red-100 text-red-600 border border-red-200',
        'buy': 'bg-green-100 text-green-600 border border-green-200',
        'hold': 'bg-blue-100 text-blue-600 border border-blue-200',
        'warning': 'bg-orange-100 text-orange-600 border border-orange-200',
      }
      return sigMap[props.text] || 'bg-slate-100 text-slate-500 border border-slate-200'
    }
    case 'score': {
      if (props.score >= 70) return 'bg-red-100 text-red-700 border border-red-200'
      if (props.score >= 55) return 'bg-amber-100 text-amber-700 border border-amber-200'
      if (props.score >= 40) return 'bg-blue-100 text-blue-700 border border-blue-200'
      return 'bg-slate-100 text-slate-600 border border-slate-200'
    }
    default: {
      const s = getTagStyle(props.text)
      return `${s.bg} ${s.text} ${s.border}`
    }
  }
})

const sizeClass = computed(() => {
  return props.size === 'md'
    ? 'text-sm px-2.5 py-1'
    : 'text-xs px-1.5 py-0.5'
})
</script>

<template>
  <span :class="['inline-flex items-center rounded-full font-medium border', sizeClass, styleClass]">
    <slot>{{ text }}</slot>
  </span>
</template>
