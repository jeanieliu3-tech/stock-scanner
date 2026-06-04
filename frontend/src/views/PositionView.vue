<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { getBatchQuotes, searchStocks as searchStocksApi, getStockScores, getPositionHealthScore } from '@/api'
import type { StockScore } from '@/types/stock'
import StockDetailSheet from '@/components/StockDetailSheet.vue'
import StockTag from '@/components/StockTag.vue'

interface Position {
  id: string
  code: string
  name: string
  quantity: number
  costPrice: number
  addTime: number
}

interface PositionDisplay extends Position {
  currentPrice: number
  changePercent: number
  marketValue: number
  profit: number
  profitPercent: number
  signal?: string
  signalType?: 'sell' | 'hold' | 'buy'
  advice?: string
  totalScore?: number
  marketBaseScore?: number
  techBonusScore?: number
  trendScore?: number
  momentumScore?: number
  volumeScore?: number
  techScore?: number
  macdSignal?: string
  bollPosition?: string
  isGoldenCross?: boolean
  isAboveWater?: boolean
  highlights?: string[]
  recommendation?: string
  // 持仓健康度
  positionHealthScore?: number
  positionHealthLabel?: string
}

const STORAGE_KEY = 'stock_position_list'
const STOP_LOSS_KEY = 'stop_loss_percent'

const positions = ref<PositionDisplay[]>([])
const isAddDialogOpen = ref(false)
const isSearchDialogOpen = ref(false)
const isEditDialogOpen = ref(false)
const selectedPosition = ref<PositionDisplay | null>(null)
const isDetailSheetOpen = ref(false)
const detailCode = ref('')
const detailCostPrice = ref(0)
const form = ref({ code: '', name: '', quantity: '', costPrice: '' })
const editForm = ref({ code: '', name: '', quantity: '', costPrice: '' })
const searchResults = ref<Array<{ code: string; name: string }>>([])
const searchLoading = ref(false)
const isTradingSession = ref(false)
let tradingInterval: ReturnType<typeof setInterval> | null = null
let refreshInterval: ReturnType<typeof setInterval> | null = null

const loadPositions = (): Position[] => {
  try {
    const data = localStorage.getItem(STORAGE_KEY)
    return data ? JSON.parse(data) : []
  } catch { return [] }
}

const savePositions = (list: Position[]) => {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list))
}

const getStopLossPercent = (): number => {
  const val = localStorage.getItem(STOP_LOSS_KEY)
  return val ? parseFloat(val) : -8
}

const isTradingTime = (): boolean => {
  const now = new Date()
  const day = now.getDay()
  const hour = now.getHours()
  const minute = now.getMinutes()
  if (day === 0 || day === 6) return false
  if ((hour === 9 && minute >= 30) || (hour >= 10 && hour < 12)) return true
  if (hour >= 13 && hour < 15) return true
  return false
}

interface TradeSignal {
  signal: string
  type: 'SELL' | 'BUY' | 'HOLD' | 'WARNING'
  level: 'HIGH' | 'MEDIUM' | 'LOW'
  title: string
  reason: string
}

const generateSignal = (price: number, costPrice: number, changePercent: number, stopLoss: number): TradeSignal | null => {
  const profitPercent = ((price - costPrice) / costPrice) * 100
  if (profitPercent <= stopLoss) return { signal: '止损出局', type: 'WARNING', level: 'HIGH', title: '触发止损', reason: `亏损${profitPercent.toFixed(1)}%，超过${stopLoss}%止损线` }
  if (profitPercent >= 15 && changePercent > 0) return { signal: '建议止盈', type: 'SELL', level: 'MEDIUM', title: '高位止盈', reason: `盈利${profitPercent.toFixed(1)}%，可分批锁定利润` }
  if (changePercent <= -3) return { signal: '注意风险', type: 'WARNING', level: 'LOW', title: '注意风险', reason: `今日下跌${changePercent.toFixed(1)}%，关注是否企稳` }
  if (profitPercent > 0) return { signal: '持仓盈利', type: 'HOLD', level: 'LOW', title: '建议持有', reason: `盈利${profitPercent.toFixed(1)}%，趋势向好` }
  if (profitPercent > -5) return { signal: '轻仓观望', type: 'HOLD', level: 'LOW', title: '轻仓观望', reason: '小幅亏损，继续观察' }
  return null
}

