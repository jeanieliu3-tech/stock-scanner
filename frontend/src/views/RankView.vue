<template>
  <div class="rank-view space-y-4">
    <!-- 顶部统计 -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
      <div class="bg-gradient-to-br from-blue-500 to-blue-700 rounded-xl p-4 text-white">
        <div class="text-blue-100 text-xs mb-1">扫描总数</div>
        <div class="text-2xl font-bold">{{ scanResult?.totalStocks?.toLocaleString() ?? '-' }}</div>
      </div>
      <div class="bg-gradient-to-br from-emerald-500 to-emerald-700 rounded-xl p-4 text-white">
        <div class="text-emerald-100 text-xs mb-1">有效标的</div>
        <div class="text-2xl font-bold">{{ scanResult?.validStocks?.toLocaleString() ?? '-' }}</div>
      </div>
      <div class="bg-gradient-to-br from-amber-500 to-amber-700 rounded-xl p-4 text-white">
        <div class="text-amber-100 text-xs mb-1">深度分析</div>
        <div class="text-2xl font-bold">{{ scanResult?.analyzedStocks?.toLocaleString() ?? '-' }}</div>
      </div>
      <div class="bg-gradient-to-br from-purple-500 to-purple-700 rounded-xl p-4 text-white">
        <div class="text-purple-100 text-xs mb-1">耗时</div>
        <div class="text-2xl font-bold">{{ scanResult ? (scanResult.costMs / 1000).toFixed(1) + 's' : '-' }}</div>
      </div>
    </div>

    <!-- 操作栏 -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center gap-3 bg-white rounded-xl p-4 shadow-sm">
      <button
        class="px-5 py-2.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium text-sm flex items-center gap-2 disabled:opacity-50"
        :disabled="scanning"
        @click="startScan"
      >
        <Loader2 v-if="scanning" class="w-4 h-4 animate-spin" />
        <Zap v-else class="w-4 h-4" />
        {{ scanning ? '扫描中...' : '全A股扫描' }}
      </button>

      <div v-if="rankData" class="text-xs text-slate-400">
        扫描时间: {{ rankData.scanTime }}
      </div>

      <div class="flex-1" />

      <!-- 筛选器组 -->
      <div class="flex items-center gap-2 flex-wrap">
        <!-- 行业筛选 -->
        <select
          v-model="industryFilter"
          class="text-sm border rounded-lg px-3 py-2 bg-white focus:ring-2 focus:ring-blue-500 outline-none"
        >
          <option value="">全部行业</option>
          <option v-for="ind in industries" :key="ind" :value="ind">{{ ind }}</option>
        </select>

        <!-- 信号筛选 -->
        <select
          v-model="filterType"
          class="text-sm border rounded-lg px-3 py-2 bg-white focus:ring-2 focus:ring-blue-500 outline-none"
        >
          <option value="">全部信号</option>
          <option value="golden_cross">MACD金叉</option>
          <option value="above_water">水上金叉</option>
          <option value="strong">强势股(涨幅>3%)</option>
          <option value="volume_break">放量突破</option>
          <option value="surge_limit">大涨(涨幅>5%)</option>
        </select>

        <!-- 涨跌幅筛选 -->
        <select
          v-model="changeFilter"
          class="text-sm border rounded-lg px-3 py-2 bg-white focus:ring-2 focus:ring-blue-500 outline-none"
        >
          <option value="">涨跌幅</option>
          <option value="3">涨幅>3%</option>
          <option value="5">涨幅>5%</option>
          <option value="7">涨幅>7%</option>
          <option value="-3">跌幅>3%</option>
        </select>

        <!-- 评分阈值 -->
        <select
          v-model="scoreFilter"
          class="text-sm border rounded-lg px-3 py-2 bg-white focus:ring-2 focus:ring-blue-500 outline-none"
        >
          <option value="">评分阈值</option>
          <option value="75">≥75 强烈推荐</option>
          <option value="60">≥60 积极关注</option>
          <option value="45">≥45 一般关注</option>
          <option value="30">≥30 观望</option>
        </select>

        <!-- 排序 -->
        <select
          v-model="sortBy"
          class="text-sm border rounded-lg px-3 py-2 bg-white focus:ring-2 focus:ring-blue-500 outline-none"
        >
          <option value="totalScore">综合评分</option>
          <option value="changePercent">涨跌幅</option>
          <option value="turnoverRate">换手率</option>
          <option value="volumeRatio">量比</option>
          <option value="techScore">技术评分</option>
          <option value="momentumScore">动量评分</option>
        </select>

        <!-- 每页条数 -->
        <select
          v-model="pageSize"
          class="text-sm border rounded-lg px-3 py-2 bg-white focus:ring-2 focus:ring-blue-500 outline-none"
        >
          <option :value="20">20条/页</option>
          <option :value="50">50条/页</option>
          <option :value="100">100条/页</option>
        </select>
      </div>
    </div>

    <!-- 排名表格 -->
    <div class="bg-white rounded-xl shadow-sm overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead class=" text-slate-600" style="background:#e8e8e8">
            <tr>
              <th class="px-2 py-3 text-center font-bold" style="width:40px">★</th>
              <th class="px-2 py-3 text-center font-bold" style="width:40px">排名</th>
              <th class="px-2 py-3 text-left font-bold" style="min-width:100px">股票</th>
              <th class="px-2 py-3 text-right font-bold">现价</th>
              <th class="px-2 py-3 text-right font-bold">涨跌幅</th>
              <th class="px-2 py-3 text-right font-bold hidden md:table-cell">量比</th>
              <th class="px-2 py-3 text-right font-bold hidden lg:table-cell">换手率</th>
              <th class="px-2 py-3 text-right font-bold" title="纯技术面打分：趋势+动量+量能+技术指标，不包含基本面">综合评分</th>
              <th class="px-2 py-3 text-center font-bold">共振强度</th>
              <th class="px-2 py-3 text-center font-bold">趋势</th>
              <th class="px-2 py-3 text-center font-bold">动量</th>
              <th class="px-2 py-3 text-center font-bold">量能</th>
              <th class="px-2 py-3 text-center font-bold">技术</th>
              <th class="px-2 py-3 text-center font-bold hidden md:table-cell">MACD</th>
              <th class="px-2 py-3 text-center font-bold hidden lg:table-cell">BOLL</th>
              <th class="px-2 py-3 text-center font-bold hidden lg:table-cell">亮点</th>
              <th class="px-2 py-3 text-center font-bold">建议</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr
              v-for="(item, idx) in rankData?.items ?? []"
              :key="item.code"
              class="transition-colors cursor-pointer"
              :class="[
                idx % 2 === 1 ? 'bg-slate-50/70' : '',
                isFavorite(item.code) ? 'bg-amber-50!' : '',
                selectedCode === item.code ? 'ring-2 ring-blue-300' : '',
                'hover:bg-blue-50/50'
              ]"
              @click="selectRow(item.code)"
              @dblclick="openDetail(item.code)"
            >
              <!-- 自选标记 -->
              <td class="px-2 py-3 text-center">
                <button
                  class="text-base leading-none"
                  :class="isFavorite(item.code) ? 'text-amber-400' : 'text-slate-300 hover:text-amber-400'"
                  @click.stop="toggleFavorite(item.code)"
                  :title="isFavorite(item.code) ? '取消自选' : '加入自选'"
                >⚡</button>
              </td>

              <!-- 排名 -->
              <td class="px-2 py-3 text-center">
                <span
                  v-if="item.rank <= 3"
                  class="inline-flex items-center justify-center w-7 h-7 rounded-full text-xs font-bold text-white"
                  :class="{
                    'bg-amber-500': item.rank === 1,
                    'bg-slate-400': item.rank === 2,
                    'bg-amber-700': item.rank === 3,
                  }"
                >{{ item.rank }}</span>
                <span v-else class="text-slate-400 font-medium">{{ item.rank }}</span>
              </td>

              <!-- 股票 -->
              <td class="px-2 py-3">
                <div class="font-medium text-slate-800">{{ item.name }}</div>
                <div class="text-xs text-slate-400">{{ item.code }}</div>
                <div v-if="item.industry" class="text-xs text-blue-400 mt-0.5">{{ item.industry }}</div>
              </td>

              <!-- 现价 -->
              <td class="px-2 py-3 text-right font-mono">{{ item.price.toFixed(2) }}</td>

              <!-- 涨跌幅 -->
              <td class="px-2 py-3 text-right font-mono font-bold" :class="item.changePercent > 0 ? 'text-red-500' : item.changePercent < 0 ? 'text-green-600' : 'text-slate-500'">
                {{ item.changePercent > 0 ? '+' : '' }}{{ item.changePercent.toFixed(2) }}%
              </td>

              <!-- 量比 -->
              <td class="px-2 py-3 text-right font-mono hidden md:table-cell">
                <span v-if="item.volumeRatio > 0" :class="item.volumeRatio >= 3 ? 'text-red-500 font-medium' : item.volumeRatio >= 2 ? 'text-amber-600' : 'text-slate-600'">
                  {{ item.volumeRatio.toFixed(2) }}
                </span>
                <span v-else class="text-slate-300">-</span>
              </td>

              <!-- 换手率 -->
              <td class="px-2 py-3 text-right font-mono hidden lg:table-cell">
                <span v-if="item.turnoverRate > 0" class="text-slate-600">{{ item.turnoverRate.toFixed(2) }}%</span>
                <span v-else class="text-slate-300">-</span>
              </td>

              <!-- 综合评分 -->
              <td class="px-2 py-3 text-right">
                <span
                  class="inline-block px-2 py-0.5 rounded-full text-xs font-bold"
                  :class="{
                    'bg-red-100 text-red-700': item.totalScore >= 70,
                    'bg-amber-100 text-amber-700': item.totalScore >= 55,
                    'bg-blue-100 text-blue-700': item.totalScore >= 40,
                    'bg-slate-100 text-slate-600': item.totalScore < 40,
                  }"
                >{{ item.totalScore.toFixed(1) }}</span>
              </td>

              <!-- 共振强度 -->
              <td class="px-2 py-3 text-center">
                <div class="flex items-center justify-center gap-0.5">
                  <template v-for="n in 5" :key="n">
                    <svg class="w-3.5 h-3.5" viewBox="0 0 20 20" :class="n <= starCount(item.totalScore) ? 'text-amber-400' : n - 0.5 <= starCount(item.totalScore) ? 'text-amber-300' : 'text-slate-200'">
                      <path fill="currentColor" d="M10 1l2.39 6.34H19l-5.3 3.69 2.03 6.47L10 13.77 4.27 17.5l2.03-6.47L1 7.34h6.61z"/>
                    </svg>
                  </template>
                </div>
              </td>

              <!-- 趋势 -->
              <td class="px-2 py-3 text-center"><ScoreBar :score="item.trendScore" :max="25" /></td>
              <!-- 动量 -->
              <td class="px-2 py-3 text-center"><ScoreBar :score="item.momentumScore" :max="35" /></td>
              <!-- 量能 -->
              <td class="px-2 py-3 text-center"><ScoreBar :score="item.volumeScore" :max="15" /></td>
              <!-- 技术 -->
              <td class="px-2 py-3 text-center"><ScoreBar :score="item.techScore" :max="40" /></td>

              <!-- MACD -->
              <td class="px-2 py-3 text-center hidden md:table-cell">
                <StockTag v-if="item.macdSignal" type="macdSignal" :text="item.macdSignal" />
              </td>

              <!-- BOLL -->
              <td class="px-2 py-3 text-center hidden lg:table-cell">
                <StockTag v-if="item.bollPosition" type="bollPosition" :text="item.bollPosition" />
              </td>

              <!-- 亮点 -->
              <td class="px-2 py-3 text-center hidden lg:table-cell">
                <div class="flex flex-wrap gap-0.5 justify-center">
                  <StockTag
                    v-for="h in item.highlights?.slice(0, 3)"
                    :key="h"
                    type="highlight"
                    :text="h"
                  />
                </div>
              </td>

              <!-- 建议 -->
              <td class="px-2 py-3 text-center">
                <StockTag
                  v-if="item.recommendation"
                  type="recommendation"
                  :text="item.recommendation"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 空状态 -->
      <div v-if="!rankData?.items?.length && !scanning" class="py-16 text-center text-slate-400">
        <Trophy class="w-12 h-12 mx-auto mb-3 opacity-30" />
        <p class="text-sm">点击「全A股扫描」开始分析</p>
        <p class="text-xs mt-1">扫描约5000支A股，评分排名约需30-60秒</p>
      </div>

      <!-- 加载状态 -->
      <div v-if="scanning" class="py-16 text-center">
        <Loader2 class="w-10 h-10 mx-auto mb-3 text-blue-500 animate-spin" />
        <p class="text-sm text-slate-600 font-medium">正在扫描全A股...</p>
        <p class="text-xs text-slate-400 mt-1">获取行情数据并分析中，请稍候</p>
        <div class="mt-3 max-w-xs mx-auto bg-slate-100 rounded-full h-2 overflow-hidden">
          <div class="bg-blue-500 h-full rounded-full animate-pulse" style="width: 60%"></div>
        </div>
      </div>
    </div>

    <!-- 简评面板（单击行后显示） -->
    <div v-if="selectedStock" class="bg-white rounded-xl shadow-sm p-4 border border-blue-100">
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-3">
          <span class="text-lg font-bold text-slate-800">{{ selectedStock.name }}</span>
          <span class="text-sm text-slate-400">{{ selectedStock.code }}</span>
          <span class="font-mono font-bold" :class="selectedStock.changePercent > 0 ? 'text-red-500' : 'text-green-600'">
            {{ selectedStock.price.toFixed(2) }} {{ selectedStock.changePercent > 0 ? '+' : '' }}{{ selectedStock.changePercent.toFixed(2) }}%
          </span>
        </div>
        <button class="text-slate-400 hover:text-slate-600" @click="selectedCode = ''">✕</button>
      </div>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
        <div class="bg-slate-50 rounded-lg p-3">
          <div class="text-slate-400 text-xs mb-1">综合评分</div>
          <div class="text-lg font-bold" :class="selectedStock.totalScore >= 60 ? 'text-red-500' : selectedStock.totalScore >= 40 ? 'text-amber-600' : 'text-slate-600'">
            {{ selectedStock.totalScore.toFixed(1) }} <span class="text-xs font-normal">/ 100</span>
          </div>
        </div>
        <div class="bg-slate-50 rounded-lg p-3">
          <div class="text-slate-400 text-xs mb-1">MACD</div>
          <div class="font-medium" :class="selectedStock.macdSignal === '水上金叉' ? 'text-red-500' : selectedStock.isGoldenCross ? 'text-amber-600' : 'text-slate-600'">{{ selectedStock.macdSignal }}</div>
          <div class="text-xs text-slate-400 mt-0.5">
            {{ selectedStock.isAboveWater ? '零轴上方' : '零轴下方' }}
            <span v-if="selectedStock.macdDif !== 0"> | DIF:{{ selectedStock.macdDif.toFixed(2) }} DEA:{{ selectedStock.macdDea.toFixed(2) }}</span>
          </div>
        </div>
        <div class="bg-slate-50 rounded-lg p-3">
          <div class="text-slate-400 text-xs mb-1">BOLL</div>
          <div class="font-medium text-slate-600">{{ selectedStock.bollPosition }}</div>
          <div v-if="selectedStock.bollUpper !== 0" class="text-xs text-slate-400 mt-0.5">
            上:{{ selectedStock.bollUpper.toFixed(2) }} 中:{{ selectedStock.bollMiddle.toFixed(2) }} 下:{{ selectedStock.bollLower.toFixed(2) }}
          </div>
        </div>
        <div class="bg-slate-50 rounded-lg p-3">
          <div class="text-slate-400 text-xs mb-1">量比 / 换手率</div>
          <div class="font-medium text-slate-600">
            {{ selectedStock.volumeRatio > 0 ? selectedStock.volumeRatio.toFixed(2) : '-' }} / {{ selectedStock.turnoverRate > 0 ? selectedStock.turnoverRate.toFixed(2) + '%' : '-' }}
          </div>
        </div>
      </div>
      <div v-if="selectedStock.highlights?.length" class="mt-3 flex flex-wrap gap-1.5">
        <StockTag v-for="h in selectedStock.highlights" :key="h" type="highlight" :text="h" />
      </div>
      <div class="mt-3 text-right">
        <button class="text-sm text-blue-500 hover:text-blue-700 font-medium" @click="openDetail(selectedStock.code)">
          深度分析 →
        </button>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="rankData && rankData.totalPages > 1" class="flex items-center justify-between bg-white rounded-xl p-4 shadow-sm">
      <div class="text-sm text-slate-500">
        共 <span class="font-medium text-slate-700">{{ rankData.total }}</span> 支，第 {{ rankData.page }}/{{ rankData.totalPages }} 页
      </div>
      <div class="flex items-center gap-2">
        <button class="px-3 py-1.5 text-sm border rounded-lg hover:bg-slate-50 disabled:opacity-40" :disabled="currentPage <= 1" @click="currentPage = 1">首页</button>
        <button class="px-3 py-1.5 text-sm border rounded-lg hover:bg-slate-50 disabled:opacity-40" :disabled="currentPage <= 1" @click="currentPage--">上一页</button>
        <button class="px-3 py-1.5 text-sm border rounded-lg hover:bg-slate-50 disabled:opacity-40" :disabled="currentPage >= rankData.totalPages" @click="currentPage++">下一页</button>
        <button class="px-3 py-1.5 text-sm border rounded-lg hover:bg-slate-50 disabled:opacity-40" :disabled="currentPage >= rankData.totalPages" @click="currentPage = rankData.totalPages">末页</button>
      </div>
    </div>

    <!-- 股票详情弹窗 -->
    <StockDetailSheet
      :open="!!detailCode"
      :code="detailCode"
      @update:open="detailCode = $event ? detailCode : ''"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, defineComponent, h, onMounted } from 'vue'
