package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"stock-app/models"
)

type StockService struct {
	watchList   map[string]bool
	watchListMu sync.RWMutex
	httpClient  *http.Client
	// 专用于全A扫描翻页：pz=500，约12次请求，需要更长超时
	scanHttpClient *http.Client
	// 全A扫描缓存
	allStockCache     []models.RankStockItem
	allStockCacheTime time.Time
	allStockCacheMu   sync.RWMutex
	refreshRunning    bool // 防止并发异步刷新
}

func NewStockService() *StockService {
	sharedTransport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	// 扫描专用Transport：显式使用系统代理
	// EastMoney TLS 需要重协商，Go直连会EOF，必须走代理
	scanTransport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		Proxy:               http.ProxyFromEnvironment, // 显式启用代理链
	}
	return &StockService{
		watchList: make(map[string]bool),
		// 普通接口 30s 足够
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: sharedTransport,
		},
		// 全A扫描：走系统代理 + 120s超时
		scanHttpClient: &http.Client{
			Timeout:   120 * time.Second,
			Transport: scanTransport,
		},
	}
}

// GetMarketIndices 获取大盘指数（批量一次请求）
func (s *StockService) GetMarketIndices() (*models.MarketData, error) {
	codes := []string{"sh000001", "sz399001", "sz399006", "sh000688"}
	keyMap := map[string]string{
		"000001": "shanghai",
		"399001": "shenzhen",
		"399006": "chinext",
		"000688": "star50",
	}
	indices := make(map[string]*models.IndexData)

	// 批量获取，一次网络请求
	quotes, err := s.fetchBatchSinaQuotes(codes)
	if err == nil {
		for _, q := range quotes {
			// 从 sinaCode 提取纯代码匹配 key
			for suffix, key := range keyMap {
				if strings.HasSuffix(q.Code, suffix) {
					indices[key] = &models.IndexData{
						Price:         q.Price,
						Change:        q.Change,
						ChangePercent: q.ChangePercent,
					}
					break
				}
			}
		}
	}

	// 批量失败时回退逐个请求
	if len(indices) == 0 {
		for _, code := range codes {
			quote, err2 := s.fetchSinaQuote(code)
			if err2 != nil {
				continue
			}
			for suffix, key := range keyMap {
				if strings.HasSuffix(code, suffix) {
					indices[key] = &models.IndexData{
						Price:         quote.Price,
						Change:        quote.Change,
						ChangePercent: quote.ChangePercent,
					}
					break
				}
			}
		}
	}

	shChange := 0.0
	if indices["shanghai"] != nil {
		shChange = indices["shanghai"].ChangePercent
	}
	status := "warning"
	if shChange > 0.5 {
		status = "safe"
	} else if shChange < -0.5 {
		status = "danger"
	}

	return &models.MarketData{
		Time:    time.Now().Format(time.RFC3339),
		Status:  status,
		Indices: indices,
	}, nil
}

// GetBatchQuotes 批量获取报价
func (s *StockService) GetBatchQuotes(codes []string) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)
	if len(codes) == 0 {
		return result, nil
	}

	sinaCodes := make([]string, len(codes))
	for i, code := range codes {
		sinaCodes[i] = toSinaCode(code)
	}

	quotes, err := s.fetchBatchSinaQuotes(sinaCodes)
	if err != nil {
		return result, err
	}

	for _, q := range quotes {
		result[q.Code] = map[string]string{
			"price":         fmt.Sprintf("%.2f", q.Price),
			"changePercent": fmt.Sprintf("%.2f", q.ChangePercent),
			"name":          q.Name,
		}
	}
	return result, nil
}

// DiagnoseStock 个股诊断
func (s *StockService) DiagnoseStock(code string) (*models.DiagnoseResult, error) {
	sinaCode := toSinaCode(code)
	quote, err := s.fetchSinaQuote(sinaCode)
	if err != nil {
		return nil, err
	}

	klines, _ := s.fetchKLineData(code, 60)
	macdData := calcMACD(klines)
	bollData := calcBOLL(klines)

	var recommendation, analysis string
	var score float64

	score = 50

	if macdData.Dif > macdData.Dea {
		score += 15
	}
	if macdData.Dif > 0 {
		score += 10
	}
	if len(klines) > 0 {
		lastClose := klines[len(klines)-1].Close
		if bollData != nil && lastClose > bollData.Middle {
			score += 10
		}
	}
	if quote.ChangePercent > 0 {
		score += 5
	}
	if quote.Volume > 0 && len(klines) > 1 {
		avgVol := 0.0
		for _, k := range klines {
			avgVol += k.Volume
		}
		avgVol /= float64(len(klines))
		if avgVol > 0 && quote.Volume > avgVol*1.5 {
			score += 10
		}
	}

	if score >= 70 {
		recommendation = "buy"
		analysis = fmt.Sprintf("%s 综合评分%.0f分，MACD%s，趋势向好，可考虑逢低布局。", quote.Name, score, macdData.Signal)
	} else if score >= 50 {
		recommendation = "hold"
		analysis = fmt.Sprintf("%s 综合评分%.0f分，MACD%s，建议观望等待更明确信号。", quote.Name, score, macdData.Signal)
	} else {
		recommendation = "avoid"
		analysis = fmt.Sprintf("%s 综合评分%.0f分，MACD%s，趋势偏弱，建议规避。", quote.Name, score, macdData.Signal)
	}

	return &models.DiagnoseResult{
		Name:           quote.Name,
		Code:           code,
		Price:          quote.Price,
		ChangePercent:  quote.ChangePercent,
		Recommendation: recommendation,
		Analysis:       analysis,
		Score:          score,
	}, nil
}