const loadPositionData = async () => {
  try {
    const stored = loadPositions()
    const codes = stored.map(p => p.code)
    if (codes.length > 0) {
      const res = await getBatchQuotes(codes)
      if (res.code === 200) {
        const quotes = (res.data || {}) as Record<string, { price: string; changePercent: string }>
        const stopLoss = getStopLossPercent()
        const displays: PositionDisplay[] = stored.map(pos => {
          const quote = quotes[pos.code] || {}
          const currentPrice = parseFloat(quote.price) || 0
          const changePercent = parseFloat(quote.changePercent) || 0
          const marketValue = currentPrice * pos.quantity
          const costValue = pos.costPrice * pos.quantity
          const profit = marketValue - costValue
          const profitPercent = costValue > 0 ? (profit / costValue) * 100 : 0
          const signal = generateSignal(currentPrice, pos.costPrice, changePercent, stopLoss)
          return {
            ...pos, currentPrice, changePercent, marketValue, profit, profitPercent,
            signal: signal?.signal,
            signalType: signal?.type === 'WARNING' || signal?.type === 'SELL' ? 'sell' : signal?.type === 'BUY' ? 'buy' : 'hold' as const,
            advice: signal?.reason,
          }
        })
        positions.value = displays
        loadStockScores(codes)
      }
    } else {
      positions.value = []
    }
  } catch (err) {
    console.error('加载失败:', err)
  }
}

const loadStockScores = async (codes: string[]) => {
  try {
    // 批量获取市场评分（统一数据源）
    const res = await getStockScores(codes)
    if (res.code === 200) {
      const scores = (res.data || {}) as Record<string, StockScore>
      positions.value = positions.value.map(pos => {
        const score = scores[pos.code]
        if (score) {
          return {
            ...pos,
            totalScore: score.totalScore,
            marketBaseScore: score.marketBaseScore,
            techBonusScore: score.techBonusScore,
            trendScore: score.trendScore,
            momentumScore: score.momentumScore,
            volumeScore: score.volumeScore,
            techScore: score.techScore,
            macdSignal: score.macdSignal,
            bollPosition: score.bollPosition,
            isGoldenCross: score.isGoldenCross,
            isAboveWater: score.isAboveWater,
            highlights: score.highlights || [],
            recommendation: score.recommendation,
          }
        }
        return pos
      })
    }

    // 逐只获取持仓健康度（含成本价个性化调整）
    for (const pos of positions.value) {
      if (pos.costPrice > 0) {
        try {
          const healthRes = await getPositionHealthScore(pos.code, pos.costPrice)
          if (healthRes.code === 200 && healthRes.data) {
            const health = healthRes.data as StockScore
            pos.positionHealthScore = health.positionHealthScore
            pos.positionHealthLabel = health.positionHealthLabel
          }
        } catch { /* skip individual failures */ }
      }
    }
  } catch (err) {
    console.error('获取评分失败:', err)
  }
}

const refreshPositionPrices = async () => {
  if (positions.value.length === 0) return
  try {
    const codes = positions.value.map(p => p.code)
    const res = await getBatchQuotes(codes)
    if (res.code === 200) {
      const quotes = (res.data || {}) as Record<string, { price: string; changePercent: string }>
      const stopLoss = getStopLossPercent()
      positions.value = positions.value.map(pos => {
        const quote = quotes[pos.code] || {}
        const currentPrice = parseFloat(quote.price) || pos.currentPrice
        const changePercent = parseFloat(quote.changePercent) || pos.changePercent
        const marketValue = currentPrice * pos.quantity
        const costValue = pos.costPrice * pos.quantity
        const profit = marketValue - costValue
        const profitPercent = costValue > 0 ? (profit / costValue) * 100 : 0
        const signal = generateSignal(currentPrice, pos.costPrice, changePercent, stopLoss)
        return {
          ...pos, currentPrice, changePercent, marketValue, profit, profitPercent,
          signal: signal?.signal,
          signalType: signal?.type === 'WARNING' || signal?.type === 'SELL' ? 'sell' : signal?.type === 'BUY' ? 'buy' : 'hold' as const,
          advice: signal?.reason,
        }
      })
    }
  } catch (err) {
    console.error('刷新价格失败:', err)
  }
}

