package models

// IndexData 指数数据
type IndexData struct {
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
}

// MarketData 大盘数据
type MarketData struct {
	Time    string               `json:"time"`
	Status  string               `json:"status"`
	Indices map[string]*IndexData `json:"indices"`
}

// DualEngineScore 双引擎评分
type DualEngineScore struct {
	SectorHeatScore      int      `json:"sectorHeatScore"`
	SectorPositionScore  int      `json:"sectorPositionScore"`
	CapitalStrengthScore int      `json:"capitalStrengthScore"`
	MarketBaseScore      int      `json:"marketBaseScore"`
	BollTrendScore       int      `json:"bollTrendScore"`
	MacdSignalScore      int      `json:"macdSignalScore"`
	SignalConfirmScore   int      `json:"signalConfirmScore"`
	TechBonusScore       int      `json:"techBonusScore"`
	TotalScore           float64  `json:"totalScore"`
	Highlights           []string `json:"highlights"`
}

// DualEngineStock 双引擎标的
type DualEngineStock struct {
	Code                string          `json:"code"`
	Name                string          `json:"name"`
	Price               float64         `json:"price"`
	ChangePercent       float64         `json:"changePercent"`
	Sector              string          `json:"sector"`
	VolumeRatio         float64         `json:"volumeRatio"`
	TurnoverRate        float64         `json:"turnoverRate"`
	BandwidthRatio      float64         `json:"bandwidthRatio"`
	Dif                 float64         `json:"dif"`
	Macd                float64         `json:"macd"`
	MacdSignal          string          `json:"macdSignal"`
	BollPosition        string          `json:"bollPosition"`
	IsGoldenCross       bool            `json:"isGoldenCross"`
	IsAboveWater        bool            `json:"isAboveWater"`
	Recommendation      string          `json:"recommendation"`
	Score               DualEngineScore `json:"score"`
	RecommendedPosition float64         `json:"recommendedPosition"`
}

// ScanResult 扫描结果
type ScanResult struct {
	TotalScanned       int              `json:"totalScanned"`
	ValidQuotes        int              `json:"validQuotes"`
	DeepAnalyzedCount  int              `json:"deepAnalyzedCount"`
	Core               []DualEngineStock `json:"core"`
	Satellite          []DualEngineStock `json:"satellite"`
	CoreTotalWeight    float64          `json:"coreTotalWeight"`
	SatelliteTotalWeight float64        `json:"satelliteTotalWeight"`
	CashReserve        float64          `json:"cashReserve"`
	ScanTime           string           `json:"scanTime,omitempty"`
}

// KLineData K线数据
type KLineData struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// MacdData MACD数据
type MacdData struct {
	Dif          float64 `json:"dif"`
	Dea          float64 `json:"dea"`
	Macd         float64 `json:"macd"`
	Status       string  `json:"status"`
	Signal       string  `json:"signal"`
	AxisPosition string  `json:"axisPosition"`
}

// BollData BOLL数据
type BollData struct {
	Upper     float64 `json:"upper"`
	Middle    float64 `json:"middle"`
	Lower     float64 `json:"lower"`
	Position  string  `json:"position"`
	Bandwidth float64 `json:"bandwidth"`
}

// StockTechnicalDetail 股票技术详情
type StockTechnicalDetail struct {
	Code          string      `json:"code"`
	Name          string      `json:"name"`
	Price         float64     `json:"price"`
	ChangePercent float64     `json:"changePercent"`
	Macd          MacdData    `json:"macd"`
	Boll          BollData    `json:"boll"`
	KlineHistory  []KLineData `json:"klineHistory"`
}

// DiagnoseResult 诊断结果
type DiagnoseResult struct {
	Name           string  `json:"name"`
	Code           string  `json:"code"`
	Price          float64 `json:"price"`
	ChangePercent  float64 `json:"changePercent"`
	Recommendation string  `json:"recommendation"`
	Analysis       string  `json:"analysis"`
	Score          float64 `json:"score,omitempty"`
}

// StockScore 股票评分（统一接口，数据来源于全A扫描缓存，确保全平台一致）
type StockScore struct {
	TotalScore      float64  `json:"totalScore"`
	MarketBaseScore float64  `json:"marketBaseScore"`
	TechBonusScore  float64  `json:"techBonusScore"`
	TrendScore      float64  `json:"trendScore"`
	MomentumScore   float64  `json:"momentumScore"`
	VolumeScore     float64  `json:"volumeScore"`
	TechScore       float64  `json:"techScore"`
	MacdSignal      string   `json:"macdSignal"`
	BollPosition    string   `json:"bollPosition"`
	IsGoldenCross   bool     `json:"isGoldenCross"`
	IsAboveWater    bool     `json:"isAboveWater"`
	Highlights      []string `json:"highlights"`
	Recommendation  string   `json:"recommendation"`
	ResonanceStar   float64  `json:"resonanceStar"`
	// 持仓个性化字段（仅当从持仓页调用时有意义）
	PositionHealthScore float64 `json:"positionHealthScore,omitempty"`
	PositionHealthLabel string  `json:"positionHealthLabel,omitempty"`
}