import { Zap, Loader2, Trophy } from 'lucide-vue-next'
import { scanAllAShares, getAllStockRank } from '@/api'
import type { AllStockScanResult, AllStockRankResponse, RankStockItem } from '@/types/stock'
import StockDetailSheet from '@/components/StockDetailSheet.vue'
import StockTag from '@/components/StockTag.vue'

// ScoreBar 组件（使用渲染函数避免 runtime compilation 问题）
const ScoreBar = defineComponent({
  props: {
    score: { type: Number, default: 0 },
    max: { type: Number, default: 25 },
  },
  setup(props) {
    const percent = computed(() => props.max > 0 ? Math.min(100, (props.score / props.max) * 100) : 0)
    const barColor = computed(() => {
      if (percent.value >= 70) return 'bg-red-500'
      if (percent.value >= 40) return 'bg-amber-500'
      if (percent.value >= 20) return 'bg-blue-400'
      return 'bg-slate-300'
    })
    return () => h('div', { class: 'flex items-center gap-1' }, [
      h('div', { class: 'w-12 h-1.5 bg-slate-100 rounded-full overflow-hidden' }, [
        h('div', {
          class: ['h-full rounded-full transition-all', barColor.value],
          style: { width: percent.value + '%' }
        })
      ]),
      h('span', { class: 'text-xs text-slate-500 w-6 text-right' }, props.score.toFixed(1))
    ])
  }
})