const handleSearch = async (keyword: string) => {
  if (keyword.length < 1) { searchResults.value = []; return }
  searchLoading.value = true
  try {
    const res = await searchStocksApi(keyword)
    searchResults.value = res.code === 200 ? (res.data || []) : []
  } finally {
    searchLoading.value = false
  }
}

const selectStock = (stock: { code: string; name: string }) => {
  form.value = { ...form.value, code: stock.code, name: stock.name }
  isSearchDialogOpen.value = false
  searchResults.value = []
}

const addPosition = () => {
  const quantity = parseInt(form.value.quantity)
  const costPrice = parseFloat(form.value.costPrice)
  if (!form.value.code || Number.isNaN(quantity) || quantity <= 0 || Number.isNaN(costPrice) || costPrice <= 0) return
  const newPosition: Position = {
    id: `pos_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
    code: form.value.code, name: form.value.name, quantity, costPrice, addTime: Date.now(),
  }
  savePositions([...loadPositions(), newPosition])
  isAddDialogOpen.value = false
  form.value = { code: '', name: '', quantity: '', costPrice: '' }
  loadPositionData()
}

const openEditDialog = (pos: PositionDisplay) => {
  selectedPosition.value = pos
  editForm.value = { code: pos.code, name: pos.name, quantity: pos.quantity.toString(), costPrice: pos.costPrice.toString() }
  isEditDialogOpen.value = true
}

const saveEdit = () => {
  if (!selectedPosition.value) return
  const quantity = parseInt(editForm.value.quantity)
  const costPrice = parseFloat(editForm.value.costPrice)
  if (Number.isNaN(quantity) || quantity <= 0 || Number.isNaN(costPrice) || costPrice <= 0) return
  const updated: Position = { ...selectedPosition.value, quantity, costPrice }
  const stored = loadPositions()
  const idx = stored.findIndex(p => p.id === selectedPosition.value!.id)
  if (idx >= 0) { stored[idx] = updated; savePositions(stored) }
  positions.value = positions.value.map(p => {
    if (p.id === selectedPosition.value!.id) {
      const marketValue = p.currentPrice * quantity
      const costValue = costPrice * quantity
      const profit = marketValue - costValue
      const profitPercent = costValue > 0 ? (profit / costValue) * 100 : 0
      return { ...p, quantity, costPrice, marketValue, profit, profitPercent }
    }
    return p
  })
  isEditDialogOpen.value = false
}

const handleDeletePosition = (pos: PositionDisplay) => {
  if (!confirm(`确定要删除 ${pos.name} 的持仓记录吗？`)) return
  const stored = loadPositions()
  savePositions(stored.filter(p => p.id !== pos.id))
  positions.value = positions.value.filter(p => p.id !== pos.id)
}

const openDetailSheet = (pos: PositionDisplay) => {
  detailCode.value = pos.code
  detailCostPrice.value = pos.costPrice
  isDetailSheetOpen.value = true
}

const totalProfit = computed(() => positions.value.reduce((sum, p) => sum + p.profit, 0))
const totalValue = computed(() => positions.value.reduce((sum, p) => sum + p.marketValue, 0))
const totalCost = computed(() => positions.value.reduce((sum, p) => sum + p.costPrice * p.quantity, 0))
const totalProfitPercent = computed(() => totalCost.value > 0 ? (totalProfit.value / totalCost.value) * 100 : 0)

onMounted(() => {
  loadPositionData()
  isTradingSession.value = isTradingTime()
  tradingInterval = setInterval(() => { isTradingSession.value = isTradingTime() }, 60000)
  refreshInterval = setInterval(() => { if (isTradingSession.value && positions.value.length > 0) refreshPositionPrices() }, 5000)
})

onUnmounted(() => {
  if (tradingInterval) clearInterval(tradingInterval)
  if (refreshInterval) clearInterval(refreshInterval)
})
</script>

<template>
  <div>
    <!-- 顶部统计 -->
    <div class="bg-gradient-to-r from-blue-500 to-blue-600 px-4 py-5 -mx-4 sm:-mx-6 lg:-mx-8 -mt-6 mb-4">
      <div class="max-w-7xl mx-auto">
        <div class="flex justify-between items-center mb-4">
          <span class="text-white text-lg font-semibold">我的持仓</span>
          <span :class="['text-xs px-2 py-0.5 rounded-full', isTradingSession ? 'bg-green-500 text-white' : 'bg-slate-400 text-white']">
            {{ isTradingSession ? '交易中' : '已收盘' }}
          </span>
        </div>
        <div class="flex justify-between">
          <div>
            <div class="text-blue-200 text-xs">总市值</div>
            <div class="text-white text-2xl font-bold mt-1">¥{{ totalValue.toFixed(2) }}</div>
          </div>
          <div class="text-right">
            <div class="text-blue-200 text-xs">总盈亏</div>
            <div class="flex items-center justify-end mt-1">
              <svg v-if="totalProfit >= 0" class="w-5 h-5 text-green-300" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" /></svg>
              <svg v-else class="w-5 h-5 text-red-300" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6" /></svg>
              <span :class="['text-xl font-bold ml-1', totalProfit >= 0 ? 'text-green-300' : 'text-red-300']">
                {{ totalProfit >= 0 ? '+' : '' }}{{ totalProfit.toFixed(2) }}
              </span>
            </div>
            <div :class="['text-sm mt-1', totalProfitPercent >= 0 ? 'text-green-200' : 'text-red-200']">
              {{ totalProfitPercent >= 0 ? '+' : '' }}{{ totalProfitPercent.toFixed(2) }}%
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加按钮 -->
    <div class="mb-4">
      <button
        class="w-full py-3 rounded-xl border-2 border-dashed border-blue-300 text-blue-500 font-medium hover:bg-blue-50 transition-colors"
        @click="isAddDialogOpen = true"
      >
        + 手动添加持仓
      </button>
    </div>

    <!-- 评分说明 -->
    <div v-if="positions.length > 0" class="bg-blue-50 rounded-xl p-3 mb-4 flex items-start gap-2">
      <svg class="w-4 h-4 text-blue-500 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
      <div class="text-xs text-blue-700 leading-relaxed">
        <span class="font-semibold">综合评分</span>：纯技术面打分，基于趋势、动量、量能、技术指标计算，不包含基本面。<br />
        <span class="font-semibold">持仓健康度</span>：综合评分结合您的持仓盈亏计算（±15分调整），反映当前持仓风险。
      </div>
    </div>

    <!-- 持仓列表 -->
    <div v-if="positions.length === 0" class="bg-white rounded-xl py-12 text-center shadow-sm border border-slate-100">
      <svg class="w-12 h-12 text-slate-300 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" /></svg>
      <div class="text-slate-500 mb-2">暂无持仓记录</div>
      <div class="text-slate-400 text-sm">点击上方按钮添加您的第一笔持仓</div>
    </div>

    <div
      v-for="pos in positions"
      :key="pos.id"
      class="bg-white rounded-xl p-4 mb-3 shadow-sm border border-slate-100"
    >
      <div class="flex justify-between items-start mb-3 cursor-pointer" @click="openDetailSheet(pos)">
        <div>
          <div class="flex items-center">
            <span class="text-slate-800 font-semibold">{{ pos.name }}</span>
            <StockTag
              v-if="pos.signal && pos.signalType"
              type="signalType"
              :text="pos.signalType"
              class="ml-2"
            >
              {{ pos.signal }}
            </StockTag>
          </div>
          <div class="text-slate-400 text-sm mt-1">{{ pos.code }}</div>
          <div v-if="pos.totalScore !== undefined" class="flex items-center gap-1.5 mt-1.5 flex-wrap">
            <!-- 综合评分 -->
            <StockTag type="score" :score="pos.totalScore" :text="`综合评分 ${pos.totalScore.toFixed(0)}`" />
            <!-- 持仓健康度（与市场评分明确区分） -->
            <StockTag
              v-if="pos.positionHealthScore !== undefined"
              type="score"
              :score="pos.positionHealthScore"
              :text="`持仓健康度 ${pos.positionHealthScore.toFixed(0)}`"
              size="md"
            />
            <!-- MACD信号标签 -->
            <StockTag v-if="pos.macdSignal" type="macdSignal" :text="pos.macdSignal" />
            <!-- 推荐等级 -->
            <StockTag v-if="pos.recommendation" type="recommendation" :text="pos.recommendation" />
            <!-- 亮点标签 -->
            <StockTag
              v-for="h in (pos.highlights || []).slice(0, 3)"
              :key="h"
              type="highlight"
              :text="h"
            />
          </div>
        </div>
        <div class="text-right">
          <div class="text-slate-800 font-semibold text-lg">¥{{ pos.currentPrice.toFixed(2) }}</div>
          <div :class="['flex items-center justify-end mt-1 text-sm', pos.changePercent >= 0 ? 'text-red-500' : 'text-green-500']">
            {{ pos.changePercent >= 0 ? '+' : '' }}{{ pos.changePercent.toFixed(2) }}%
          </div>
        </div>
      </div>

      <div class="bg-slate-50 rounded-lg p-3 mb-3">
        <div class="flex justify-between items-center mb-2">
          <span class="text-slate-500 text-sm">持仓</span>
          <span class="text-slate-700">{{ pos.quantity }}股 @ ¥{{ pos.costPrice.toFixed(2) }}</span>
        </div>
        <div class="flex justify-between items-center">
          <span class="text-slate-500 text-sm">盈亏</span>
          <div class="flex items-center">
            <span :class="['font-semibold', pos.profit >= 0 ? 'text-red-500' : 'text-green-500']">
              {{ pos.profit >= 0 ? '+' : '' }}¥{{ pos.profit.toFixed(2) }}
            </span>
            <span :class="['text-sm ml-2', pos.profitPercent >= 0 ? 'text-red-500' : 'text-green-500']">
              ({{ pos.profitPercent >= 0 ? '+' : '' }}{{ pos.profitPercent.toFixed(2) }}%)
            </span>
          </div>
        </div>
      </div>

      <div v-if="pos.advice" class="bg-blue-50 rounded-lg p-3 mb-3">
        <div class="flex items-start">
          <svg class="w-3.5 h-3.5 text-blue-500 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
          <span class="text-blue-700 text-sm ml-2">{{ pos.advice }}</span>
        </div>
      </div>

      <div class="flex gap-2">
        <button class="flex-1 py-2 rounded-lg border border-red-200 text-red-500 text-sm font-medium hover:bg-red-50 transition-colors" @click="openEditDialog(pos)">
          调整
        </button>
        <button class="flex-1 py-2 rounded-lg border border-orange-200 text-orange-500 text-sm font-medium hover:bg-orange-50 transition-colors flex items-center justify-center gap-1" @click="openDetailSheet(pos)">
          🎯 止盈分析
        </button>
        <button class="py-2 px-3 rounded-lg border border-slate-200 text-slate-500 text-sm font-medium hover:bg-slate-50 transition-colors flex items-center justify-center gap-1" @click="openDetailSheet(pos)">
          详情
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" /></svg>
        </button>
      </div>
    </div>

    <!-- 添加持仓弹窗 -->
    <Teleport to="body">
      <div v-if="isAddDialogOpen" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/50" @click.self="isAddDialogOpen = false">
        <div class="bg-white rounded-t-2xl sm:rounded-2xl w-full sm:max-w-md p-6 max-h-[85vh] overflow-y-auto">
          <h3 class="text-lg font-semibold text-slate-900 mb-4">添加持仓</h3>
          <div class="space-y-4">
            <div>
              <label class="text-sm text-slate-600 mb-2 block">股票代码/名称</label>
              <div class="bg-slate-50 rounded-xl px-4 py-3 flex items-center cursor-pointer" @click="isSearchDialogOpen = true">
                <svg class="w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
                <span class="ml-2 text-sm" :class="form.name ? 'text-slate-700' : 'text-slate-400'">
                  {{ form.name ? `${form.name} (${form.code})` : '点击搜索并选择股票' }}
                </span>
              </div>
            </div>
            <div>
              <label class="text-sm text-slate-600 mb-2 block">持仓数量（股）</label>
              <input v-model="form.quantity" type="number" placeholder="请输入持仓数量" class="w-full px-4 py-3 bg-slate-50 rounded-xl outline-none text-sm text-slate-700" />
            </div>
            <div>
              <label class="text-sm text-slate-600 mb-2 block">持仓成本价（元/股）</label>
              <input v-model="form.costPrice" type="number" step="0.01" placeholder="请输入成本价" class="w-full px-4 py-3 bg-slate-50 rounded-xl outline-none text-sm text-slate-700" />
            </div>
          </div>
          <div class="flex gap-3 mt-6">
            <button class="flex-1 py-2.5 border border-slate-200 rounded-xl text-slate-600 font-medium hover:bg-slate-50" @click="isAddDialogOpen = false">取消</button>
            <button class="flex-1 py-2.5 bg-blue-600 text-white rounded-xl font-medium hover:bg-blue-700" @click="addPosition">保存</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 搜索弹窗 -->
    <Teleport to="body">
      <div v-if="isSearchDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" @click.self="isSearchDialogOpen = false">
        <div class="bg-white rounded-2xl w-[90%] max-w-md p-6">
          <h3 class="text-lg font-semibold text-slate-900 mb-4">搜索股票</h3>
          <input
            :value="form.code"
            type="text"
            placeholder="输入股票代码或名称"
            class="w-full px-4 py-3 bg-slate-50 rounded-xl outline-none text-sm text-slate-700 mb-4"
            @input="(e) => { const val = (e.target as HTMLInputElement).value; form.code = val; form.name = ''; handleSearch(val) }"
          />
          <div v-if="searchLoading" class="text-center py-4 text-slate-400 text-sm">搜索中...</div>
          <div v-else-if="searchResults.length > 0" class="max-h-64 overflow-y-auto">
            <div
              v-for="stock in searchResults"
              :key="stock.code"
              class="p-3 border-b border-slate-100 cursor-pointer hover:bg-slate-50"
              @click="selectStock(stock)"
            >
              <div class="text-slate-800 font-medium">{{ stock.name }}</div>
              <div class="text-slate-400 text-sm">{{ stock.code }}</div>
            </div>
          </div>
          <div v-else-if="form.code.length > 0" class="text-center py-4 text-slate-400 text-sm">未找到相关股票</div>
        </div>
      </div>
    </Teleport>

    <!-- 编辑弹窗 -->
    <Teleport to="body">
      <div v-if="isEditDialogOpen" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/50" @click.self="isEditDialogOpen = false">
        <div class="bg-white rounded-t-2xl sm:rounded-2xl w-full sm:max-w-md p-6">
          <h3 class="text-lg font-semibold text-slate-900 mb-4">调整持仓 - {{ selectedPosition?.name }}</h3>
          <div class="space-y-4">
            <div>
              <label class="text-sm text-slate-600 mb-2 block">持仓数量（股）</label>
              <input v-model="editForm.quantity" type="number" placeholder="请输入持仓数量" class="w-full px-4 py-3 bg-slate-50 rounded-xl outline-none text-sm text-slate-700" />
            </div>
            <div>
              <label class="text-sm text-slate-600 mb-2 block">持仓成本价（元/股）</label>
              <input v-model="editForm.costPrice" type="number" step="0.01" placeholder="请输入成本价" class="w-full px-4 py-3 bg-slate-50 rounded-xl outline-none text-sm text-slate-700" />
            </div>
          </div>
          <div class="flex gap-3 mt-6">
            <button class="flex-1 py-2.5 border border-red-200 text-red-500 rounded-xl font-medium hover:bg-red-50" @click="handleDeletePosition(selectedPosition!); isEditDialogOpen = false">删除</button>
            <button class="flex-1 py-2.5 border border-slate-200 text-slate-600 rounded-xl font-medium hover:bg-slate-50" @click="isEditDialogOpen = false">取消</button>
            <button class="flex-1 py-2.5 bg-blue-600 text-white rounded-xl font-medium hover:bg-blue-700" @click="saveEdit">保存</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 股票详情 -->
    <StockDetailSheet v-model:open="isDetailSheetOpen" :code="detailCode" :cost-price="detailCostPrice" />
  </div>
</template>