// SinaQuote 新浪行情数据
type SinaQuote struct {
	Code         string
	Name         string
	Open         float64
	PrevClose    float64
	Price        float64
	High         float64
	Low          float64
	Volume       float64
	Amount       float64
	Buy1Vol      float64
	Buy1Price    float64
	Date         string
	Time         string
	Change       float64
	ChangePercent float64
	TurnoverRate float64
	Industry     string
}

// WatchScanStock 关注列表扫描结果
type WatchScanStock struct {
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	Price               float64 `json:"price"`
	ChangePercent       float64 `json:"changePercent"`
	Score               float64 `json:"score"`
	Sector              string  `json:"sector"`
	FundHeat            string  `json:"fundHeat"`
	RecommendedPosition float64 `json:"recommendedPosition"`
	MacdStatus          string  `json:"macdStatus"`
	BollStatus          string  `json:"bollStatus"`
}

// WatchScanResult 关注列表扫描结果
type WatchScanResult struct {
	Signals []WatchScanStock `json:"signals"`
}

// SearchItem 搜索结果
type SearchItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// RankStockItem 全A排名股票项
type RankStockItem struct {
	Rank            int      `json:"rank"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Price           float64  `json:"price"`
	ChangePercent   float64  `json:"changePercent"`
	TurnoverRate    float64  `json:"turnoverRate"`
	VolumeRatio     float64  `json:"volumeRatio"`
	TotalScore      float64  `json:"totalScore"`
	TrendScore      float64  `json:"trendScore"`
	MomentumScore   float64  `json:"momentumScore"`
	VolumeScore     float64  `json:"volumeScore"`
	TechScore       float64  `json:"techScore"`
	MacdSignal      string   `json:"macdSignal"`
	BollPosition    string   `json:"bollPosition"`
	IsGoldenCross   bool     `json:"isGoldenCross"`
	IsAboveWater      bool     `json:"isAboveWater"`
	Highlights        []string `json:"highlights"`
	Recommendation    string   `json:"recommendation"`
	Industry          string   `json:"industry"`
	ResonanceStar     float64  `json:"resonanceStar"` // 共振强度星级(0-5)
	MacdDif           float64  `json:"macdDif"`
	MacdDea           float64  `json:"macdDea"`
	MacdHistogram     float64  `json:"macdHistogram"`
	BollUpper         float64  `json:"bollUpper"`
	BollMiddle        float64  `json:"bollMiddle"`
	BollLower         float64  `json:"bollLower"`
}

// AllStockScanResult 全A扫描结果
type AllStockScanResult struct {
	TotalStocks    int             `json:"totalStocks"`
	ValidStocks    int             `json:"validStocks"`
	AnalyzedStocks int             `json:"analyzedStocks"`
	ScanTime       string          `json:"scanTime"`
	CostMs         int64           `json:"costMs"`
	TopList        []RankStockItem `json:"topList"`
}

// AllStockRankRequest 全A排名请求
type AllStockRankRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	SortBy   string `json:"sortBy"`
	Order    string `json:"order"`
	MinScore float64 `json:"minScore"`
	Filter   string `json:"filter"`
}

// AllStockRankResponse 全A排名响应
type AllStockRankResponse struct {
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
	Items      []RankStockItem `json:"items"`
	ScanTime   string          `json:"scanTime"`
	CostMs     int64           `json:"costMs"`
}

// SellTarget 卖出目标
type SellTarget struct {
	Price      float64 `json:"price"`
	Profit     float64 `json:"profit"`
	Label      string  `json:"label"`
	Confidence string  `json:"confidence"` // high/medium/low
}

// SellAdviceResult 卖出建议结果
type SellAdviceResult struct {
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	CurrentPrice   float64    `json:"currentPrice"`
	CostPrice      float64    `json:"costPrice"`
	ProfitPercent  float64    `json:"profitPercent"`
	StopLossPrice  float64    `json:"stopLossPrice"`
	Target1        SellTarget `json:"target1"`
	Target2        SellTarget `json:"target2"`
	Target3        SellTarget `json:"target3"`
	Suggestion     string     `json:"suggestion"`
	SuggestionType string     `json:"suggestionType"` // hold/partial/full/stoploss
	Basis          []string   `json:"basis"`
}
