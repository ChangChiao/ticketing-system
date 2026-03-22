## Context

這是一個全新的演唱會售票系統，目標對標拓元、KKTIX 等平台。核心挑戰在於開賣瞬間的萬人併發搶票場景——需要在高壓下保證不超賣、公平排隊、且付款流程順暢。

場館以台北大巨蛋為參考模型，使用者選擇「區域 + 張數」，系統自動分配連續座位。前端使用 Canvas 渲染場館圖，後端以 Go 處理高併發請求。

系統僅面向購票者，不包含後台管理介面（後台為未來擴展）。

## Goals / Non-Goals

**Goals:**
- 支援萬人同時排隊搶票，系統不崩潰且不超賣
- 提供公平的排隊機制，按先到先得順序進入選位
- Canvas 場館圖即時顯示各區剩餘票數
- 自動分配同排連續座位，提供良好觀賞體驗
- 整合 LINE Pay 完成付款，10 分鐘內未付款自動釋放座位
- 防止機器人與作弊行為

**Non-Goals:**
- 後台管理介面（活動建立、場館管理、銷售報表）— 第一版用 DB seed/migration 建立資料
- 多付款方式（信用卡、超商付款）— 僅支援 LINE Pay
- 退票/換票功能
- 多語系支援
- 手機 App（僅 Web 響應式設計）

## Decisions

### D1: 後端語言 — Go (Gin)

**選擇**: Go + Gin framework

**替代方案**:
- Node.js (Express/Fastify): 生態成熟但單線程模型在萬人併發場景下 CPU/memory 效率不如 Go
- Java (Spring Boot): 效能好但啟動慢、記憶體用量高，不利於快速擴縮容
- Rust (Actix): 效能最佳但學習曲線陡峭，開發速度慢

**理由**: Go 的 goroutine 模型天生適合高併發場景，記憶體用量低有利於 K8s Pod 快速擴展，編譯為單一 binary 部署簡單。Gin 是最成熟的 Go Web framework。

### D2: 前端框架 — Next.js + Canvas

**選擇**: Next.js 14 (App Router) + HTML5 Canvas + Zustand

**替代方案**:
- Nuxt.js (Vue): 可行但 React 生態的 Canvas 相關 library 更豐富
- SvelteKit: 效能好但社群資源較少
- SVG 座位圖: DOM 節點過多，大場館效能差

**理由**: Next.js SSR 確保首屏載入速度（SEO 不是重點但 SSR 改善 LCP）。Canvas 渲染場館區域圖效能好，即使場館有數十個區域也不會有 DOM 瓶頸。Zustand 輕量且適合管理 WebSocket 連線狀態。

### D3: 排隊機制 — Redis Sorted Set + WebSocket

**選擇**: 使用者進入等候室時 ZADD 到 Redis Sorted Set（score = timestamp），排隊控制器每隔數秒 ZPOPMIN 取出一批使用者，透過 WebSocket 通知「輪到你了」。

**替代方案**:
- Redis List (LPUSH/RPOP): 簡單但無法查詢使用者排隊位置
- 資料庫 Queue: 太慢，無法承受萬人併發寫入
- Polling 取代 WebSocket: 萬人每秒 polling 會壓垮 API

**理由**: Sorted Set 允許 O(log N) 查詢使用者位置（ZRANK），方便顯示「前方還有 N 人」。WebSocket 長連線比 polling 節省大量請求。

**控制參數**:
- `max_concurrent`: 選位頁最大同時人數 500
- `batch_size`: 每批放入 50 人
- `batch_interval`: 每 5 秒放一批
- `session_ttl`: 選位 session 10 分鐘

### D4: 座位鎖定 — Redis Lua Script 原子操作

**選擇**: 座位分配與鎖定使用 Redis Lua Script，一次檢查+鎖定多個座位，全成功或全失敗。

**替代方案**:
- PostgreSQL SELECT FOR UPDATE: 可行但在高併發下 lock contention 嚴重
- Redis SETNX 逐一鎖定: 非原子性，可能部分成功部分失敗
- Distributed Lock (Redlock): 過度複雜，不需要跨節點鎖

**理由**: Lua Script 在 Redis 中是原子執行的，不會有 race condition。配合 TTL 10 分鐘自動釋放，不需要額外的清理機制。

### D5: 座位分配演算法 — 中間排優先 + Sliding Window

**選擇**: 從區域的中間排開始往前後擴散搜尋，每排內用 sliding window 找連續空位。

**理由**: 中間排觀賞體驗最佳，優先填滿可提升購票者滿意度。Sliding window 時間複雜度 O(n)，每排座位數有限（通常 < 50），效能不是問題。

### D6: 資料一致性 — Redis 快取 + PostgreSQL 持久化

**選擇**: 即時座位狀態存放 Redis（單一真相來源），付款成功後同步寫入 PostgreSQL。EventSeat 在活動建立時預先 INSERT。

**替代方案**:
- 僅用 PostgreSQL: 高併發下讀寫效能不足
- 僅用 Redis: 資料持久性不夠，Redis 重啟會遺失資料

**理由**: 搶票期間 Redis 是座位狀態的 source of truth，低延遲處理併發。PostgreSQL 作為持久化層，確保付款完成的訂單不會遺失。Redis 資料可從 DB 重建。

### D7: 付款 — LINE Pay V3 API

**選擇**: LINE Pay V3 Request/Confirm 流程。

**流程**: 建立訂單 → 呼叫 LINE Pay Request API 取得付款 URL → 使用者跳轉 LINE 確認 → LINE 回呼 confirmUrl → 後端呼叫 Confirm API → 訂單成立。

**異常處理**:
- 使用者取消: redirect 到 cancelUrl，釋放座位
- 10 分鐘逾時: 排程檢查 expired locks，釋放座位
- Confirm 失敗: 重試 3 次，仍失敗則標記為需人工處理

## Risks / Trade-offs

**[Redis 單點故障]** → 使用 Redis Cluster（至少 3 master + 3 replica）。搶票期間若 Redis 掛掉，系統暫停售票並顯示維護頁面，待恢復後從 PostgreSQL 重建 Redis 狀態。

**[座位分配不公平]** → 排隊制本身保證先到先得的公平性。但同一批次內的 50 人可能同時選同一區域，靠 Lua Script 原子鎖定解決衝突，失敗者立即重新分配。

**[LINE Pay API 延遲或故障]** → 座位鎖定 TTL 設為 10 分鐘已包含 LINE Pay 處理時間。若 LINE Pay 長時間故障，座位會因 TTL 過期自動釋放，不會永久卡住。

**[Canvas 場館圖載入效能]** → 場館底圖資料（區域 polygon）透過 CDN 快取，僅即時票數透過 WebSocket 更新。區域數量有限（通常 < 20），Canvas 渲染壓力小。

**[EventSeat 預建資料量]** → 大場館一場活動約 40,000 筆 EventSeat。PostgreSQL 處理這個量級的 INSERT 和索引沒有問題。活動建立時一次性 batch insert。

**[WebSocket 連線數]** → 萬人同時連線需要足夠的 WebSocket Pod。每個 Go Pod 可承受約 10,000 連線，2-3 個 Pod 即可。透過 Redis Pub/Sub 跨 Pod 廣播訊息。