const scanning = ref(false)
const scanResult = ref<AllStockScanResult | null>(null)
const rankData = ref<AllStockRankResponse | null>(null)
const currentPage = ref(1)
const pageSize = ref(20)
const sortBy = ref('totalScore')
const filterType = ref('')
const industryFilter = ref('')
const changeFilter = ref('')
const scoreFilter = ref('')
const detailCode = ref('')
const selectedCode = ref('')

// 自选股（本地存储）
const favorites = ref<string[]>(JSON.parse(localStorage.getItem('stock_favorites') || '[]'))

function isFavorite(code: string) {
  return favorites.value.includes(code)
}

function toggleFavorite(code: string) {
  if (isFavorite(code)) {
    favorites.value = favorites.value.filter(c => c !== code)
  } else {
    favorites.value.push(code)
  }
  localStorage.setItem('stock_favorites', JSON.stringify(favorites.value))
}

// 行业列表
const industries = computed(() => {
  const set = new Set<string>()
  rankData.value?.items?.forEach(item => {
    if (item.industry) set.add(item.industry)
  })
  return Array.from(set).sort()
})

// 星级计算（每10分半星，满分5星）
function starCount(score: number): number {
  return Math.min(5, Math.round(score / 10) / 2)
}

// 选中的股票
const selectedStock = computed(() => {
  if (!selectedCode.value || !rankData.value?.items) return null
  return rankData.value.items.find(item => item.code === selectedCode.value) || null
})

