## Why

演唱會售票是高度時間敏感且併發密集的場景。現有的售票平台（拓元、KKTIX）經常在開賣瞬間出現系統崩潰、超賣、或使用者體驗極差的問題。我們需要一個從架構層面就為萬人搶票設計的售票系統，透過排隊機制控制流量、原子性座位鎖定防止超賣、並整合 LINE Pay 提供流暢的購票體驗。

## What Changes

這是一個全新專案，從零建立以下能力：

- **活動瀏覽系統**：活動列表與詳情頁面，展示場次、票價、場館資訊
- **虛擬排隊機制**：開賣時使用者進入等候室，系統分批放入選位頁，控制同時在線人數
- **Canvas 場館圖**：以 Canvas 渲染場館區域圖（台北大巨蛋模式），顯示各區即時剩餘票數
- **區域選位 + 自動座位分配**：使用者選「區域 + 張數」，系統自動分配同排連續座位
- **10 分鐘付款限時**：座位鎖定 10 分鐘 TTL，逾時自動釋放
- **LINE Pay 金流整合**：Reserve → Confirm 兩步驟付款流程
- **訂單管理**：購票成功頁、我的訂單、電子票券
- **防作弊機制**：CAPTCHA、單帳號單 session、Rate Limiting

## Capabilities

### New Capabilities

- `event-browsing`: 活動列表與詳情頁面，包含場次、票價、場館資訊的展示
- `queue-system`: 虛擬排隊機制，Redis Sorted Set 排序，WebSocket 即時推送，分批放入選位頁
- `venue-map`: Canvas 場館區域圖渲染，多邊形區域繪製，即時剩餘票數色彩標示
- `seat-allocation`: 區域選位與自動座位分配演算法，Redis Lua Script 原子性鎖定，10 分鐘 TTL
- `line-pay-integration`: LINE Pay V3 API 整合，Reserve/Confirm 流程，異常處理
- `order-management`: 訂單建立、查詢、電子票券，付款狀態追蹤
- `anti-fraud`: CAPTCHA 驗證、單帳號限制、裝置指紋、API Rate Limiting

### Modified Capabilities

（無，全新專案）

## Impact

- **新增前端應用**：Next.js (React) SPA/SSR，含 Canvas 場館圖元件、WebSocket 連線管理、Zustand 狀態管理
- **新增後端服務**：Go (Gin/Echo) API 伺服器、WebSocket 伺服器、排隊控制器 Worker
- **新增資料庫**：PostgreSQL（場館/活動/座位/訂單/付款）
- **新增快取層**：Redis Cluster（排隊佇列、座位鎖定、即時庫存）+ Redis Pub/Sub（WebSocket 廣播）
- **外部依賴**：LINE Pay V3 API、CAPTCHA 服務（hCaptcha/Turnstile）
- **部署基礎設施**：Docker + Kubernetes、Cloudflare CDN、HPA 自動擴展
