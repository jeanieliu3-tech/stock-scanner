package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"stock-app/models"
	"stock-app/services"
)

type StockHandler struct {
	service *services.StockService
}

func NewStockHandler(s *services.StockService) *StockHandler {
	return &StockHandler{service: s}
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": data})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": code, "msg": msg, "data": nil})
}

func (h *StockHandler) GetMarketStatus(c *gin.Context) {
	data, err := h.service.GetMarketIndices()
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, data)
}

func (h *StockHandler) GetQuotes(c *gin.Context) {
	codesStr := c.Query("codes")
	if codesStr == "" {
		ok(c, map[string]interface{}{})
		return
	}
	codes := strings.Split(codesStr, ",")
	data, err := h.service.GetBatchQuotes(codes)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, data)
}

func (h *StockHandler) DiagnoseStock(c *gin.Context) {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Code == "" {
		fail(c, 400, "请提供股票代码")
		return
	}
	data, err := h.service.DiagnoseStock(body.Code)
	if err != nil {
		fail(c, 404, body.Code+" 不是有效的A股代码或无法获取数据")
		return
	}
	ok(c, data)
}

func (h *StockHandler) ScanStocks(c *gin.Context) {
	data, err := h.service.ScanStocks()
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, data)
}

func (h *StockHandler) ScanCoreSatellite(c *gin.Context) {
	data, err := h.service.ScanCoreSatellite()
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, data)
}

func (h *StockHandler) ScanDualEngineFast(c *gin.Context) {
	data, err := h.service.ScanDualEngineFast()
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, data)
}

func (h *StockHandler) GetWatchList(c *gin.Context) {
	data := h.service.GetWatchList()
	ok(c, data)
}

func (h *StockHandler) AddWatchList(c *gin.Context) {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Code == "" {
		fail(c, 400, "请提供股票代码")
		return
	}
	success, msg := h.service.AddWatchList(body.Code)
	code := 200
	if !success {
		code = 400
	}
	c.JSON(http.StatusOK, gin.H{"code": code, "msg": msg, "data": nil})
}

func (h *StockHandler) RemoveWatchList(c *gin.Context) {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Code == "" {
		fail(c, 400, "请提供股票代码")
		return
	}
	_, msg := h.service.RemoveWatchList(body.Code)
	ok(c, nil)
	_ = msg
}

func (h *StockHandler) ScanWatchList(c *gin.Context) {
	data, err := h.service.ScanWatchList()
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, data)
}

func (h *StockHandler) GetStockDetail(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		fail(c, 400, "请提供股票代码")
		return
	}
	data, err := h.service.GetStockDetail(code)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, data)
}

func (h *StockHandler) GetStockScores(c *gin.Context) {
	codesStr := c.Query("codes")
	if codesStr == "" {
		ok(c, map[string]interface{}{})
		return
	}
	codes := strings.Split(codesStr, ",")
	data := h.service.GetStockScores(codes)
	ok(c, data)
}

func (h *StockHandler) GetUnifiedScore(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		fail(c, 400, "请提供股票代码")
		return
	}
	data := h.service.GetUnifiedScore(code)
	if data == nil {
		fail(c, 404, "未找到该股票评分，请先执行全A扫描")
		return
	}
	ok(c, data)
}

// GetPositionHealthScore 获取持仓健康度（市场评分 + 持仓个性化调整）
func (h *StockHandler) GetPositionHealthScore(c *gin.Context) {
	code := c.Query("code")
	costPriceStr := c.Query("costPrice")
	if code == "" {
		fail(c, 400, "请提供股票代码")
		return
	}
	costPrice, _ := strconv.ParseFloat(costPriceStr, 64)
	data := h.service.GetPositionHealthScore(code, costPrice)
	if data == nil {
		fail(c, 404, "未找到该股票评分，请先执行全A扫描")
		return
	}
	ok(c, data)
}

func (h *StockHandler) SearchStocks(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		ok(c, []interface{}{})
		return
	}
	data := h.service.SearchStocks(keyword)
	if data == nil {
		data = []models.SearchItem{}
	}
	result := make([]map[string]string, len(data))
	for i, item := range data {
		result[i] = map[string]string{"code": item.Code, "name": item.Name}
	}
	ok(c, result)
	_ = strconv.Itoa(0)
}

func (h *StockHandler) ScanAllAShares(c *gin.Context) {
	data, err := h.service.ScanAllAShares()
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, data)
}

func (h *StockHandler) GetAllStockRank(c *gin.Context) {
	var req models.AllStockRankRequest
	req.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	req.PageSize, _ = strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	req.SortBy = c.DefaultQuery("sortBy", "totalScore")
	req.Order = c.DefaultQuery("order", "desc")
	req.MinScore, _ = strconv.ParseFloat(c.DefaultQuery("minScore", "0"), 64)
	req.Filter = c.DefaultQuery("filter", "")

	data, err := h.service.GetRankWithPagination(req)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, data)
}

func (h *StockHandler) GetSellAdvice(c *gin.Context) {
	var body struct {
		Code      string  `json:"code"`
		CostPrice float64 `json:"costPrice"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Code == "" {
		fail(c, 400, "请提供股票代码和成本价")
		return
	}
	data, err := h.service.CalcSellAdvice(body.Code, body.CostPrice)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, data)
}