function selectRow(code: string) {
  selectedCode.value = selectedCode.value === code ? '' : code
}

async function startScan() {
  scanning.value = true
  try {
    const res = await scanAllAShares()
    if (res.code === 200 && res.data) {
      scanResult.value = res.data
      currentPage.value = 1
      await loadRank()
    }
  } catch (e) {
    console.error('全A扫描失败', e)
  } finally {
    scanning.value = false
  }
}

async function loadRank() {
  try {
    const params: Record<string, string | number> = {
      page: currentPage.value,
      pageSize: pageSize.value,
      sortBy: sortBy.value,
      order: 'desc',
    }
    // 合并筛选条件
    let filter = filterType.value
    if (changeFilter.value) {
      const val = parseFloat(changeFilter.value)
      if (val > 0) filter += (filter ? ',' : '') + `change_gt_${val}`
      else filter += (filter ? ',' : '') + `change_lt_${Math.abs(val)}`
    }
    if (scoreFilter.value) {
      filter += (filter ? ',' : '') + `score_gte_${scoreFilter.value}`
    }
    if (industryFilter.value) {
      filter += (filter ? ',' : '') + `industry_${industryFilter.value}`
    }
    if (filter) params.filter = filter

    const res = await getAllStockRank(params)
    if (res.code === 200 && res.data) {
      rankData.value = res.data
    }
  } catch (e) {
    console.error('加载排名失败', e)
  }
}

function openDetail(code: string) {
  detailCode.value = code
}

// 监听分页和排序变化
watch([currentPage, pageSize, sortBy, filterType, changeFilter, scoreFilter, industryFilter], () => {
  loadRank()
})

// 初始加载
onMounted(() => {
  loadRank()
})
</script>