// ScanDualEngineFast 快速双引擎扫描
func (s *StockService) ScanDualEngineFast() (*models.ScanResult, error) {
	pool := getDefaultStockPool()
	totalScanned := len(pool)

	quotes := s.fetchBatchQuotesFromPool(pool)
	validQuotes := len(quotes)

	sorted := make([]models.SinaQuote, 0, len(quotes))
	for _, q := range quotes {
		if q.Price > 0 && q.ChangePercent > -9.5 && q.ChangePercent < 9.5 {
			sorted = append(sorted, q)
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ChangePercent > sorted[j].ChangePercent
	})

	deepCount := 30
	if len(sorted) < deepCount {
		deepCount = len(sorted)
	}
	topStocks := sorted[:deepCount]

	// 并发获取K线数据（原串行30次HTTP请求→并发20组，大幅提速）
	type klineResult struct {
		index    int
		quote    models.SinaQuote
		klines   []models.KLineData
		macdData *MacdResult
		bollData *BollResult
	}
	klineResults := make([]klineResult, deepCount)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20)
	for i, q := range topStocks {
		wg.Add(1)
		go func(idx int, quote models.SinaQuote) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			klines, _ := s.fetchKLineData(quote.Code, 60)
			macdData := calcMACD(klines)
			bollData := calcBOLL(klines)
			klineResults[idx] = klineResult{
				index:    idx,
				quote:    quote,
				klines:   klines,
				macdData: macdData,
				bollData: bollData,
			}
		}(i, q)
	}
	wg.Wait()

	coreStocks := make([]models.DualEngineStock, 0)
	satelliteStocks := make([]models.DualEngineStock, 0)
	for i, kr := range klineResults {
		dualStock := s.buildDualEngineStock(kr.quote, kr.macdData, kr.bollData, kr.klines, i+1)

		if i < 3 && dualStock.Score.TotalScore >= 50 {
			coreStocks = append(coreStocks, dualStock)
		} else if dualStock.Score.TotalScore >= 40 {
			satelliteStocks = append(satelliteStocks, dualStock)
		}
	}

	coreWeight := 0.0
	if len(coreStocks) > 0 {
		coreWeight = 0.45
	}
	satWeight := 0.0
	if len(satelliteStocks) > 0 {
		satWeight = 0.30
	}
	cashReserve := 1.0 - coreWeight - satWeight
	if cashReserve < 0.2 {
		cashReserve = 0.2
	}

	// 将扫描评分同步到 allStockCache，确保持仓页/排名页看到完全一致的评分
	allScored := append(coreStocks, satelliteStocks...)
	s.syncDualEngineScoresToCache(allScored)

	return &models.ScanResult{
		TotalScanned:         totalScanned,
		ValidQuotes:          validQuotes,
		DeepAnalyzedCount:    deepCount,
		Core:                 coreStocks,
		Satellite:            satelliteStocks,
		CoreTotalWeight:      coreWeight,
		SatelliteTotalWeight: satWeight,
		CashReserve:          cashReserve,
		ScanTime:             time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// ScanCoreSatellite 核心-卫星扫描
func (s *StockService) ScanCoreSatellite() (*models.ScanResult, error) {
	return s.ScanDualEngineFast()
}

// ScanStocks 扫描股票
func (s *StockService) ScanStocks() (*models.ScanResult, error) {
	return s.ScanDualEngineFast()
}

// GetStockDetail 获取股票详情
func (s *StockService) GetStockDetail(code string) (*models.StockTechnicalDetail, error) {
	sinaCode := toSinaCode(code)
	quote, err := s.fetchSinaQuote(sinaCode)
	if err != nil {
		return nil, err
	}

	klines, _ := s.fetchKLineData(code, 120)
	macdData := calcMACD(klines)
	bollData := calcBOLL(klines)

	klineResult := make([]models.KLineData, 0, len(klines))
	for _, k := range klines {
		klineResult = append(klineResult, models.KLineData{
			Date:   k.Date,
			Open:   k.Open,
			High:   k.High,
			Low:    k.Low,
			Close:  k.Close,
			Volume: k.Volume,
		})
	}

	macd := models.MacdData{Dif: 0, Dea: 0, Macd: 0, Status: "数据不足", Signal: "未知", AxisPosition: "未知"}
	if macdData != nil {
		macd = models.MacdData{
			Dif:          macdData.Dif,
			Dea:          macdData.Dea,
			Macd:         macdData.Macd,
			Status:       macdData.Status,
			Signal:       macdData.Signal,
			AxisPosition: macdData.AxisPosition,
		}
	}

	boll := models.BollData{Upper: 0, Middle: 0, Lower: 0, Position: "数据不足", Bandwidth: 0}
	if bollData != nil {
		position := "中轨附近"
		if len(klines) > 0 {
			lastClose := klines[len(klines)-1].Close
			if lastClose >= bollData.Upper {
				position = "突破上轨"
			} else if lastClose >= bollData.Middle {
				position = "上轨与中轨之间"
			} else if lastClose >= bollData.Lower {
				position = "中轨与下轨之间"
			} else {
				position = "跌破下轨(超卖)"
			}
		}
		boll = models.BollData{
			Upper:     bollData.Upper,
			Middle:    bollData.Middle,
			Lower:     bollData.Lower,
			Position:  position,
			Bandwidth: bollData.Bandwidth,
		}
	}

	return &models.StockTechnicalDetail{
		Code:          code,
		Name:          quote.Name,
		Price:         quote.Price,
		ChangePercent: quote.ChangePercent,
		Macd:          macd,
		Boll:          boll,
		KlineHistory:  klineResult,
	}, nil
}

// WatchList operations
func (s *StockService) GetWatchList() []string {
	s.watchListMu.RLock()
	defer s.watchListMu.RUnlock()
	result := make([]string, 0, len(s.watchList))
	for code := range s.watchList {
		result = append(result, code)
	}
	sort.Strings(result)
	return result
}

func (s *StockService) AddWatchList(code string) (bool, string) {
	s.watchListMu.Lock()
	defer s.watchListMu.Unlock()
	if s.watchList[code] {
		return false, "已在关注列表中"
	}
	s.watchList[code] = true
	return true, "添加成功"
}

func (s *StockService) RemoveWatchList(code string) (bool, string) {
	s.watchListMu.Lock()
	defer s.watchListMu.Unlock()
	if !s.watchList[code] {
		return false, "不在关注列表中"
	}
	delete(s.watchList, code)
	return true, "移除成功"
}

func (s *StockService) ScanWatchList() (*models.WatchScanResult, error) {
	s.watchListMu.RLock()
	codes := make([]string, 0, len(s.watchList))
	for code := range s.watchList {
		codes = append(codes, code)
	}
	s.watchListMu.RUnlock()

	signals := make([]models.WatchScanStock, 0)
	for _, code := range codes {
		quote, err := s.fetchSinaQuote(toSinaCode(code))
		if err != nil {
			continue
		}
		if quote.ChangePercent > 2 && quote.Volume > 0 {
			signals = append(signals, models.WatchScanStock{
				Code:                code,
				Name:                quote.Name,
				Price:               quote.Price,
				ChangePercent:       quote.ChangePercent,
				Score:               60 + quote.ChangePercent*2,
				Sector:              "未知",
				FundHeat:            "活跃",
				RecommendedPosition: 0.2,
				MacdStatus:          "MACD金叉",
				BollStatus:          "BOLL中轨上方",
			})
		}
	}

	return &models.WatchScanResult{Signals: signals}, nil
}

// GetStockScores 获取评分（统一数据源：优先从全A扫描缓存读取，确保全平台一致）
func (s *StockService) GetStockScores(codes []string) map[string]*models.StockScore {
	result := make(map[string]*models.StockScore)

	// 读取缓存（绝不阻塞等待扫描——ScanAllAShares 遍历5000只股票耗时30-120秒）
	s.allStockCacheMu.RLock()
	cache := s.allStockCache
	cacheTime := s.allStockCacheTime
	s.allStockCacheMu.RUnlock()

	// 缓存为空（首次启动）：返回空结果，等待后台扫描填充
	if cache == nil {
		return result
	}

	// 缓存过期超 10 分钟：后台异步触发全量刷新，但当前请求直接返回缓存数据
	// 过期数据远比错误的 fallback 数据可靠
	if time.Since(cacheTime) > 10*time.Minute {
		s.triggerAsyncCacheRefresh()
	}

	// 构建缓存快速查找映射
	cacheMap := make(map[string]models.RankStockItem)
	for _, item := range cache {
		cacheMap[item.Code] = item
	}

	for _, code := range codes {
		if cached, ok := cacheMap[code]; ok {
			result[code] = rankItemToStockScore(cached)
		}
		// 缓存未命中直接跳过，不再使用独立的 fallback 计算
		// 扫描页双引擎结果通过 syncDualEngineScoresToCache 写入此缓存，
		// 排名页全A扫描通过 ScanAllAShares 写入此缓存，
		// 这保证了全平台评分数据源一致
	}

	return result
}

// triggerAsyncCacheRefresh 异步触发全量缓存刷新（无阻塞，防重复）
func (s *StockService) triggerAsyncCacheRefresh() {
	s.allStockCacheMu.Lock()
	if s.refreshRunning {
		s.allStockCacheMu.Unlock()
		return // 已有刷新任务在运行
	}
	s.refreshRunning = true
	s.allStockCacheMu.Unlock()

	go func() {
		defer func() {
			s.allStockCacheMu.Lock()
			s.refreshRunning = false
			s.allStockCacheMu.Unlock()
		}()
		s.ScanAllAShares()
	}()
}

// rankItemToStockScore 将 RankStockItem 映射为统一的 StockScore
func rankItemToStockScore(item models.RankStockItem) *models.StockScore {
	return &models.StockScore{
		TotalScore:      item.TotalScore,
		MarketBaseScore: item.TrendScore + item.MomentumScore + item.VolumeScore,
		TechBonusScore:  item.TechScore,
		TrendScore:      item.TrendScore,
		MomentumScore:   item.MomentumScore,
		VolumeScore:     item.VolumeScore,
		TechScore:       item.TechScore,
		MacdSignal:      item.MacdSignal,
		BollPosition:    item.BollPosition,
		IsGoldenCross:   item.IsGoldenCross,
		IsAboveWater:    item.IsAboveWater,
		Highlights:      item.Highlights,
		Recommendation:  item.Recommendation,
		ResonanceStar:   item.ResonanceStar,
	}
}

// dualEngineStockToRankItem 将双引擎扫描结果转换为全A缓存格式（确保全平台数据源一致）
func dualEngineStockToRankItem(d models.DualEngineStock) models.RankStockItem {
	return models.RankStockItem{
		Code:          d.Code,
		Name:          d.Name,
		Price:         d.Price,
		ChangePercent: d.ChangePercent,
		TurnoverRate:  d.TurnoverRate,
		VolumeRatio:   d.VolumeRatio,
		TotalScore:    d.Score.TotalScore,
		TrendScore:    float64(d.Score.MarketBaseScore) / 3,
		MomentumScore: float64(d.Score.MarketBaseScore) / 3,
		VolumeScore:   float64(d.Score.MarketBaseScore) / 3,
		TechScore:     float64(d.Score.TechBonusScore),
		MacdSignal:    d.MacdSignal,
		BollPosition:  d.BollPosition,
		IsGoldenCross: d.IsGoldenCross,
		IsAboveWater:  d.IsAboveWater,
		Highlights:    d.Score.Highlights,
		Recommendation: d.Recommendation,
	}
}

// syncDualEngineScoresToCache 将双引擎扫描评分合并到全A缓存
// 确保扫描页、持仓页、排名页使用完全一致的评分数据源
func (s *StockService) syncDualEngineScoresToCache(stocks []models.DualEngineStock) {
	s.allStockCacheMu.Lock()
	defer s.allStockCacheMu.Unlock()

	// 构建已有代码→索引映射，用于 upsert
	existingMap := make(map[string]int)
	for i, item := range s.allStockCache {
		existingMap[item.Code] = i
	}

	for _, stock := range stocks {
		rankItem := dualEngineStockToRankItem(stock)
		if idx, ok := existingMap[stock.Code]; ok {
			// 已存在：原地替换为最新扫描结果
			s.allStockCache[idx] = rankItem
		} else {
			// 新股票：追加到缓存
			s.allStockCache = append(s.allStockCache, rankItem)
		}
	}
}

// computeStockScoreFallback 缓存未命中时的回退计算（已废弃，保留仅用于紧急降级）
// 正常路径应通过 GetStockScores → ScanAllAShares → allStockCache 获取评分
func (s *StockService) computeStockScoreFallback(code string) *models.StockScore {
	klines, _ := s.fetchKLineData(code, 60)
	if len(klines) < 10 {
		return nil
	}
	macdData := calcMACD(klines)
	bollData := calcBOLL(klines)
	quote, err := s.fetchSinaQuote(toSinaCode(code))
	if err != nil || quote == nil {
		return nil
	}

	// 使用与 ScanAllAShares 一致的评分维度
	cp := quote.ChangePercent
	tr := quote.TurnoverRate

	trendScore := 0.0
	if cp >= 4 {
		trendScore = 16 + (cp-4)*1.75
	} else if cp >= 2 {
		trendScore = 11 + (cp-2)*2.5
	} else if cp >= 0 {
		trendScore = 5 + cp*3
	} else if cp >= -2 {
		trendScore = 3 + (cp+2)*1
	} else {
		trendScore = math.Max(0, 1+(cp+5)*0.33)
	}

	momentumScore := 0.0
	if tr > 0 {
		if tr >= 15 {
			momentumScore += 15
		} else if tr >= 1 {
			momentumScore += math.Log1p(tr) * 5.5
		}
		amt := quote.Amount
		if amt > 0 {
			amtYi := amt / 1e8
			if amtYi >= 20 {
				momentumScore += 15
			} else if amtYi >= 1 {
				momentumScore += math.Log1p(amtYi) * 4.5
			}
		}
	}

	volumeScore := 0.0
	if tr > 3 {
		volumeScore = 7 + (tr-3)*0.86
	} else if tr > 1 {
		volumeScore = 3 + (tr-1)*2
	} else if tr > 0 {
		volumeScore = tr * 3
	}

	techScore := 0.0
	macdSignal := "未知"
	bollPosition := "待分析"
	isGoldenCross := false
	isAboveWater := false
	var highlights []string

	if macdData != nil {
		if macdData.Dif > macdData.Dea {
			isGoldenCross = true
			if macdData.Dif > 0 {
				isAboveWater = true
				techScore += 20
				macdSignal = "水上金叉"
				highlights = append(highlights, "水上金叉")
			} else {
				techScore += 12
				macdSignal = "水下金叉"
				highlights = append(highlights, "MACD金叉")
			}
		} else {
			gap := macdData.Dea - macdData.Dif
			if gap < 0.05 {
				techScore += 4
				macdSignal = "即将金叉"
			} else {
				macdSignal = "死叉"
			}
		}
	}

	if bollData != nil {
		lastClose := klines[len(klines)-1].Close
		upper := bollData.Upper
		lower := bollData.Lower
		if upper > lower {
			position := (lastClose - lower) / (upper - lower)
			if position >= 0.7 {
				bollPosition = "上轨区域"
				techScore += 8
			} else if position >= 0.5 {
				bollPosition = "中轨上方"
				techScore += 5
			} else if position >= 0.3 {
				bollPosition = "中轨下方"
			} else {
				bollPosition = "下轨区域"
				highlights = append(highlights, "超卖反弹")
				techScore += 3
			}
			if bollData.Bandwidth > 15 {
				techScore += 5
				highlights = append(highlights, "布林开口")
			}
		}
	}

	if cp > 5 {
		highlights = append(highlights, "强势上涨")
	} else if cp > 2 {
		highlights = append(highlights, "稳步上涨")
	}
	if tr > 5 {
		highlights = append(highlights, "资金活跃")
	}

	if len(highlights) > 4 {
		highlights = highlights[:4]
	}
	if highlights == nil {
		highlights = []string{}
	}

	totalScore := math.Min(100, math.Round((trendScore+momentumScore+volumeScore+techScore)*10)/10)
	recommendation := "回避"
	if totalScore >= 75 {
		recommendation = "强烈推荐"
	} else if totalScore >= 60 {
		recommendation = "积极关注"
	} else if totalScore >= 45 {
		recommendation = "一般关注"
	} else if totalScore >= 30 {
		recommendation = "观望"
	} else if totalScore >= 15 {
		recommendation = "谨慎观望"
	}

	return &models.StockScore{
		TotalScore:      totalScore,
		MarketBaseScore: math.Round((trendScore+momentumScore+volumeScore)*10) / 10,
		TechBonusScore:  math.Round(techScore*10) / 10,
		TrendScore:      math.Round(trendScore*10) / 10,
		MomentumScore:   math.Round(momentumScore*10) / 10,
		VolumeScore:     math.Round(volumeScore*10) / 10,
		TechScore:       math.Round(techScore*10) / 10,
		MacdSignal:      macdSignal,
		BollPosition:    bollPosition,
		IsGoldenCross:   isGoldenCross,
		IsAboveWater:    isAboveWater,
		Highlights:      highlights,
		Recommendation:  recommendation,
	}
}

// GetUnifiedScore 获取单只股票的统一评分（数据源 = 全A扫描缓存）
func (s *StockService) GetUnifiedScore(code string) *models.StockScore {
	result := s.GetStockScores([]string{code})
	if score, ok := result[code]; ok {
		return score
	}
	return nil
}

// GetPositionHealthScore 获取持仓健康度评分（含成本价个性化调整）
func (s *StockService) GetPositionHealthScore(code string, costPrice float64) *models.StockScore {
	baseScore := s.GetUnifiedScore(code)
	if baseScore == nil {
		return nil
	}

	// 获取当前行情
	quote, err := s.fetchSinaQuote(toSinaCode(code))
	if err != nil || quote == nil || quote.Price <= 0 || costPrice <= 0 {
		baseScore.PositionHealthScore = baseScore.TotalScore
		baseScore.PositionHealthLabel = "市场评分"
		return baseScore
	}

	// 计算盈亏百分比
	profitPercent := (quote.Price - costPrice) / costPrice * 100

	// 持仓健康度 = 市场评分基础上根据盈亏调整（±15分范围）
	healthAdjust := 0.0
	if profitPercent > 20 {
		healthAdjust = 15 // 大幅盈利，强烈建议关注止盈
	} else if profitPercent > 10 {
		healthAdjust = 10
	} else if profitPercent > 5 {
		healthAdjust = 5
	} else if profitPercent > 0 {
		healthAdjust = 2
	} else if profitPercent > -5 {
		healthAdjust = -3 // 小幅亏损，需要关注
	} else if profitPercent > -10 {
		healthAdjust = -8
	} else {
		healthAdjust = -15 // 大幅亏损，需止损
	}

	healthScore := math.Min(100, math.Max(0, baseScore.TotalScore+healthAdjust))
	baseScore.PositionHealthScore = math.Round(healthScore*10) / 10

	if profitPercent > 10 {
		baseScore.PositionHealthLabel = "持仓健康度（建议关注止盈）"
	} else if profitPercent > 0 {
		baseScore.PositionHealthLabel = "持仓健康度（盈利持有）"
	} else if profitPercent > -5 {
		baseScore.PositionHealthLabel = "持仓健康度（小幅浮亏）"
	} else {
		baseScore.PositionHealthLabel = "持仓健康度（注意风险）"
	}

	return baseScore
}

// SearchStocks 搜索股票
func (s *StockService) SearchStocks(keyword string) []models.SearchItem {
	var result []models.SearchItem

	if len(keyword) < 1 {
		return result
	}

	// 优先从全A股缓存中搜索（包含约5000支股票的代码和名称）
	s.allStockCacheMu.RLock()
	cache := s.allStockCache
	s.allStockCacheMu.RUnlock()

	if len(cache) > 0 {
		keywordLower := strings.ToLower(keyword)
		for _, item := range cache {
			if strings.Contains(item.Code, keywordLower) || strings.Contains(item.Name, keyword) {
				result = append(result, models.SearchItem{Code: item.Code, Name: item.Name})
				if len(result) >= 20 {
					break
				}
			}
		}
		return result
	}

	// 缓存未命中时，从新浪API实时获取全A股列表进行搜索
	allCodes, err := s.FetchAllAShareCodes()
	if err != nil || len(allCodes) == 0 {
		allCodes = getDefaultStockPool()
	}

	// 先按代码匹配
	matchedCodes := make([]string, 0)
	for _, code := range allCodes {
		if strings.Contains(code, keyword) {
			matchedCodes = append(matchedCodes, code)
			if len(matchedCodes) >= 50 {
				break
			}
		}
	}

	// 批量获取匹配代码的行情（包含名称）
	if len(matchedCodes) > 0 {
		quotes := s.fetchBatchQuotesConcurrent(matchedCodes, 50, 10)
		for _, q := range quotes {
			if q.Name != "" && (strings.Contains(q.Code, keyword) || strings.Contains(q.Name, keyword)) {
				result = append(result, models.SearchItem{Code: q.Code, Name: q.Name})
			}
		}
	}

	// 如果代码没匹配到，按名称搜索（取前200支批量获取名称）
	if len(result) == 0 {
		searchPool := allCodes
		if len(searchPool) > 200 {
			searchPool = searchPool[:200]
		}
		quotes := s.fetchBatchQuotesConcurrent(searchPool, 50, 10)
		for _, q := range quotes {
			if q.Name != "" && strings.Contains(q.Name, keyword) {
				result = append(result, models.SearchItem{Code: q.Code, Name: q.Name})
				if len(result) >= 20 {
					break
				}
			}
		}
	}

	return result
}

// ======== Private Helper Methods ========

// decodeGBK GBK/GB18030转UTF-8
func decodeGBK(data []byte) string {
	// 首先检查是否为有效的UTF-8，如果是则直接返回
	if utf8.Valid(data) {
		return string(data)
	}
	// 优先使用GB18030解码（新浪API返回GB18030）
	decoder := simplifiedchinese.GB18030.NewDecoder()
	decoded, err := decoder.Bytes(data)
	if err == nil {
		return string(decoded)
	}
	// 回退到GBK
	decoder = simplifiedchinese.GBK.NewDecoder()
	decoded, err = decoder.Bytes(data)
	if err == nil {
		return string(decoded)
	}
	// 最后尝试按流式解码
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder())
	decoded2, err2 := io.ReadAll(reader)
	if err2 != nil {
		return string(data)
	}
	return string(decoded2)
}

