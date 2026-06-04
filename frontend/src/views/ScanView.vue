<script setup lang="ts">
import { ref } from 'vue'
import {
  diagnoseStock,
  scanCoreSatellite,
  getWatchList,
  addWatchList,
  removeWatchList,
  scanWatchList,
} from '@/api'
import type { ScanResult, WatchScanResult, DiagnoseResult, WatchScanStock } from '@/types/stock'
import StockDetailSheet from '@/components/StockDetailSheet.vue'
import StockTag from '@/components/StockTag.vue'

const inputCode = ref('')
const diagnosing = ref(false)
const diagnoseResult = ref<DiagnoseResult | null>(null)
const scanLoading = ref(false)
const watchScanLoading = ref(false)
const scanResult = ref<ScanResult | null>(null)
const watchScanResult = ref<WatchScanResult | null>(null)
// 扫描错误类型：null=未扫描 | 'empty'=无匹配 | 'timeout'=超时 | 'error'=服务器错误
const scanError = ref<'empty' | 'timeout' | 'error' | null>(null)
const watchList = ref<string[]>([])
const detailOpen = ref(false)
const selectedCode = ref('')
const alertVisible = ref(false)
const alertStock = ref<WatchScanStock | null>(null)

const handleDiagnose = async () => {
  if (!inputCode.value.trim()) return
  diagnosing.value = true
  try {
    const res = await diagnoseStock(inputCode.value.trim())
    if (res.code === 200) {
      diagnoseResult.value = res.data as DiagnoseResult
    }
  } catch (error) {
    console.error('诊断失败:', error)
  } finally {
    diagnosing.value = false
  }
}

const handleFullScan = async () => {
  scanLoading.value = true
  scanResult.value = null
  scanError.value = null
  try {
    const res = await scanCoreSatellite()
    if (res.code === 200) {
      const data = res.data as ScanResult
      scanResult.value = data
      // 判断是否为空结果（接口正常但无匹配）
      if (!data || (data.core.length === 0 && data.satellite.length === 0)) {
        scanError.value = 'empty'
        console.info('[Scan] 200 OK 但核心+卫星均为空，属于策略无匹配，非服务错误')
      }
    } else {
      scanError.value = 'error'
      console.warn(`[Scan] 业务错误 code=${res.code} msg=${res.msg}`)
    }
  } catch (error: unknown) {
    // AbortError = 前端超时；TypeError network = 网络断开；其他 = 服务异常
    if (error instanceof Error) {
      if (error.name === 'AbortError') {
        scanError.value = 'timeout'
        console.error('[Scan] 请求超时 (AbortError):', error.message)
      } else {
        scanError.value = 'error'
        console.error('[Scan] 请求异常:', error.name, error.message)
      }
    } else {
      scanError.value = 'error'
      console.error('[Scan] 未知错误:', error)
    }
  } finally {
    scanLoading.value = false
  }
}

const handleAddWatch = async () => {
  if (!inputCode.value.trim()) return
  try {
    const res = await addWatchList(inputCode.value.trim())
    if (res.code === 200) {
      watchList.value.push(inputCode.value.trim())
      inputCode.value = ''
    }
  } catch (error) {
    console.error('添加关注失败:', error)
  }
}

const handleRemoveWatch = async (code: string) => {
  try {
    await removeWatchList(code)
    watchList.value = watchList.value.filter(c => c !== code)
  } catch (error) {
    console.error('移除失败:', error)
  }
}

const handleScanWatchList = async () => {
  watchScanLoading.value = true
  try {
    const res = await scanWatchList()
    if (res.code === 200) {
      const result = res.data as WatchScanResult
      watchScanResult.value = result
      if (result.signals && result.signals.length > 0) {
        alertStock.value = result.signals[0]
        alertVisible.value = true
      }
    }
  } catch (error) {
    console.error('扫描失败:', error)
  } finally {
    watchScanLoading.value = false
  }
}

const loadWatchList = async () => {
  try {
    const res = await getWatchList()
    if (res.code === 200) {
      watchList.value = res.data as string[]
    }
  } catch (error) {
    console.error('获取关注列表失败:', error)
  }
}

const openDetail = (code: string) => {
  selectedCode.value = code
  detailOpen.value = true
}

const formatScore = (score: number) => {
  const intScore = Math.floor(score)
  const decimal = score - intScore
  return decimal > 0.01 ? score.toFixed(1) : intScore.toString()
}

/** 诊断建议 → 统一标签文本映射 */
const getDiagnosisRecText = (rec: string): string => {
  switch (rec) {
    case 'buy': return '建议买入'
    case 'avoid': return '建议规避'
    default: return '建议观望'
  }
}

// Initialize watch list
loadWatchList()
</script>

