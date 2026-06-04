<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getMarketStatus, scanAllAShares } from '@/api'
import type { MarketData, IndexData, AllStockScanResult, RankStockItem } from '@/types/stock'
import StockDetailSheet from '@/components/StockDetailSheet.vue'
import StockTag from '@/components/StockTag.vue'
import { useRouter } from 'vue-router'
import {
  Zap, RefreshCw, TrendingUp, TrendingDown, BarChart3,
  ChevronRight, Search, Star, ArrowUpRight, ArrowDownRight,
  Activity, Eye, Crown, Award, Settings
} from 'lucide-vue-next'

interface IndexCardItem {
  name: string
  key: keyof MarketData['indices']
  prefix: string
}

const INDEX_CARDS: IndexCardItem[] = [
  { name: '上证指数', key: 'shanghai', prefix: 'SH' },
  { name: '深证成指', key: 'shenzhen', prefix: 'SZ' },
  { name: '创业板指', key: 'chinext', prefix: 'CY' },
  { name: '科创50', key: 'star50', prefix: 'KC' },
]

const router = useRouter()
const marketStatus = ref<'safe' | 'warning' | 'danger'>('safe')
const rawMarketData = ref<MarketData | null>(null)
const indicesLoading = ref(true)
const scanResult = ref<AllStockScanResult | null>(null)
const topStocks = ref<RankStockItem[]>([])
const loading = ref(false)
const detailOpen = ref(false)
const selectedCode = ref('')

onMounted(() => {
  loadMarketData()
  handleScanAll()
})

const loadMarketData = async () => {
  indicesLoading.value = true
  try {
    const res = await getMarketStatus()
    if (res.code === 200 && res.data) {
      const data = res.data as MarketData
      marketStatus.value = data.status
      rawMarketData.value = data
    }
  } catch (error) {
    console.error('获取大盘数据失败:', error)
  } finally {
    indicesLoading.value = false
  }
}

function getIndexData(key: keyof MarketData['indices']): IndexData | null {
  return rawMarketData.value?.indices?.[key] ?? null
}

const handleScanAll = async () => {
  loading.value = true
  try {
    const res = await scanAllAShares()
    if (res.code === 200 && res.data) {
      scanResult.value = res.data as AllStockScanResult
      topStocks.value = (res.data as AllStockScanResult).topList || []
    }
  } catch (error) {
    console.error('全A股扫描失败:', error)
  } finally {
    loading.value = false
  }
}

const getStatusColor = (status: string) => {
  switch (status) {
    case 'safe': return 'bg-red-500'
    case 'warning': return 'bg-yellow-500'
    case 'danger': return 'bg-green-500'
    default: return 'bg-gray-500'
  }
}

const getStatusText = (status: string) => {
  switch (status) {
    case 'safe': return '偏多'
    case 'warning': return '震荡'
    case 'danger': return '偏空'
    default: return '未知'
  }
}

const openDetail = (code: string) => {
  selectedCode.value = code
  detailOpen.value = true
}

const goToRank = () => {
  router.push('/rank')
}