func (s *StockService) fetchSinaQuote(sinaCode string) (*models.SinaQuote, error) {
	url := fmt.Sprintf("https://hq.sinajs.cn/list=%s", sinaCode)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Referer", "https://finance.sina.com.cn")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseSinaQuote(decodeGBK(body), sinaCode)
}

func (s *StockService) fetchBatchSinaQuotes(sinaCodes []string) ([]models.SinaQuote, error) {
	codeStr := strings.Join(sinaCodes, ",")
	url := fmt.Sprintf("https://hq.sinajs.cn/list=%s", codeStr)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Referer", "https://finance.sina.com.cn")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseBatchSinaQuotes(decodeGBK(body))
}

func (s *StockService) fetchBatchQuotesFromPool(codes []string) []models.SinaQuote {
	batchSize := 30
	var allQuotes []models.SinaQuote

	for i := 0; i < len(codes); i += batchSize {
		end := i + batchSize
		if end > len(codes) {
			end = len(codes)
		}
		batch := codes[i:end]
		sinaCodes := make([]string, len(batch))
		for j, code := range batch {
			sinaCodes[j] = toSinaCode(code)
		}
		quotes, err := s.fetchBatchSinaQuotes(sinaCodes)
		if err != nil {
			continue
		}
		allQuotes = append(allQuotes, quotes...)
	}

	return allQuotes
}