<template>
  <div class="space-y-4">
    <!-- 个股诊断 -->
    <div class="bg-white rounded-xl p-4 shadow-sm border border-slate-100">
      <div class="text-sm font-semibold text-slate-700 mb-3">个股诊断</div>
      <div class="text-xs text-slate-400 mb-3">此评分使用独立简化算法，仅供参考，与全A扫描排名不通用</div>
      <div class="flex gap-2">
        <div class="flex-1 flex items-center bg-slate-50 rounded-xl px-4 py-2.5">
          <svg class="w-4 h-4 text-slate-400 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
          <input
            v-model="inputCode"
            class="flex-1 ml-2 bg-transparent outline-none text-sm text-slate-700 placeholder:text-slate-400"
            placeholder="输入股票代码"
            @keyup.enter="handleDiagnose"
          />
        </div>
        <button
          class="px-4 py-2.5 rounded-xl text-sm font-medium text-white transition-all"
          :class="diagnosing ? 'bg-slate-400' : 'bg-blue-600 hover:bg-blue-700'"
          :disabled="diagnosing"
          @click="handleDiagnose"
        >
          {{ diagnosing ? '诊断中...' : '诊断' }}
        </button>
      </div>

      <!-- 诊断结果 -->
      <div v-if="diagnoseResult" class="mt-4 p-3 bg-slate-50 rounded-lg">
        <div class="flex justify-between items-center mb-2">
          <span class="font-bold text-slate-900">{{ diagnoseResult.name }}</span>
          <StockTag type="recommendation" :text="getDiagnosisRecText(diagnoseResult.recommendation)" />
        </div>
        <div class="text-sm text-slate-600">{{ diagnoseResult.analysis }}</div>
      </div>
    </div>

    <!-- 关注列表 -->
    <div class="bg-white rounded-xl p-4 shadow-sm border border-slate-100">
      <div class="text-sm font-semibold text-slate-700 mb-3">关注列表</div>
      <div class="flex gap-2 mb-3">
        <div class="flex-1 flex items-center bg-slate-50 rounded-xl px-4 py-2.5">
          <svg class="w-4 h-4 text-slate-400 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
          <input
            v-model="inputCode"
            class="flex-1 ml-2 bg-transparent outline-none text-sm text-slate-700 placeholder:text-slate-400"
            placeholder="输入股票代码"
          />
        </div>
        <button class="px-4 py-2.5 rounded-xl text-sm font-medium bg-blue-600 text-white hover:bg-blue-700 transition-all" @click="handleAddWatch">
          添加
        </button>
      </div>

      <div v-if="watchList.length > 0" class="space-y-2">
        <div
          v-for="code in watchList"
          :key="code"
          class="flex items-center justify-between p-2 bg-slate-50 rounded-lg"
        >
          <span class="text-sm text-slate-700">{{ code }}</span>
          <div class="flex gap-2">
            <button class="text-xs text-blue-600 hover:text-blue-800 px-2 py-1" @click="inputCode = code; handleDiagnose()">
              诊断
            </button>
            <button class="text-xs text-slate-400 hover:text-red-500 px-2 py-1" @click="handleRemoveWatch(code)">
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
            </button>
          </div>
        </div>
        <button
          class="w-full py-2.5 rounded-xl text-sm font-medium transition-all"
          :class="watchScanLoading ? 'bg-slate-400 text-white' : 'bg-blue-50 text-blue-600 hover:bg-blue-100'"
          :disabled="watchScanLoading"
          @click="handleScanWatchList"
        >
          {{ watchScanLoading ? '扫描中...' : '扫描关注列表' }}
        </button>
      </div>
      <div v-else class="text-sm text-slate-400 text-center py-4">暂无关注的股票</div>
    </div>

    <!-- 全池扫描 -->
    <button
      class="w-full flex items-center justify-center gap-2 py-3 rounded-xl font-medium text-white transition-all"
      :class="scanLoading ? 'bg-slate-400 cursor-not-allowed' : 'bg-blue-600 hover:bg-blue-700'"
      :disabled="scanLoading"
      @click="handleFullScan"
    >
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
      {{ scanLoading ? '扫描中...' : '全池扫描（核心-卫星策略）' }}
    </button>

    <!-- 空状态提示：情况B 服务错误/超时 -->
    <div
      v-if="!scanLoading && scanError === 'timeout'"
      class="bg-orange-50 border border-orange-200 rounded-xl p-4 text-center"
    >
      <div class="text-2xl mb-2">⏳</div>
      <div class="text-sm font-medium text-orange-700">扫描服务繁忙，请稍后重试</div>
      <div class="text-xs text-orange-500 mt-1">网络超时，服务器正在处理大量请求</div>
    </div>
    <div
      v-else-if="!scanLoading && scanError === 'error'"
      class="bg-red-50 border border-red-200 rounded-xl p-4 text-center"
    >
      <div class="text-2xl mb-2">⚠️</div>
      <div class="text-sm font-medium text-red-700">扫描服务繁忙，请稍后重试</div>
      <div class="text-xs text-red-500 mt-1">服务器内部错误，请检查控制台日志</div>
    </div>

    <!-- 空状态提示：情况A 计算完成但无匹配 -->
    <div
      v-else-if="!scanLoading && scanError === 'empty'"
      class="bg-slate-50 border border-slate-200 rounded-xl p-4 text-center"
    >
      <div class="text-2xl mb-2">🔍</div>
      <div class="text-sm font-medium text-slate-600">当前策略未匹配到符合条件的股票</div>
      <div class="text-xs text-slate-400 mt-1">建议放宽筛选条件或稍后再试</div>
    </div>

    <!-- 扫描结果 -->
    <template v-if="scanResult && scanError !== 'empty'">
      <div v-if="scanResult.core.length > 0">
        <div class="flex items-center gap-2 mb-3">
          <div class="w-1 h-6 bg-yellow-500 rounded" />
          <div class="text-lg font-bold text-slate-900">核心标的池</div>
          <span class="text-xs bg-yellow-100 text-yellow-800 px-2 py-0.5 rounded-full font-medium">
            建议仓位 {{ (scanResult.coreTotalWeight * 100).toFixed(0) }}%
          </span>
        </div>
        <div
          v-for="(stock, index) in scanResult.core"
          :key="stock.code"
          class="bg-white rounded-xl p-4 mb-3 border-l-4 border-l-yellow-500 shadow-sm border border-slate-100 cursor-pointer hover:shadow-md transition-shadow"
          @click="openDetail(stock.code)"
        >
          <div class="flex justify-between items-start mb-2">
            <div class="flex items-center gap-2">
              <span v-if="index < 3" class="text-lg">{{ ['🥇','🥈','🥉'][index] }}</span>
              <div>
                <span class="font-bold text-slate-900">{{ stock.name }}</span>
                <span class="text-xs text-slate-500 ml-2">{{ stock.code }}</span>
              </div>
            </div>
            <div class="text-right">
              <div class="font-bold text-slate-900">¥{{ stock.price.toFixed(2) }}</div>
              <div :class="['text-sm font-medium', stock.changePercent >= 0 ? 'text-red-500' : 'text-green-500']">
                {{ stock.changePercent >= 0 ? '+' : '' }}{{ stock.changePercent.toFixed(2) }}%
              </div>
            </div>
          </div>
          <div class="flex items-center gap-2 mb-2 flex-wrap">
            <StockTag type="highlight" :text="stock.sector" />
            <StockTag type="score" :score="stock.score.totalScore" :text="`市场评分 ${formatScore(stock.score.totalScore)}`" />
            <StockTag v-if="stock.recommendation" type="recommendation" :text="stock.recommendation" />
          </div>
          <div class="text-xs text-slate-600 mb-2">资金热度：{{ (stock as any).fundHeat || '--' }}</div>
          <div class="bg-gradient-to-r from-blue-50 to-indigo-50 rounded-lg p-3">
            <div class="flex justify-between items-center">
              <span class="text-sm text-slate-700">建议仓位</span>
              <span class="text-xl font-bold text-blue-600">{{ (stock.recommendedPosition * 100).toFixed(1) }}%</span>
            </div>
          </div>
          <div class="mt-2 pt-2 border-t border-slate-100 flex flex-wrap gap-1.5">
            <StockTag v-if="stock.macdSignal" type="macdSignal" :text="stock.macdSignal" />
            <StockTag v-if="stock.bollPosition" type="bollPosition" :text="stock.bollPosition" />
          </div>
        </div>
      </div>

      <div v-if="scanResult.satellite.length > 0">
        <div class="flex items-center gap-2 mb-3">
          <div class="w-1 h-6 bg-slate-400 rounded" />
          <div class="text-lg font-bold text-slate-900">卫星标的池</div>
          <span class="text-xs bg-slate-100 text-slate-800 px-2 py-0.5 rounded-full font-medium">
            建议仓位 {{ (scanResult.satelliteTotalWeight * 100).toFixed(0) }}%
          </span>
        </div>
        <div
          v-for="stock in scanResult.satellite"
          :key="stock.code"
          class="bg-white rounded-xl p-4 mb-3 border-l-4 border-l-slate-400 shadow-sm border border-slate-100 cursor-pointer hover:shadow-md transition-shadow"
          @click="openDetail(stock.code)"
        >
          <div class="flex justify-between items-start mb-2">
            <div class="flex items-center gap-2">
              <div>
                <span class="font-bold text-slate-900">{{ stock.name }}</span>
                <span class="text-xs text-slate-500 ml-2">{{ stock.code }}</span>
              </div>
            </div>
            <div class="text-right">
              <div class="font-bold text-slate-900">¥{{ stock.price.toFixed(2) }}</div>
              <div :class="['text-sm font-medium', stock.changePercent >= 0 ? 'text-red-500' : 'text-green-500']">
                {{ stock.changePercent >= 0 ? '+' : '' }}{{ stock.changePercent.toFixed(2) }}%
              </div>
            </div>
          </div>
          <div class="flex items-center gap-2 mb-2 flex-wrap">
            <StockTag type="highlight" :text="stock.sector" />
            <StockTag type="score" :score="stock.score.totalScore" :text="`市场评分 ${formatScore(stock.score.totalScore)}`" />
            <StockTag v-if="stock.recommendation" type="recommendation" :text="stock.recommendation" />
          </div>
          <div class="bg-gradient-to-r from-blue-50 to-indigo-50 rounded-lg p-3">
            <div class="flex justify-between items-center">
              <span class="text-sm text-slate-700">建议仓位</span>
              <span class="text-xl font-bold text-blue-600">{{ (stock.recommendedPosition * 100).toFixed(1) }}%</span>
            </div>
          </div>
          <div class="mt-2 pt-2 border-t border-slate-100 flex flex-wrap gap-1.5">
            <StockTag v-if="stock.macdSignal" type="macdSignal" :text="stock.macdSignal" />
            <StockTag v-if="stock.bollPosition" type="bollPosition" :text="stock.bollPosition" />
          </div>
        </div>
      </div>
    </template>

    <!-- 关注列表扫描结果 -->
    <div
      v-if="watchScanResult && watchScanResult.signals && watchScanResult.signals.length > 0"
      class="bg-white rounded-xl p-4 shadow-sm border border-slate-100"
    >
      <div class="text-sm font-semibold text-slate-700 mb-3">关注列表扫描结果</div>
      <div
        v-for="stock in watchScanResult.signals"
        :key="stock.code"
        class="p-3 bg-green-50 rounded-lg mb-2 cursor-pointer hover:bg-green-100 transition-colors"
        @click="openDetail(stock.code)"
      >
        <div class="flex justify-between">
          <span class="font-bold text-slate-900">{{ stock.name }}({{ stock.code }})</span>
          <span class="text-green-600 font-medium">+{{ (stock.recommendedPosition * 100).toFixed(1) }}%</span>
        </div>
        <div class="flex items-center gap-1.5 mt-1">
          <StockTag type="highlight" :text="stock.sector" />
          <StockTag type="score" :score="stock.score" :text="`${stock.score}分`" />
        </div>
      </div>
    </div>

    <!-- 买入提醒弹窗 -->
    <Teleport to="body">
      <div v-if="alertVisible" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" @click.self="alertVisible = false">
        <div class="bg-white rounded-2xl w-[90%] max-w-md p-6 shadow-xl">
          <div class="flex items-center gap-2 text-red-600 mb-4">
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" /></svg>
            <span class="font-semibold">买入提醒</span>
          </div>
          <div v-if="alertStock" class="py-2">
            <div class="text-center mb-4">
              <div class="text-2xl font-bold text-slate-900">{{ alertStock.name }}</div>
              <div class="text-slate-500 text-sm">({{ alertStock.code }})</div>
            </div>
            <div class="text-center mb-4">
              <div class="text-4xl font-bold text-red-600">{{ (alertStock.recommendedPosition * 100).toFixed(1) }}%</div>
              <div class="text-sm text-slate-500">预期收益率</div>
            </div>
            <div class="space-y-2 text-sm">
              <div class="flex justify-between p-2 bg-slate-50 rounded">
                <span class="text-slate-600">当前价格</span>
                <span class="font-medium">¥{{ alertStock.price.toFixed(2) }}</span>
              </div>
              <div class="flex justify-between p-2 bg-slate-50 rounded">
                <span class="text-slate-600">综合评分</span>
                <StockTag type="score" :score="alertStock.score" :text="`${alertStock.score}分`" size="md" />
              </div>
              <div class="flex justify-between p-2 bg-slate-50 rounded">
                <span class="text-slate-600">所属板块</span>
                <StockTag type="highlight" :text="alertStock.sector" />
              </div>
              <div class="flex justify-between p-2 bg-slate-50 rounded">
                <span class="text-slate-600">资金热度</span>
                <StockTag type="highlight" :text="alertStock.fundHeat" />
              </div>
            </div>
          </div>
          <button
            class="w-full mt-4 py-2.5 bg-blue-600 text-white rounded-xl font-medium hover:bg-blue-700 transition-all"
            @click="alertVisible = false"
          >
            我知道了
          </button>
        </div>
      </div>
    </Teleport>

    <!-- 股票详情 -->
    <StockDetailSheet v-model:open="detailOpen" :code="selectedCode" />
  </div>
</template>