const formatCost = (ms: number) => {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

const top10 = computed(() => topStocks.value.slice(0, 10))
</script>

<template>
  <div class="space-y-4">
    <!-- 大盘指数卡片 -->
    <div class="flex items-center justify-between mb-1">
      <div class="flex items-center gap-2">
        <Activity class="w-4 h-4 text-slate-500" />
        <span class="text-xs font-medium text-slate-400 uppercase tracking-wide">核心市场指数</span>
      </div>
      <button class="w-7 h-7 flex items-center justify-center rounded-lg hover:bg-slate-100 text-slate-300 hover:text-slate-500 transition-colors">
        <Settings class="w-4 h-4" />
      </button>
    </div>

    <!-- 骨架屏 -->
    <div v-if="indicesLoading" class="flex gap-3 overflow-x-auto pb-1" style="scrollbar-width:none;-ms-overflow-style:none">
      <div
        v-for="n in 4"
        :key="n"
        class="bg-white rounded-xl p-3.5 shadow-sm border border-slate-100 shrink-0 w-[152px] animate-pulse"
      >
        <div class="h-3 bg-slate-200 rounded w-14 mb-2"></div>
        <div class="h-6 bg-slate-200 rounded w-20 mb-2"></div>
        <div class="h-3 bg-slate-100 rounded w-16"></div>
      </div>
    </div>

    <!-- 实数据 -->
    <div v-else class="flex gap-3 overflow-x-auto pb-1 -mx-1 px-1" style="scrollbar-width:none;-ms-overflow-style:none">
      <div
        v-for="card in INDEX_CARDS"
        :key="card.key"
        class="bg-white rounded-xl p-3.5 shadow-sm border border-slate-100 shrink-0 w-[152px] hover:shadow-md hover:border-blue-200 transition-all cursor-pointer"
      >
        <div class="text-xs text-slate-400 font-medium mb-1.5 truncate">{{ card.name }}</div>
        <div class="text-lg font-bold text-slate-900 mb-1">
          {{ getIndexData(card.key)?.price ? getIndexData(card.key)!.price.toFixed(2) : '--' }}
        </div>
        <div
          v-if="getIndexData(card.key)?.price"
          :class="[
            'text-xs font-medium flex items-center gap-0.5 whitespace-nowrap',
            (getIndexData(card.key)?.changePercent ?? 0) >= 0 ? 'text-red-500' : 'text-green-500'
          ]"
        >
          <ArrowUpRight v-if="(getIndexData(card.key)?.changePercent ?? 0) >= 0" class="w-3 h-3" />
          <ArrowDownRight v-else class="w-3 h-3" />
          {{ (getIndexData(card.key)?.changePercent ?? 0) >= 0 ? '+' : '' }}{{ getIndexData(card.key)?.changePercent?.toFixed(2) ?? '0.00' }}%
        </div>
      </div>
    </div>

    <!-- 市场状态 -->
    <div class="bg-white rounded-xl p-4 shadow-sm border border-slate-100">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div :class="['w-3 h-3 rounded-full', getStatusColor(marketStatus)]" />
          <div>
            <div class="text-sm text-slate-500">市场情绪</div>
            <div
              :class="[
                'text-xl font-bold',
                marketStatus === 'safe' ? 'text-red-600' : marketStatus === 'danger' ? 'text-green-600' : 'text-yellow-600'
              ]"
            >
              {{ getStatusText(marketStatus) }}
            </div>
          </div>
        </div>
        <div class="text-right text-xs text-slate-400">
          {{ rawMarketData?.time ? new Date(rawMarketData.time).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) : '--' }}
        </div>
      </div>
      <!-- 市场情绪阈值说明 -->
      <div class="mt-2 flex items-center gap-3 text-xs text-slate-400">
        <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-red-500"></span>偏多（涨≥0.5%）</span>
        <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-yellow-500"></span>震荡</span>
        <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-green-500"></span>偏空（跌≥0.5%）</span>
      </div>
    </div>

    <!-- 市场偏空风险提示 -->
    <div v-if="marketStatus === 'danger'" class="bg-red-50 border border-red-200 rounded-xl p-3 flex items-start gap-2">
      <svg class="w-4 h-4 text-red-500 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" /></svg>
      <div class="text-sm text-red-700 leading-relaxed">
        <span class="font-semibold">⚠ 市场偏空风险提示：</span>当前上证指数跌幅超过0.5%，大盘走势偏弱。建议谨慎操作，控制仓位，所有评分与推荐仅供参考，不构成投资建议。
      </div>
    </div>

    <!-- 全A股扫描统计 -->
    <div
      v-if="scanResult"
      class="bg-gradient-to-br from-blue-600 via-indigo-600 to-purple-700 text-white rounded-xl p-5 shadow-lg relative overflow-hidden"
    >
      <!-- 背景装饰 -->
      <div class="absolute top-0 right-0 w-32 h-32 bg-white/5 rounded-full -translate-y-1/2 translate-x-1/2" />
      <div class="absolute bottom-0 left-0 w-24 h-24 bg-white/5 rounded-full translate-y-1/2 -translate-x-1/2" />
      
      <div class="relative">
        <div class="flex items-center gap-2 mb-4">
          <Zap class="w-5 h-5 text-yellow-300" />
          <span class="text-base font-bold">全A股扫描</span>
          <span class="text-xs bg-white/20 px-2 py-0.5 rounded-full ml-auto">
            {{ formatCost(scanResult.costMs) }}
          </span>
        </div>

        <div class="grid grid-cols-3 gap-3">
          <div class="bg-white/10 rounded-lg p-3 text-center backdrop-blur-sm">
            <div class="text-2xl font-bold text-yellow-300">{{ scanResult.totalStocks }}</div>
            <div class="text-xs opacity-80 mt-1">扫描总数</div>
          </div>
          <div class="bg-white/10 rounded-lg p-3 text-center backdrop-blur-sm">
            <div class="text-2xl font-bold text-green-300">{{ scanResult.validStocks }}</div>
            <div class="text-xs opacity-80 mt-1">有效行情</div>
          </div>
          <div class="bg-white/10 rounded-lg p-3 text-center backdrop-blur-sm">
            <div class="text-2xl font-bold text-blue-300">{{ scanResult.analyzedStocks }}</div>
            <div class="text-xs opacity-80 mt-1">分析完成</div>
          </div>
        </div>

        <div class="text-xs text-white/60 text-center mt-3">
          扫描时间: {{ scanResult.scanTime }}
        </div>
      </div>
    </div>

    <!-- 全A股扫描按钮 -->
    <button
      class="w-full flex items-center justify-center gap-2 py-3.5 rounded-xl font-medium text-white transition-all shadow-sm"
      :class="loading ? 'bg-slate-400 cursor-not-allowed' : 'bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 active:scale-[0.98]'"
      :disabled="loading"
      @click="handleScanAll"
    >
      <RefreshCw v-if="loading" class="w-5 h-5 animate-spin" />
      <Zap v-else class="w-5 h-5" />
      {{ loading ? '正在扫描全A股(约5000支)...' : '扫描全A股' }}
    </button>

    <!-- Top 10 排名 -->
    <div v-if="top10.length > 0">
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-2">
          <Crown class="w-5 h-5 text-yellow-500" />
          <span class="text-lg font-bold text-slate-900">综合评分 Top 10</span>
          <span class="text-xs text-slate-400 hidden sm:inline">纯技术面打分，不包含基本面</span>
        </div>
        <button
          class="flex items-center gap-1 text-sm text-blue-600 hover:text-blue-700 font-medium"
          @click="goToRank"
        >
          完整排名
          <ChevronRight class="w-4 h-4" />
        </button>
      </div>

      <div class="space-y-2">
        <div
          v-for="(stock, index) in top10"
          :key="stock.code"
          class="bg-white rounded-xl p-3.5 shadow-sm border border-slate-100 cursor-pointer hover:shadow-md hover:border-blue-200 transition-all"
          @click="openDetail(stock.code)"
        >
          <div class="flex items-center gap-3">
            <!-- 排名 -->
            <div
              :class="[
                'w-8 h-8 rounded-lg flex items-center justify-center text-sm font-bold shrink-0',
                index === 0 ? 'bg-yellow-400 text-yellow-900' :
                index === 1 ? 'bg-slate-300 text-slate-700' :
                index === 2 ? 'bg-amber-600 text-white' :
                'bg-slate-100 text-slate-500'
              ]"
            >
              {{ index + 1 }}
            </div>

            <!-- 股票信息 -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-bold text-slate-900 truncate">{{ stock.name }}</span>
                <span class="text-xs text-slate-400 shrink-0">{{ stock.code }}</span>
                <span v-if="stock.industry" class="text-xs text-blue-400 shrink-0">{{ stock.industry }}</span>
              </div>
              <div class="flex items-center gap-2 mt-1">
                <!-- 推荐等级 -->
                <StockTag
                  v-if="stock.recommendation"
                  type="recommendation"
                  :text="stock.recommendation"
                />
                <StockTag
                  v-else
                  type="score"
                  :score="stock.totalScore"
                  :text="`综合评分 ${stock.totalScore.toFixed(0)}`"
                />
                <!-- 亮点标签 -->
                <StockTag
                  v-for="h in stock.highlights.slice(0, 2)"
                  :key="h"
                  type="highlight"
                  :text="h"
                />
              </div>
            </div>

            <!-- 价格与涨跌 -->
            <div class="text-right shrink-0">
              <div class="font-bold text-slate-900">¥{{ stock.price.toFixed(2) }}</div>
              <div
                :class="[
                  'text-sm font-medium flex items-center justify-end gap-0.5',
                  stock.changePercent >= 0 ? 'text-red-500' : 'text-green-500'
                ]"
              >
                <ArrowUpRight v-if="stock.changePercent >= 0" class="w-3 h-3" />
                <ArrowDownRight v-else class="w-3 h-3" />
                {{ stock.changePercent >= 0 ? '+' : '' }}{{ stock.changePercent.toFixed(2) }}%
              </div>
            </div>

            <!-- 评分条 -->
            <div class="w-16 shrink-0">
              <div class="text-xs text-slate-400 text-center mb-1">综合评分</div>
              <div class="text-xs text-slate-500 text-center mb-1 font-bold">{{ stock.totalScore.toFixed(1) }}分</div>
              <div class="h-2 bg-slate-100 rounded-full overflow-hidden">
                <div
                  :class="[
                    'h-full rounded-full transition-all',
                    stock.totalScore >= 60 ? 'bg-red-500' :
                    stock.totalScore >= 40 ? 'bg-blue-500' : 'bg-slate-400'
                  ]"
                  :style="{ width: Math.min(stock.totalScore, 100) + '%' }"
                />
              </div>
            </div>
          </div>

          <!-- 指标详情 -->
          <div class="grid grid-cols-4 gap-2 mt-2.5 pt-2.5 border-t border-slate-50">
            <div class="text-center">
              <div class="text-xs text-slate-400">趋势</div>
              <div class="text-sm font-semibold text-slate-700">{{ stock.trendScore.toFixed(1) }}</div>
            </div>
            <div class="text-center">
              <div class="text-xs text-slate-400">动量</div>
              <div class="text-sm font-semibold text-slate-700">{{ stock.momentumScore.toFixed(1) }}</div>
            </div>
            <div class="text-center">
              <div class="text-xs text-slate-400">量能</div>
              <div class="text-sm font-semibold text-slate-700">{{ stock.volumeScore.toFixed(1) }}</div>
            </div>
            <div class="text-center">
              <div class="text-xs text-slate-400">技术</div>
              <div class="text-sm font-semibold text-slate-700">{{ stock.techScore.toFixed(1) }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="!loading && !scanResult" class="text-center py-12 text-slate-400">
      <BarChart3 class="w-12 h-12 mx-auto mb-3 text-slate-300" />
      <div class="text-base font-medium">点击上方按钮扫描全A股</div>
      <div class="text-sm mt-1">约5000支股票的综合评分与排名</div>
    </div>

    <!-- 股票详情弹窗 -->
    <StockDetailSheet
      :open="detailOpen"
      :code="selectedCode"
      @update:open="detailOpen = $event"
    />
  </div>
</template>