func (s *StockService) fetchKLineData(code string, count int) ([]models.KLineData, error) {
	// 使用腾讯K线API（新浪API已被封禁）
	prefix := "sz"
	if strings.HasPrefix(code, "6") || strings.HasPrefix(code, "9") {
		prefix = "sh"
	}
	symbol := prefix + code

	// 计算日期范围
	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, -4, 0).Format("2006-01-02") // 4个月前
	url := fmt.Sprintf("https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param=%s,day,%s,%s,%d,qfq", symbol, startDate, endDate, count)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 腾讯API返回JSON: {"code":0,"data":{"sh600000":{"qfqday":[[date,open,close,high,low,volume],...]}}}
	var result struct {
		Code int `json:"code"`
		Data map[string]struct {
			Qfqday [][]string `json:"qfqday"`
			Day     [][]string `json:"day"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 查找数据
	var rawData [][]string
	for _, v := range result.Data {
		if len(v.Qfqday) > 0 {
			rawData = v.Qfqday
		} else if len(v.Day) > 0 {
			rawData = v.Day
		}
		break
	}

	if len(rawData) == 0 {
		return nil, fmt.Errorf("no kline data")
	}

	var klines []models.KLineData
	for _, item := range rawData {
		if len(item) < 6 {
			continue
		}
		open, _ := strconv.ParseFloat(item[1], 64)
		close_, _ := strconv.ParseFloat(item[2], 64)
		high, _ := strconv.ParseFloat(item[3], 64)
		low, _ := strconv.ParseFloat(item[4], 64)
		vol, _ := strconv.ParseFloat(item[5], 64)
		klines = append(klines, models.KLineData{
			Date:   item[0],
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close_,
			Volume: vol,
		})
	}
	return klines, nil
}

func (s *StockService) buildDualEngineStock(q models.SinaQuote, macdData *MacdResult, bollData *BollResult, klines []models.KLineData, rank int) models.DualEngineStock {
	marketScore := 0
	techScore := 0
	var highlights []string

	// Market base score
	if q.ChangePercent > 5 {
		marketScore += 20
		highlights = append(highlights, "强势上涨")
	} else if q.ChangePercent > 2 {
		marketScore += 15
	} else if q.ChangePercent > 0 {
		marketScore += 10
	}

	if q.TurnoverRate > 5 {
		marketScore += 20
		highlights = append(highlights, "资金爆量")
	} else if q.TurnoverRate > 3 {
		marketScore += 15
	}

	sectorScore := 0
	if rank <= 3 {
		sectorScore = 15
		highlights = append(highlights, "板块龙头")
	}

	marketScore += sectorScore

	// Tech score
	macdSignalStr := "死叉"
	isGoldenCross := false
	isAboveWater := false
	bandwidthRatio := 0.0
	dif := 0.0
	macd := 0.0

	if macdData != nil {
		dif = macdData.Dif
		macd = macdData.Macd
		if macdData.Dif > macdData.Dea {
			isGoldenCross = true
			techScore += 15
			if macdData.Dif > 0 {
				isAboveWater = true
				techScore += 10
				macdSignalStr = "水上金叉"
				highlights = append(highlights, "水上金叉")
			} else {
				macdSignalStr = "水下金叉"
			}
		} else {
			macdSignalStr = "死叉"
		}
	}

	bollPosition := "待分析"
	if bollData != nil && len(klines) > 0 {
		lastClose := klines[len(klines)-1].Close
		if bollData.Upper > 0 {
			bandwidthRatio = (bollData.Upper - bollData.Lower) / bollData.Middle * 100
		}
		if bollData.Upper > bollData.Lower {
			pos := (lastClose - bollData.Lower) / (bollData.Upper - bollData.Lower)
			if pos >= 0.7 {
				bollPosition = "上轨区域"
				techScore += 10
			} else if pos >= 0.5 {
				bollPosition = "中轨上方"
				techScore += 5
			} else if pos >= 0.3 {
				bollPosition = "中轨下方"
			} else {
				bollPosition = "下轨区域"
				highlights = append(highlights, "超卖反弹")
				techScore += 3
			}
		}
		if bandwidthRatio > 15 {
			techScore += 10
			highlights = append(highlights, "布林开口")
		}
	}

	volumeRatio := 0.0
	if len(klines) >= 5 {
		avgVol := 0.0
		for i := len(klines) - 5; i < len(klines); i++ {
			avgVol += klines[i].Volume
		}
		avgVol /= 5
		if avgVol > 0 && q.Volume > 0 {
			volumeRatio = q.Volume / avgVol
		}
	}

	if volumeRatio > 3 {
		highlights = append(highlights, "量能爆发")
	}

	totalScore := float64(marketScore + techScore)
	if len(highlights) > 4 {
		highlights = highlights[:4]
	}
	if highlights == nil {
		highlights = []string{}
	}

	recommendedPos := 0.15
	recommendation := "观望"
	if totalScore >= 75 {
		recommendedPos = 0.30
		recommendation = "强烈推荐"
	} else if totalScore >= 60 {
		recommendedPos = 0.25
		recommendation = "积极关注"
	} else if totalScore >= 45 {
		recommendedPos = 0.20
		recommendation = "一般关注"
	} else if totalScore >= 30 {
		recommendedPos = 0.15
		recommendation = "观望"
	} else if totalScore >= 15 {
		recommendedPos = 0.10
		recommendation = "谨慎观望"
	} else {
		recommendedPos = 0.05
		recommendation = "回避"
	}

	return models.DualEngineStock{
		Code:          q.Code,
		Name:          q.Name,
		Price:         q.Price,
		ChangePercent: q.ChangePercent,
		Sector:        "热门板块",
		VolumeRatio:   volumeRatio,
		TurnoverRate:  q.TurnoverRate,
		BandwidthRatio: bandwidthRatio,
		Dif:            dif,
		Macd:           macd,
		MacdSignal:     macdSignalStr,
		BollPosition:   bollPosition,
		IsGoldenCross:  isGoldenCross,
		IsAboveWater:   isAboveWater,
		Recommendation: recommendation,
		Score: models.DualEngineScore{
			SectorHeatScore:      sectorScore,
			SectorPositionScore:  sectorScore,
			CapitalStrengthScore: marketScore - sectorScore,
			MarketBaseScore:      marketScore,
			BollTrendScore:       techScore / 3,
			MacdSignalScore:      techScore / 3,
			SignalConfirmScore:   techScore / 3,
			TechBonusScore:       techScore,
			TotalScore:           totalScore,
			Highlights:           highlights,
		},
		RecommendedPosition: float64(recommendedPos),
	}
}

// ======== Parsing Functions ========

func parseSinaQuote(content, sinaCode string) (*models.SinaQuote, error) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if !strings.Contains(line, sinaCode) {
			continue
		}
		parts := strings.SplitN(line, "\"", 3)
		if len(parts) < 3 {
			continue
		}
		dataStr := strings.TrimSpace(parts[1])
		if dataStr == "" {
			continue
		}
		return parseSinaQuoteData(dataStr, sinaCode)
	}
	return nil, fmt.Errorf("no data for %s", sinaCode)
}

func parseBatchSinaQuotes(content string) ([]models.SinaQuote, error) {
	var quotes []models.SinaQuote
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if !strings.Contains(line, "hq_str_") {
			continue
		}
		re := regexp.MustCompile(`hq_str_([a-zA-Z0-9]+)="(.*)"`)
		matches := re.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}
		sinaCode := matches[1]
		dataStr := strings.TrimSpace(matches[2])
		if dataStr == "" {
			continue
		}
		quote, err := parseSinaQuoteData(dataStr, sinaCode)
		if err != nil {
			continue
		}
		quotes = append(quotes, *quote)
	}
	return quotes, nil
}

func parseSinaQuoteData(dataStr, sinaCode string) (*models.SinaQuote, error) {
	fields := strings.Split(dataStr, ",")
	if len(fields) < 32 {
		return nil, fmt.Errorf("insufficient data fields for %s", sinaCode)
	}

	name := fields[0]
	open, _ := strconv.ParseFloat(fields[1], 64)
	prevClose, _ := strconv.ParseFloat(fields[2], 64)
	price, _ := strconv.ParseFloat(fields[3], 64)
	high, _ := strconv.ParseFloat(fields[4], 64)
	low, _ := strconv.ParseFloat(fields[5], 64)
	volume, _ := strconv.ParseFloat(fields[8], 64)
	amount, _ := strconv.ParseFloat(fields[9], 64)

	change := 0.0
	changePercent := 0.0
	if prevClose > 0 {
		change = price - prevClose
		changePercent = (change / prevClose) * 100
	}

	code := strings.TrimPrefix(sinaCode, "sh")
	code = strings.TrimPrefix(code, "sz")

	return &models.SinaQuote{
		Code:           code,
		Name:           name,
		Open:           open,
		PrevClose:      prevClose,
		Price:          price,
		High:           high,
		Low:            low,
		Volume:         volume,
		Amount:         amount,
		Date:           fields[30],
		Time:           fields[31],
		Change:         change,
		ChangePercent:  changePercent,
	}, nil
}

// ======== Technical Analysis ========

type MacdResult struct {
	Dif          float64
	Dea          float64
	Macd         float64
	Hist         float64
	PrevHist     float64
	Status       string
	Signal       string
	AxisPosition string
}

type BollResult struct {
	Upper     float64
	Middle    float64
	Lower     float64
	Bandwidth float64
}

func calcMACD(klines []models.KLineData) *MacdResult {
	if len(klines) < 26 {
		return &MacdResult{Status: "数据不足", Signal: "未知", AxisPosition: "未知"}
	}

	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}

	ema12 := calcEMA(closes, 12)
	ema26 := calcEMA(closes, 26)

	difLine := make([]float64, len(ema12))
	for i := range ema12 {
		difLine[i] = ema12[i] - ema26[i]
	}

	deaLine := calcEMA(difLine, 9)

	lastIdx := len(difLine) - 1
	dif := difLine[lastIdx]
	dea := deaLine[lastIdx]
	hist := (dif - dea) * 2
	macd := hist

	// 计算前一日柱状图
	prevHist := 0.0
	if lastIdx >= 1 {
		prevHist = (difLine[lastIdx-1] - deaLine[lastIdx-1]) * 2
	}

	signal := "死叉"
	if dif > dea {
		signal = "金叉"
	}

	axisPos := "零轴下方"
	if dif > 0 && dea > 0 {
		axisPos = "零轴上方(多头排列)"
	} else if dif > 0 {
		axisPos = "零轴上方"
	}

	status := signal
	if macd > 0 {
		status += "，红柱"
	} else {
		status += "，绿柱"
	}

	return &MacdResult{
		Dif:          math.Round(dif*1000) / 1000,
		Dea:          math.Round(dea*1000) / 1000,
		Macd:         math.Round(macd*1000) / 1000,
		Hist:         math.Round(hist*1000) / 1000,
		PrevHist:     math.Round(prevHist*1000) / 1000,
		Status:       status,
		Signal:       signal,
		AxisPosition: axisPos,
	}
}

func calcEMA(data []float64, period int) []float64 {
	result := make([]float64, len(data))
	if len(data) == 0 {
		return result
	}
	multiplier := 2.0 / float64(period+1)
	result[0] = data[0]
	for i := 1; i < len(data); i++ {
		result[i] = (data[i]-result[i-1])*multiplier + result[i-1]
	}
	return result
}

func calcBOLL(klines []models.KLineData) *BollResult {
	n := 20
	if len(klines) < n {
		return nil
	}

	recent := klines[len(klines)-n:]
	sum := 0.0
	for _, k := range recent {
		sum += k.Close
	}
	middle := sum / float64(n)

	variance := 0.0
	for _, k := range recent {
		variance += math.Pow(k.Close-middle, 2)
	}
	stdDev := math.Sqrt(variance / float64(n))

	upper := middle + 2*stdDev
	lower := middle - 2*stdDev

	bandwidth := 0.0
	if middle > 0 {
		bandwidth = (upper - lower) / middle * 100
	}

	return &BollResult{
		Upper:     math.Round(upper*100) / 100,
		Middle:    math.Round(middle*100) / 100,
		Lower:     math.Round(lower*100) / 100,
		Bandwidth: math.Round(bandwidth*100) / 100,
	}
}

// toSinaCode converts stock code to Sina API format
func toSinaCode(code string) string {
	if strings.HasPrefix(code, "sh") || strings.HasPrefix(code, "sz") {
		return code
	}
	if strings.HasPrefix(code, "6") || strings.HasPrefix(code, "9") {
		return "sh" + code
	}
	return "sz" + code
}

// fromSinaCode converts Sina API format to plain stock code
func fromSinaCode(sinaCode string) string {
	code := strings.TrimPrefix(sinaCode, "sh")
	code = strings.TrimPrefix(code, "sz")
	return code
}

// FetchAllAShareCodes 获取全A股代码列表
func (s *StockService) FetchAllAShareCodes() ([]string, error) {
	nodes := []string{"sh_a", "sz_a"}
	var allCodes []string

	for _, node := range nodes {
		codes, err := s.fetchStockListPaged(node)
		if err != nil {
			continue
		}
		allCodes = append(allCodes, codes...)
	}

	if len(allCodes) == 0 {
		return getDefaultStockPool(), nil
	}

	return allCodes, nil
}

// generateAllAShareCodes 生成全A股候选代码（覆盖沪市主板/科创板、深市主板/创业板/中小板、北交所）
func generateAllAShareCodes() []string {
	var codes []string
	// 沪市主板：600000-605999
	for i := 600000; i <= 605999; i++ {
		codes = append(codes, fmt.Sprintf("%06d", i))
	}
	// 科创板：688000-689999
	for i := 688000; i <= 689999; i++ {
		codes = append(codes, fmt.Sprintf("%06d", i))
	}
	// 深市主板：000001-004999
	for i := 1; i <= 4999; i++ {
		codes = append(codes, fmt.Sprintf("%06d", i))
	}
	// 中小板：002001-004999（已被深市主板覆盖，但仍保留）
	// 创业板：300000-301999
	for i := 300000; i <= 301999; i++ {
		codes = append(codes, fmt.Sprintf("%06d", i))
	}
	// 北交所：涉及8开头、4开头等，保守覆盖
	for i := 800000; i <= 839999; i++ {
		codes = append(codes, fmt.Sprintf("%06d", i))
	}
	for i := 400000; i <= 439999; i++ {
		codes = append(codes, fmt.Sprintf("%06d", i))
	}
	return codes
}

// fetchAllAShareWithQuotes 获取全A股代码和行情数据
// push2.eastmoney.com 存在Go TLS兼容问题（EOF），改用新浪hq.sinajs.cn批量API
func (s *StockService) fetchAllAShareWithQuotes() ([]models.SinaQuote, error) {
	codes := generateAllAShareCodes()
	log.Printf("[INFO] fetchAllAShareWithQuotes: %d candidate codes to batch-query", len(codes))

	// 批量从新浪行情API获取实时行情（每批300支，兼容性好）
	batchSize := 300
	var allQuotes []models.SinaQuote

	for i := 0; i < len(codes); i += batchSize {
		end := i + batchSize
		if end > len(codes) {
			end = len(codes)
		}
		sinaCodes := make([]string, end-i)
		for j, code := range codes[i:end] {
			sinaCodes[j] = toSinaCode(code)
		}

		codeStr := strings.Join(sinaCodes, ",")
		url := fmt.Sprintf("https://hq.sinajs.cn/list=%s", codeStr)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Referer", "https://finance.sina.com.cn")
		req.Header.Set("User-Agent", "Mozilla/5.0")

		var resp *http.Response
		var err error
		for retry := 0; retry < 2; retry++ {
			resp, err = s.scanHttpClient.Do(req)
			if err == nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if err != nil {
			log.Printf("[WARN] Sina batch failed at offset=%d: %v", i, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		quotes, _ := parseBatchSinaQuotes(decodeGBK(body))
		for _, q := range quotes {
			if q.Price > 0 && q.Name != "" && !strings.Contains(q.Name, "ST") && !strings.Contains(q.Name, "st") && !strings.Contains(q.Name, "退") {
				allQuotes = append(allQuotes, q)
			}
		}

		if (i/batchSize+1)%10 == 0 {
			log.Printf("[INFO] Sina batch progress: %d/%d, valid=%d", end, len(codes), len(allQuotes))
		}
		time.Sleep(50 * time.Millisecond)
	}

	log.Printf("[INFO] fetchAllAShareWithQuotes total: %d valid quotes from %d candidates", len(allQuotes), len(codes))
	return allQuotes, nil
}

func (s *StockService) fetchStockListPaged(node string) ([]string, error) {
	var allCodes []string
	// 增大页尺寸200（原80），减少请求次数；延迟50ms（原500ms）
	pageSize := 200

	for page := 1; ; page++ {
		url := fmt.Sprintf("https://vip.stock.finance.sina.com.cn/quotes_service/api/json_v2.php/Market_Center.getHQNodeData?page=%d&num=%d&sort=symbol&asc=1&node=%s", page, pageSize, node)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Referer", "https://finance.sina.com.cn")
		req.Header.Set("User-Agent", "Mozilla/5.0")

		// 使用 scanHttpClient（绕代理+120s超时）
		resp, err := s.scanHttpClient.Do(req)
		if err != nil {
			break
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			break
		}

		// 新浪API可能返回GBK编码
		decodedBody := decodeGBK(body)

		var items []struct {
			Symbol string `json:"symbol"`
			Code   string `json:"code"`
			Name   string `json:"name"`
		}
		if err := json.Unmarshal([]byte(decodedBody), &items); err != nil {
			// JSON解析失败，重试一次
			time.Sleep(100 * time.Millisecond)
			resp2, err2 := s.scanHttpClient.Do(req)
			if err2 != nil {
				continue
			}
			body2, _ := io.ReadAll(resp2.Body)
			resp2.Body.Close()
			decodedBody2 := decodeGBK(body2)
			if err2 := json.Unmarshal([]byte(decodedBody2), &items); err2 != nil {
				continue
			}
		}

		if len(items) == 0 {
			break
		}

		for _, item := range items {
			if item.Symbol != "" {
				code := fromSinaCode(item.Symbol)
				if len(code) == 6 {
					allCodes = append(allCodes, code)
				}
			}
		}

		if len(items) < pageSize {
			break
		}

		if page > 100 {
			break
		}

		// 每页之间添加微延迟避免被限流
		time.Sleep(50 * time.Millisecond)
	}

	return allCodes, nil
}

// ScanAllAShares 全A股扫描评分排名（精细化评分+MACD/BOLL深度分析）
func (s *StockService) ScanAllAShares() (*models.AllStockScanResult, error) {
	startTime := time.Now()

	// 检查缓存（5分钟内有效）
	s.allStockCacheMu.RLock()
	if s.allStockCache != nil && time.Since(s.allStockCacheTime) < 5*time.Minute {
		cached := s.allStockCache
		s.allStockCacheMu.RUnlock()
		topList := make([]models.RankStockItem, 0, 50)
		for i := range cached {
			if i >= 50 {
				break
			}
			topList = append(topList, cached[i])
		}
		return &models.AllStockScanResult{
			TotalStocks:    len(cached),
			ValidStocks:    len(cached),
			AnalyzedStocks: len(cached),
			ScanTime:       s.allStockCacheTime.Format("2006-01-02 15:04:05"),
			CostMs:         0,
			TopList:        topList,
		}, nil
	}
	s.allStockCacheMu.RUnlock()

	// Step 2: 获取全A股行情数据（含换手率）
	quotes, err := s.fetchAllAShareWithQuotes()
	if err != nil {
		return nil, err
	}
	totalStocks := len(quotes)

	// 构建行业映射（从名称推断，Sina API不返回行业字段）
	codeIndustryMap := s.inferIndustryMap(quotes)
	validStocks := len(quotes)

	// Step 3: 精细化评分（连续化评分，避免离散阶梯）
	type stockWithScore struct {
		quote         models.SinaQuote
		trendScore    float64
		momentumScore float64
		volumeScore   float64
	}

	scored := make([]stockWithScore, 0, validStocks)
	for _, q := range quotes {
		if q.Price <= 0 || q.PrevClose <= 0 {
			continue
		}
		if strings.Contains(q.Name, "ST") || strings.Contains(q.Name, "st") || strings.Contains(q.Name, "退") {
			continue
		}
		if q.ChangePercent >= 9.9 || q.ChangePercent <= -9.9 {
			continue
		}

		// === 趋势得分（25分制，连续化） ===
		trendScore := 0.0
		cp := q.ChangePercent
		if cp >= 8 {
			trendScore = 23 + math.Min(2, (cp-8)*0.5)
		} else if cp >= 4 {
			trendScore = 16 + (cp-4)*1.75
		} else if cp >= 2 {
			trendScore = 11 + (cp-2)*2.5
		} else if cp >= 0 {
			trendScore = 5 + cp*3
		} else if cp >= -2 {
			trendScore = 3 + (cp+2)*1
		} else if cp >= -5 {
			trendScore = math.Max(0, 1+(cp+5)*0.33)
		} else {
			trendScore = 0
		}

		// === 动量得分（35分制，连续化） ===
		momentumScore := 0.0
		tr := q.TurnoverRate
		if tr > 0 {
			if tr >= 15 {
				momentumScore += 15
			} else if tr >= 1 {
				momentumScore += math.Log1p(tr) * 5.5
			} else {
				momentumScore += tr * 2
			}

			amt := q.Amount
			if amt > 0 {
				amtYi := amt / 1e8
				if amtYi >= 20 {
					momentumScore += 15
				} else if amtYi >= 1 {
					momentumScore += math.Log1p(amtYi) * 4.5
				} else {
					momentumScore += amtYi * 2
				}
			}

			if cp > 3 && tr > 5 {
				momentumScore += 5
			} else if cp > 1 && tr > 2 {
				momentumScore += 3
			} else if cp > 0 && tr > 1 {
				momentumScore += 1.5
			}
		}

		// === 量能得分（15分制，连续化） ===
		volumeScore := 0.0
		if q.Volume > 0 && q.Amount > 0 && q.PrevClose > 0 {
			if tr > 10 {
				volumeScore = 13 + math.Min(2, (tr-10)*0.2)
			} else if tr > 3 {
				volumeScore = 7 + (tr-3)*0.86
			} else if tr > 1 {
				volumeScore = 3 + (tr-1)*2
			} else if tr > 0 {
				volumeScore = tr * 3
			}
			if cp < 0 && tr < 0.5 && cp > -3 {
				volumeScore += 1.5
			}
		}

		scored = append(scored, stockWithScore{
			quote:         q,
			trendScore:    math.Round(trendScore*10) / 10,
			momentumScore: math.Round(momentumScore*10) / 10,
			volumeScore:   math.Round(volumeScore*10) / 10,
		})
	}

	// Step 4: 按初步得分排序，取Top500做深度分析
	sort.Slice(scored, func(i, j int) bool {
		si := scored[i].trendScore + scored[i].momentumScore + scored[i].volumeScore
		sj := scored[j].trendScore + scored[j].momentumScore + scored[j].volumeScore
		return si > sj
	})

	deepCount := 500
	if len(scored) < deepCount {
		deepCount = len(scored)
	}

	// Step 5: 对Top500做MACD+BOLL深度分析（并发20组）
	type deepResult struct {
		index    int
		macdData *MacdResult
		bollData *BollResult
		klines   []models.KLineData
	}

	deepResults := make([]deepResult, deepCount)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20)

	for i := 0; i < deepCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			code := scored[idx].quote.Code
			klines, _ := s.fetchKLineData(code, 60)
			macdData := calcMACD(klines)
			bollData := calcBOLL(klines)
			deepResults[idx] = deepResult{index: idx, macdData: macdData, bollData: bollData, klines: klines}
		}(i)
	}
	wg.Wait()

	// Step 6: 综合评分
	allRanked := make([]models.RankStockItem, 0, len(scored))

	for i := 0; i < len(scored); i++ {
		item := scored[i]
		q := item.quote

		techScore := 0.0
		macdSignal := "待分析"
		bollPosition := "待分析"
		isGoldenCross := false
		isAboveWater := false
		var highlights []string
		volumeRatio := 0.0

		if i < deepCount {
			dr := deepResults[i]
			if dr.macdData != nil && len(dr.klines) >= 12 {
				dif := dr.macdData.Dif
				dea := dr.macdData.Dea
				hist := dr.macdData.Hist

				if dif > dea {
					isGoldenCross = true
					gap := dif - dea
					if dif > 0 {
						isAboveWater = true
						techScore += 12 + math.Min(8, gap*4)
						macdSignal = "水上金叉"
						highlights = append(highlights, "水上金叉")
					} else {
						techScore += 6 + math.Min(6, gap*3)
						macdSignal = "水下金叉"
						highlights = append(highlights, "MACD金叉")
					}
					if hist > 0 {
						if dr.macdData.PrevHist >= 0 && hist > dr.macdData.PrevHist {
							techScore += 2
							highlights = append(highlights, "红柱放大")
						}
					}
				} else {
					gap := dea - dif
					if gap < 0.05 {
						techScore += 4
						macdSignal = "即将金叉"
						highlights = append(highlights, "即将金叉")
					} else if gap < 0.2 {
						techScore += 2
						macdSignal = "弱死叉"
					} else {
						macdSignal = "死叉"
					}
				}
			}

			if dr.bollData != nil && len(dr.klines) > 0 {
				lastClose := dr.klines[len(dr.klines)-1].Close
				upper := dr.bollData.Upper
				lower := dr.bollData.Lower
				bw := dr.bollData.Bandwidth

				if upper > lower {
					position := (lastClose - lower) / (upper - lower)
					if position >= 1.0 {
						bollPosition = "突破上轨"
						techScore += 12 + math.Min(3, (position-1)*10)
						highlights = append(highlights, "突破上轨")
					} else if position >= 0.7 {
						bollPosition = "上轨区域"
						techScore += 8 + (position-0.7)*13.3
					} else if position >= 0.5 {
						bollPosition = "中轨上方"
						techScore += 5 + (position-0.5)*15
					} else if position >= 0.3 {
						bollPosition = "中轨下方"
						techScore += 2 + (position-0.3)*15
					} else if position >= 0 {
						bollPosition = "下轨区域"
						techScore += position * 6.7
					} else {
						bollPosition = "跌破下轨"
						highlights = append(highlights, "超卖反弹")
						techScore += 3
					}

					if bw > 20 {
						techScore += 5
						highlights = append(highlights, "布林开口")
					} else if bw > 10 {
						techScore += 2 + (bw-10)*0.3
					}
				}
			}

			// 量比计算（腾讯K线成交量单位为手=100股，需要转换）
			if len(dr.klines) >= 5 && q.Volume > 0 {
				avgVol := 0.0
				count := 0
				startJ := len(dr.klines) - 5
				if startJ < 0 {
					startJ = 0
				}
				for j := startJ; j < len(dr.klines); j++ {
				avgVol += dr.klines[j].Volume * 100 // 手转换为股
					count++
				}
				if count > 0 {
					avgVol /= float64(count)
				}
				if avgVol > 0 {
					volumeRatio = math.Round(q.Volume/avgVol*100) / 100
					if volumeRatio >= 5 {
						techScore += 8 + math.Min(2, (volumeRatio-5)*0.4)
						highlights = append(highlights, "量能爆发")
					} else if volumeRatio >= 2 {
						techScore += 4 + (volumeRatio-2)*1.33
						if volumeRatio >= 3 {
							highlights = append(highlights, "明显放量")
						} else {
							highlights = append(highlights, "温和放量")
						}
					} else if volumeRatio >= 1 {
						techScore += (volumeRatio - 1) * 4
					}
				}
			}
		} else {
			// 非Top500，使用简化技术评分
			if q.ChangePercent > 5 {
				techScore = 12
			} else if q.ChangePercent > 3 {
				techScore = 8
			} else if q.ChangePercent > 1 {
				techScore = 4
			} else if q.ChangePercent > 0 {
				techScore = 2
			}
		}

		techScore = math.Round(techScore*10) / 10
		totalScore := item.trendScore + item.momentumScore + item.volumeScore + techScore
		totalScore = math.Min(100, math.Round(totalScore*10)/10)

		if q.ChangePercent > 7 {
			highlights = append(highlights, "强势大涨")
		} else if q.ChangePercent > 3 {
			highlights = append(highlights, "稳步上涨")
		}
		if q.TurnoverRate > 8 {
			highlights = append(highlights, "资金活跃")
		} else if q.TurnoverRate > 4 {
			highlights = append(highlights, "资金关注")
		}
		if q.Amount > 2e9 {
			highlights = append(highlights, "超大成交")
		} else if q.Amount > 1e9 {
			highlights = append(highlights, "大额成交")
		}
		if len(highlights) > 4 {
			highlights = highlights[:4]
		}
		if highlights == nil {
			highlights = []string{}
		}

		recommendation := "回避"
		if totalScore >= 75 {
			recommendation = "强烈推荐"
		} else if totalScore >= 60 {
			recommendation = "积极关注"
		} else if totalScore >= 45 {
			recommendation = "一般关注"
		} else if totalScore >= 30 {
			recommendation = "观望"
		} else if totalScore >= 15 {
			recommendation = "谨慎观望"
		}

		industry := codeIndustryMap[q.Code]

		// 计算共振星级（每10分半星，满分5星）
		resonanceStar := math.Min(5, math.Round(totalScore/10)/2)

		var macdDif, macdDea, macdHistogram, bollUpper, bollMiddle, bollLower float64
		if i < deepCount {
			dr := deepResults[i]
			if dr.macdData != nil {
				macdDif = math.Round(dr.macdData.Dif*100) / 100
				macdDea = math.Round(dr.macdData.Dea*100) / 100
				macdHistogram = math.Round(dr.macdData.Hist*100) / 100
			}
			if dr.bollData != nil {
				bollUpper = math.Round(dr.bollData.Upper*100) / 100
				bollMiddle = math.Round(dr.bollData.Middle*100) / 100
				bollLower = math.Round(dr.bollData.Lower*100) / 100
			}
		}

		allRanked = append(allRanked, models.RankStockItem{
			Rank:            0,
			Code:            q.Code,
			Name:            q.Name,
			Price:           q.Price,
			ChangePercent:   q.ChangePercent,
			TurnoverRate:    q.TurnoverRate,
			VolumeRatio:     volumeRatio,
			TotalScore:      totalScore,
			TrendScore:      item.trendScore,
			MomentumScore:   item.momentumScore,
			VolumeScore:     item.volumeScore,
			TechScore:       techScore,
			MacdSignal:      macdSignal,
			BollPosition:    bollPosition,
			IsGoldenCross:   isGoldenCross,
			IsAboveWater:    isAboveWater,
			Highlights:      highlights,
			Recommendation:  recommendation,
			Industry:        industry,
			ResonanceStar:   resonanceStar,
			MacdDif:         macdDif,
			MacdDea:         macdDea,
			MacdHistogram:   macdHistogram,
			BollUpper:       bollUpper,
			BollMiddle:      bollMiddle,
			BollLower:       bollLower,
		})
	}

	// Step 7: 按总分排序
	sort.Slice(allRanked, func(i, j int) bool {
		if allRanked[i].TotalScore != allRanked[j].TotalScore {
			return allRanked[i].TotalScore > allRanked[j].TotalScore
		}
		return allRanked[i].ChangePercent > allRanked[j].ChangePercent
	})

	for i := range allRanked {
		allRanked[i].Rank = i + 1
	}

	// 缓存（仅当结果有效时缓存，阈值降至100防止数据源不稳定时永远无法缓存）
	if len(allRanked) >= 100 {
		s.allStockCacheMu.Lock()
		s.allStockCache = allRanked
		s.allStockCacheTime = time.Now()
		s.allStockCacheMu.Unlock()
	}

	costMs := time.Since(startTime).Milliseconds()

	topList := make([]models.RankStockItem, 0, 50)
	for i := range allRanked {
		if i >= 50 {
			break
		}
		topList = append(topList, allRanked[i])
	}

	return &models.AllStockScanResult{
		TotalStocks:    totalStocks,
		ValidStocks:    validStocks,
		AnalyzedStocks: len(allRanked),
		ScanTime:       time.Now().Format("2006-01-02 15:04:05"),
		CostMs:         costMs,
		TopList:        topList,
	}, nil
}
// GetRankWithPagination 获取排名（分页）
func (s *StockService) GetRankWithPagination(req models.AllStockRankRequest) (*models.AllStockRankResponse, error) {
	startTime := time.Now()

	// 确保有缓存数据
	s.allStockCacheMu.RLock()
	cache := s.allStockCache
	cacheTime := s.allStockCacheTime
	s.allStockCacheMu.RUnlock()

	if cache == nil {
		// 缓存为空，返回空数据（避免同步触发耗时30-60s的全量扫描导致超时）
		return &models.AllStockRankResponse{
			Items:      []models.RankStockItem{},
			Total:      0,
			Page:       req.Page,
			PageSize:   req.PageSize,
			TotalPages: 0,
			ScanTime:   "请先点击「全A股扫描」",
			CostMs:     time.Since(startTime).Milliseconds(),
		}, nil
	}
	if time.Since(cacheTime) > 10*time.Minute {
		log.Printf("全A缓存已过期，返回旧数据，后台将自动刷新")
	}

	// 过滤 - 支持逗号分隔的复合过滤条件
	filters := strings.Split(req.Filter, ",")
	filtered := make([]models.RankStockItem, 0, len(cache))
	for _, item := range cache {
		if req.MinScore > 0 && item.TotalScore < req.MinScore {
			continue
		}
		skip := false
		for _, f := range filters {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			switch f {
			case "golden_cross":
				if !item.IsGoldenCross {
					skip = true
				}
			case "above_water":
				if !item.IsAboveWater {
					skip = true
				}
			case "strong":
				if item.ChangePercent < 3 {
					skip = true
				}
			case "volume_break":
				if item.VolumeRatio < 2 {
					skip = true
				}
			case "surge_limit":
				if item.ChangePercent < 5 {
					skip = true
				}
			default:
				// 复合过滤: change_gt_3, change_lt_3, score_gte_60, industry_xxx
				if strings.HasPrefix(f, "change_gt_") {
					val, err := strconv.ParseFloat(strings.TrimPrefix(f, "change_gt_"), 64)
					if err == nil && item.ChangePercent < val {
						skip = true
					}
				} else if strings.HasPrefix(f, "change_lt_") {
					val, err := strconv.ParseFloat(strings.TrimPrefix(f, "change_lt_"), 64)
					if err == nil && item.ChangePercent > -val {
						skip = true
					}
				} else if strings.HasPrefix(f, "score_gte_") {
					val, err := strconv.ParseFloat(strings.TrimPrefix(f, "score_gte_"), 64)
					if err == nil && item.TotalScore < val {
						skip = true
					}
				} else if strings.HasPrefix(f, "industry_") {
					ind := strings.TrimPrefix(f, "industry_")
					if item.Industry != ind {
						skip = true
					}
				}
			}
			if skip {
				break
			}
		}
		if !skip {
			filtered = append(filtered, item)
		}
	}

	// 排序
	sortField := req.SortBy
	if sortField == "" {
		sortField = "totalScore"
	}
	order := req.Order
	if order == "" {
		order = "desc"
	}

	sort.Slice(filtered, func(i, j int) bool {
		var vi, vj float64
		switch sortField {
		case "totalScore":
			vi, vj = filtered[i].TotalScore, filtered[j].TotalScore
		case "changePercent":
			vi, vj = filtered[i].ChangePercent, filtered[j].ChangePercent
		case "turnoverRate":
			vi, vj = filtered[i].TurnoverRate, filtered[j].TurnoverRate
		case "volumeRatio":
			vi, vj = filtered[i].VolumeRatio, filtered[j].VolumeRatio
		case "techScore":
			vi, vj = filtered[i].TechScore, filtered[j].TechScore
		case "momentumScore":
			vi, vj = filtered[i].MomentumScore, filtered[j].MomentumScore
		case "trendScore":
			vi, vj = filtered[i].TrendScore, filtered[j].TrendScore
		default:
			vi, vj = filtered[i].TotalScore, filtered[j].TotalScore
		}
		if order == "asc" {
			return vi < vj
		}
		return vi > vj
	})

	// 重新赋排名
	for i := range filtered {
		filtered[i].Rank = i + 1
	}

	total := len(filtered)
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	totalPages := (total + pageSize - 1) / pageSize
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	var items []models.RankStockItem
	if start < total {
		items = filtered[start:end]
	} else {
		items = []models.RankStockItem{}
	}

	costMs := time.Since(startTime).Milliseconds()

	return &models.AllStockRankResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Items:      items,
		ScanTime:   cacheTime.Format("2006-01-02 15:04:05"),
		CostMs:     costMs,
	}, nil
}

// fetchBatchQuotesConcurrent 并发批量获取报价
func (s *StockService) fetchBatchQuotesConcurrent(codes []string, batchSize int, concurrency int) []models.SinaQuote {
	var allQuotes []models.SinaQuote
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for i := 0; i < len(codes); i += batchSize {
		end := i + batchSize
		if end > len(codes) {
			end = len(codes)
		}
		batch := codes[i:end]

		wg.Add(1)
		go func(b []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			sinaCodes := make([]string, len(b))
			for j, code := range b {
				sinaCodes[j] = toSinaCode(code)
			}
			quotes, err := s.fetchBatchSinaQuotes(sinaCodes)
			if err != nil {
				return
			}
			mu.Lock()
			allQuotes = append(allQuotes, quotes...)
			mu.Unlock()
		}(batch)
	}
	wg.Wait()
	return allQuotes
}

// getDefaultStockPool returns the default stock pool
func getDefaultStockPool() []string {
	return []string{
		"600519", "600036", "601318", "600276", "601166", "600016", "601328", "600030", "601398", "601288",
		"600887", "600809", "600585", "601888", "600690", "601668", "600050", "600000", "601012", "600104",
		"600028", "601186", "601601", "601628", "601658", "601336", "601390", "601818", "600837", "601211",
		"688599", "688981", "688111", "688126", "688012",
		"300750", "300059", "300014", "300124", "300015", "300122", "300760", "300496", "300033", "300347",
		"000858", "002415", "002594", "000001", "002230", "000333", "002714", "000725", "002475", "002460",
	}
}

// inferIndustryMap 根据股票名称推断行业分类
func (s *StockService) inferIndustryMap(quotes []models.SinaQuote) map[string]string {
	industryMap := make(map[string]string)
	// 行业关键词映射
	industryKeywords := map[string]string{
		"银行": "银行", "证券": "证券", "保险": "保险", "信托": "金融",
		"医药": "医药", "生物": "生物", "药业": "医药", "制药": "医药",
		"电子": "电子", "半导体": "半导体", "芯片": "半导体", "集成": "半导体",
		"软件": "软件", "信息": "信息技术", "科技": "科技", "网络": "互联网",
		"汽车": "汽车", "新能源": "新能源", "锂电": "新能源", "光伏": "新能源",
		"钢铁": "钢铁", "铝": "有色金属", "铜": "有色金属", "锌": "有色金属", "矿": "采掘",
		"地产": "房地产", "建设": "建筑", "建筑": "建筑",
		"电力": "电力", "能源": "能源", "煤炭": "煤炭", "石油": "石油",
		"食品": "食品", "白酒": "白酒", "酒业": "白酒", "酒": "白酒",
		"纺织": "纺织", "服装": "纺织",
		"通信": "通信", "传媒": "传媒", "影视": "传媒", "游戏": "传媒",
		"机械": "机械", "设备": "机械设备", "装备": "机械设备",
		"化工": "化工", "化学": "化工",
		"农业": "农业", "畜牧": "农业", "种业": "农业",
		"航空": "航空", "机场": "交运", "港口": "交运", "物流": "物流",
		"军工": "军工", "航天": "军工", "国防": "军工",
		"环保": "环保", "水务": "环保",
		"商业": "商业", "零售": "商业", "超市": "商业",
		"家电": "家电", "家居": "家电",
	}
	for _, q := range quotes {
		name := q.Name
		if name == "" {
			continue
		}
		matched := false
		for keyword, industry := range industryKeywords {
			if strings.Contains(name, keyword) {
				industryMap[q.Code] = industry
				matched = true
				break
			}
		}
		if !matched {
			// 根据代码前缀分类
			if strings.HasPrefix(q.Code, "688") {
				industryMap[q.Code] = "科创板"
			} else if strings.HasPrefix(q.Code, "300") {
				industryMap[q.Code] = "创业板"
			} else if strings.HasPrefix(q.Code, "8") {
				industryMap[q.Code] = "北交所"
			} else {
				industryMap[q.Code] = "其他"
			}
		}
	}
	return industryMap
}


// CalcSellAdvice 计算卖出建议（三档目标价 + 止损价）
func (s *StockService) CalcSellAdvice(code string, costPrice float64) (*models.SellAdviceResult, error) {
	sinaCode := toSinaCode(code)
	quote, err := s.fetchSinaQuote(sinaCode)
	if err != nil {
		return nil, err
	}

	klines, _ := s.fetchKLineData(code, 60)
	bollData := calcBOLL(klines)
	macdData := calcMACD(klines)

	currentPrice := quote.Price
	profitPercent := 0.0
	if costPrice > 0 {
		profitPercent = (currentPrice - costPrice) / costPrice * 100
	}

	// 止损价：默认 -8%
	stopLossPrice := math.Round(costPrice*0.92*100) / 100

	// 近60日最高价
	hist60High := currentPrice
	for _, k := range klines {
		if k.High > hist60High {
			hist60High = k.High
		}
	}

	// === 三档目标价计算 ===
	var t1Price, t2Price, t3Price float64
	var t1Label, t2Label, t3Label string
	var t1Conf, t2Conf, t3Conf string
	var basis []string

	// Target1（保守）: BOLL上轨 or 成本价+8%
	if bollData != nil && bollData.Upper > currentPrice {
		t1Price = bollData.Upper
		t1Label = "保守止盈"
		t1Conf = "high"
		basis = append(basis, fmt.Sprintf("BOLL上轨阻力 ¥%.2f", bollData.Upper))
	} else {
		t1Price = math.Round(costPrice*1.08*100) / 100
		t1Label = "保守止盈"
		t1Conf = "medium"
		basis = append(basis, "成本价+8%")
	}

	// Target2（标准）: max(成本价+15%, 近60日最高×0.95)
	candidate2a := costPrice * 1.15
	candidate2b := hist60High * 0.95
	if candidate2a > candidate2b {
		t2Price = math.Round(candidate2a*100) / 100
		t2Label = "标准止盈"
		t2Conf = "high"
		basis = append(basis, "成本价+15%目标")
	} else {
		t2Price = math.Round(candidate2b*100) / 100
		t2Label = "标准止盈"
		t2Conf = "medium"
		basis = append(basis, fmt.Sprintf("近期高位压力 ¥%.2f×95%%", hist60High))
	}

	// Target3（积极）: 成本价+25% or BOLL带宽延伸
	if bollData != nil && bollData.Bandwidth > 10 {
		bandwidth := bollData.Upper - bollData.Lower
		t3Price = math.Round((bollData.Upper+bandwidth*0.3)*100) / 100
		t3Label = "积极止盈"
		t3Conf = "low"
		basis = append(basis, "布林带宽延伸趋势")
	} else {
		t3Price = math.Round(costPrice*1.25*100) / 100
		t3Label = "积极止盈"
		t3Conf = "low"
		basis = append(basis, "成本价+25%目标")
	}

	// 确保三档价格单调递增
	if t2Price <= t1Price {
		t2Price = math.Round(t1Price*1.06*100) / 100
	}
	if t3Price <= t2Price {
		t3Price = math.Round(t2Price*1.06*100) / 100
	}

	// MACD辅助判断
	if macdData != nil {
		if macdData.Dif > macdData.Dea && macdData.Dif > 0 {
			basis = append(basis, "MACD水上金叉维持")
		} else if macdData.Dif < macdData.Dea {
			basis = append(basis, "MACD死叉需警惕")
		}
	}

	// === 生成建议文字 ===
	suggestionType := "hold"
	var suggestion string
	switch {
	case profitPercent <= -8:
		suggestionType = "stoploss"
		suggestion = fmt.Sprintf("已触及止损线（亏损%.1f%%），建议尽快止损出局，控制风险。", -profitPercent)
	case profitPercent >= 20:
		suggestionType = "partial"
		suggestion = fmt.Sprintf("已盈利%.1f%%，建议分批减仓锁定利润，可先减50%%仓位至保守目标价¥%.2f附近，剩余仓位继续持有。", profitPercent, t1Price)
	case profitPercent >= 10:
		suggestionType = "partial"
		suggestion = fmt.Sprintf("盈利%.1f%%，BOLL上轨¥%.2f附近是较强阻力，可考虑在此区间减仓30%%。", profitPercent, t1Price)
	case profitPercent >= 0:
		suggestionType = "hold"
		suggestion = fmt.Sprintf("目前盈利%.1f%%，建议继续持有等待目标价¥%.2f。", profitPercent, t2Price)
	default:
		suggestionType = "hold"
		suggestion = fmt.Sprintf("目前小幅亏损%.1f%%，建议持仓观察，若跌破止损价¥%.2f则及时出局。", -profitPercent, stopLossPrice)
	}

	makeTarget := func(price, costP float64, label, conf string) models.SellTarget {
		profit := 0.0
		if costP > 0 {
			profit = (price - costP) / costP * 100
		}
		return models.SellTarget{
			Price:      price,
			Profit:     math.Round(profit*100) / 100,
			Label:      label,
			Confidence: conf,
		}
	}

	return &models.SellAdviceResult{
		Code:           code,
		Name:           quote.Name,
		CurrentPrice:   currentPrice,
		CostPrice:      costPrice,
		ProfitPercent:  math.Round(profitPercent*100) / 100,
		StopLossPrice:  stopLossPrice,
		Target1:        makeTarget(t1Price, costPrice, t1Label, t1Conf),
		Target2:        makeTarget(t2Price, costPrice, t2Label, t2Conf),
		Target3:        makeTarget(t3Price, costPrice, t3Label, t3Conf),
		Suggestion:     suggestion,
		SuggestionType: suggestionType,
		Basis:          basis,
	}, nil
}

// fetchStockIndustryMap 从新浪API获取股票行业信息
func (s *StockService) fetchStockIndustryMap() (map[string]string, error) {
	result := make(map[string]string)

	// 从新浪行业分类API获取
	nodes := []string{"hangye_zz01", "hangye_zz02", "hangye_zz03", "hangye_zz04",
		"hangye_zz05", "hangye_zz06", "hangye_zz07", "hangye_zz08",
		"hangye_zz09", "hangye_zz10", "hangye_zz11", "hangye_zz12"}

	type nodeResult struct {
		industry string
		codes    []string
	}
	results := make(chan nodeResult, len(nodes))

	for _, node := range nodes {
		go func(n string) {
			url := fmt.Sprintf("https://vip.stock.finance.sina.com.cn/quotes_service/api/json_v2.php/Market_Center.getHQNodeData?page=1&num=500&sort=symbol&asc=1&node=%s", n)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				results <- nodeResult{industry: "", codes: nil}
				return
			}
			req.Header.Set("Referer", "https://finance.sina.com.cn")
			req.Header.Set("User-Agent", "Mozilla/5.0")

			resp, err := s.httpClient.Do(req)
			if err != nil {
				results <- nodeResult{industry: "", codes: nil}
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				results <- nodeResult{industry: "", codes: nil}
				return
			}

			decoded := decodeGBK(body)

			var items []struct {
				Symbol    string `json:"symbol"`
				Code      string `json:"code"`
				Name      string `json:"name"`
				Industry  string `json:"industry"`
			}
			if err := json.Unmarshal([]byte(decoded), &items); err != nil {
				results <- nodeResult{industry: "", codes: nil}
				return
			}

			codes := make([]string, 0, len(items))
			indName := ""
			for _, item := range items {
				code := item.Code
				if code == "" {
					code = strings.TrimPrefix(item.Symbol, "sh")
					code = strings.TrimPrefix(code, "sz")
					code = strings.TrimPrefix(code, "bj")
				}
				if code != "" {
					codes = append(codes, code)
					if indName == "" && item.Industry != "" {
						indName = item.Industry
					}
				}
			}
			results <- nodeResult{industry: indName, codes: codes}
		}(node)
	}

	// Collect results
	for i := 0; i < len(nodes); i++ {
		res := <-results
		for _, code := range res.codes {
			if res.industry != "" {
				result[code] = res.industry
			}
		}
	}

	// Fallback: try Sina individual stock industry API for top stocks
	// This provides better industry data
	url2 := "https://vip.stock.finance.sina.com.cn/quotes_service/api/json_v2.php/Market_Center.getHQNodeData?page=1&num=6000&sort=changepercent&asc=0&node=hs_a"
	req2, err := http.NewRequest("GET", url2, nil)
	if err == nil {
		req2.Header.Set("Referer", "https://finance.sina.com.cn")
		req2.Header.Set("User-Agent", "Mozilla/5.0")

		resp2, err := s.httpClient.Do(req2)
		if err == nil {
			defer resp2.Body.Close()
			body2, _ := io.ReadAll(resp2.Body)
			decoded2 := decodeGBK(body2)

			var items2 []struct {
				Code     string `json:"code"`
				Industry string `json:"industry"`
			}
			if json.Unmarshal([]byte(decoded2), &items2) == nil {
				for _, item := range items2 {
					if item.Industry != "" {
						result[item.Code] = item.Industry
					}
				}
			}
		}
	}

	return result, nil
}
