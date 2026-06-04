# DESIGN.md

## 品牌与视觉方向
- 产品名：波段趋势共振助手
- 风格：金融数据可视化，专业简洁

## Design Tokens

### 色彩
- 主色：蓝色系 (blue-500 ~ blue-700)
- 中国股市配色约定：红涨绿跌
  - 上涨/正值：text-red-500 / text-red-600
  - 下跌/负值：text-green-500 / text-green-600
- 核心标的标识：yellow-500 左侧边框
- 卫星标的标识：slate-400 左侧边框
- 推荐等级配色：
  - 强烈推荐(≥75)：text-red-700 bg-red-50
  - 积极关注(≥60)：text-amber-700 bg-amber-50
  - 一般关注(≥45)：text-blue-700 bg-blue-50
  - 观望(≥30)：text-slate-600 bg-slate-100
  - 谨慎观望(≥15)：text-slate-600 bg-slate-100
  - 回避(<15)：text-gray-500 bg-gray-100

### 布局
- 顶部固定导航栏 + 移动端底部Tab栏
- 内容区域 max-w-7xl 居中
- 卡片圆角 rounded-xl，阴影 shadow-sm
