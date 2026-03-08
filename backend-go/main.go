package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RegisterReq struct {
	AgentID  string `json:"agent_id"`
	ShopName string `json:"shop_name"`
}

type ActionReq struct {
	AgentID        string                 `json:"agent_id"`
	ActionType     string                 `json:"action_type"`
	Params         map[string]interface{} `json:"params,omitempty"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
}

type Game struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	ShopName    string    `json:"shop_name"`
	Turn        int       `json:"turn"`
	Gold        int       `json:"gold"`
	Energy      int       `json:"energy"`
	Ingredients int       `json:"ingredients"`
	Meals       int       `json:"meals"`
	Xp          int       `json:"xp"`
	ActionsUsed int       `json:"actions_used"`
	ActionQuota int       `json:"action_quota"`
	LastAction  time.Time `json:"-"`
}

type LogItem struct {
	TS     string `json:"ts"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type Store struct {
	mu      sync.RWMutex
	games   map[string]*Game
	logs    map[string][]LogItem
	idem    map[string]struct{}
	counter int
}

var (
	store = &Store{games: map[string]*Game{}, logs: map[string][]LogItem{}, idem: map[string]struct{}{}, counter: 1000}

	actionQuota = 2
	cooldown    = 1 * time.Second

	reStatus = regexp.MustCompile(`^/api/game/([^/]+)/status$`)
	reAction = regexp.MustCompile(`^/api/game/([^/]+)/action$`)
	reNext   = regexp.MustCompile(`^/api/game/([^/]+)/next-turn$`)
	reHist   = regexp.MustCompile(`^/api/game/([^/]+)/history$`)
)

func main() {
	rand.Seed(time.Now().UnixNano())
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/game/config", handleConfig)
	mux.HandleFunc("/api/game/register", handleRegister)
	mux.HandleFunc("/api/leaderboard", handleLeaderboard)
	mux.HandleFunc("/api/game/", handleGameRoutes)

	log.Println("SnackTruck API on :8080")
	if err := http.ListenAndServe(":8080", cors(mux)); err != nil {
		log.Fatal(err)
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now().Format(time.RFC3339)})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"game":                  "SnackTruck",
		"turn_unit":             "day",
		"max_actions_per_turn":  actionQuota,
		"cooldown_seconds":      int(cooldown.Seconds()),
		"base_meal_price_range": []int{12, 20},
		"actions": []map[string]any{
			{"action_type": "buy_ingredients", "params": "quantity", "effect": "花金币买原料"},
			{"action_type": "cook", "params": "quantity", "effect": "原料->餐品，消耗精力"},
			{"action_type": "sell", "params": "quantity", "effect": "卖餐品赚金币"},
			{"action_type": "rest", "params": "none", "effect": "恢复精力"},
		},
	})
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	var req RegisterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "INVALID_PARAMS", "invalid body")
		return
	}
	if req.AgentID == "" || req.ShopName == "" {
		writeErr(w, http.StatusUnprocessableEntity, "INVALID_PARAMS", "agent_id, shop_name required")
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	id := nextID("game")
	g := &Game{ID: id, AgentID: req.AgentID, ShopName: req.ShopName, Turn: 1, Gold: 120, Energy: 100, Ingredients: 10, Meals: 0, Xp: 0, ActionsUsed: 0, ActionQuota: actionQuota}
	store.games[id] = g
	pushLogLocked(id, LogItem{TS: now(), Type: "system", Title: "开业", Detail: "餐车开业成功"})
	writeJSON(w, http.StatusCreated, map[string]any{"game_id": id, "status": formatStatus(g)})
}

func handleGameRoutes(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if m := reStatus.FindStringSubmatch(p); len(m) == 2 {
		handleStatus(w, r, m[1])
		return
	}
	if m := reAction.FindStringSubmatch(p); len(m) == 2 {
		handleAction(w, r, m[1])
		return
	}
	if m := reNext.FindStringSubmatch(p); len(m) == 2 {
		handleNextTurn(w, r, m[1])
		return
	}
	if m := reHist.FindStringSubmatch(p); len(m) == 2 {
		handleHistory(w, r, m[1])
		return
	}
	writeErr(w, http.StatusNotFound, "NOT_FOUND", "route not found")
}

func handleStatus(w http.ResponseWriter, r *http.Request, id string) {
	store.mu.RLock()
	g := store.games[id]
	store.mu.RUnlock()
	if g == nil {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "game not found")
		return
	}
	writeJSON(w, http.StatusOK, formatStatus(g))
}

func handleAction(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	var req ActionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "INVALID_PARAMS", "invalid body")
		return
	}
	if req.AgentID == "" || req.ActionType == "" {
		writeErr(w, http.StatusUnprocessableEntity, "INVALID_PARAMS", "agent_id, action_type required")
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	g := store.games[id]
	if g == nil {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "game not found")
		return
	}
	if g.AgentID != req.AgentID {
		writeErr(w, http.StatusForbidden, "FORBIDDEN", "agent mismatch")
		return
	}
	if g.ActionsUsed >= g.ActionQuota {
		writeErr(w, http.StatusTooManyRequests, "QUOTA_EXCEEDED", "actions used up for this turn")
		return
	}
	if !g.LastAction.IsZero() && time.Since(g.LastAction) < cooldown {
		writeErr(w, http.StatusTooManyRequests, "COOLDOWN_ACTIVE", "too fast, wait 1 second")
		return
	}
	if req.IdempotencyKey != "" {
		k := g.ID + ":" + req.IdempotencyKey
		if _, ok := store.idem[k]; ok {
			writeErr(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "duplicate request")
			return
		}
		store.idem[k] = struct{}{}
	}

	title, detail, code, msg := applyAction(g, req)
	if code != "" {
		writeErr(w, http.StatusBadRequest, code, msg)
		return
	}
	g.ActionsUsed++
	g.LastAction = time.Now()
	pushLogLocked(g.ID, LogItem{TS: now(), Type: "action", Title: title, Detail: detail})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": map[string]string{"title": title, "detail": detail}, "status": formatStatus(g)})
}

func handleNextTurn(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	g := store.games[id]
	if g == nil {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "game not found")
		return
	}
	g.Turn++
	g.ActionsUsed = 0
	g.ActionQuota = actionQuota
	g.Energy = clamp(g.Energy+35, 0, 100)
	if rand.Float64() < 0.25 {
		bonus := rand.Intn(18) + 8
		g.Gold += bonus
		pushLogLocked(g.ID, LogItem{TS: now(), Type: "event", Title: "路人打赏", Detail: fmt.Sprintf("获得 %d 金币", bonus)})
	}
	pushLogLocked(g.ID, LogItem{TS: now(), Type: "system", Title: "进入下一天", Detail: fmt.Sprintf("Day %d", g.Turn)})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": formatStatus(g)})
}

func handleHistory(w http.ResponseWriter, r *http.Request, id string) {
	store.mu.RLock()
	_, ok := store.games[id]
	logs := store.logs[id]
	store.mu.RUnlock()
	if !ok {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "game not found")
		return
	}
	limit := 30
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = clamp(v, 1, 100)
		}
	}
	if len(logs) > limit {
		logs = logs[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": logs})
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	type item struct {
		GameID   string `json:"game_id"`
		ShopName string `json:"shop_name"`
		Turn     int    `json:"turn"`
		Gold     int    `json:"gold"`
		Meals    int    `json:"meals"`
		Xp       int    `json:"xp"`
		Score    int    `json:"score"`
	}
	items := []item{}
	store.mu.RLock()
	for _, g := range store.games {
		score := g.Gold + g.Xp*3 + g.Meals*2 + g.Turn
		items = append(items, item{GameID: g.ID, ShopName: g.ShopName, Turn: g.Turn, Gold: g.Gold, Meals: g.Meals, Xp: g.Xp, Score: score})
	}
	store.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	if len(items) > 50 {
		items = items[:50]
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": len(items), "items": items})
}

func applyAction(g *Game, req ActionReq) (string, string, string, string) {
	q := intParam(req.Params, "quantity", 1)
	if q < 1 {
		q = 1
	}
	switch strings.ToLower(req.ActionType) {
	case "buy_ingredients":
		cost := q * 5
		if g.Gold < cost {
			return "", "", "INSUFFICIENT_GOLD", "not enough gold"
		}
		g.Gold -= cost
		g.Ingredients += q
		return "进货", fmt.Sprintf("买入原料 %d 份，花费 %d", q, cost), "", ""
	case "cook":
		if g.Ingredients < q {
			return "", "", "INSUFFICIENT_INGREDIENTS", "not enough ingredients"
		}
		needEnergy := q * 4
		if g.Energy < needEnergy {
			return "", "", "INSUFFICIENT_ENERGY", "not enough energy"
		}
		g.Ingredients -= q
		g.Energy -= needEnergy
		g.Meals += q
		g.Xp += q
		return "烹饪", fmt.Sprintf("做出餐品 %d 份", q), "", ""
	case "sell":
		if g.Meals < q {
			return "", "", "INSUFFICIENT_MEALS", "not enough meals"
		}
		unit := rand.Intn(9) + 12
		rev := unit * q
		g.Meals -= q
		g.Gold += rev
		g.Xp += q * 2
		return "营业", fmt.Sprintf("售出 %d 份，收入 %d", q, rev), "", ""
	case "rest":
		g.Energy = clamp(g.Energy+30, 0, 100)
		return "休息", "恢复体力 +30", "", ""
	default:
		return "", "", "INVALID_ACTION", "unsupported action_type"
	}
}

func formatStatus(g *Game) map[string]any {
	suggested := []string{"buy_ingredients", "cook", "sell"}
	if g.Ingredients == 0 {
		suggested = []string{"buy_ingredients"}
	} else if g.Meals == 0 {
		suggested = []string{"cook"}
	} else if g.Energy < 20 {
		suggested = []string{"rest"}
	}
	return map[string]any{
		"game": map[string]any{
			"game_id":   g.ID,
			"shop_name": g.ShopName,
			"turn":      g.Turn,
		},
		"resources": map[string]any{
			"gold":             g.Gold,
			"energy":           g.Energy,
			"ingredients":      g.Ingredients,
			"meals":            g.Meals,
			"xp":               g.Xp,
			"actions_used":     g.ActionsUsed,
			"action_quota":     g.ActionQuota,
			"cooldown_seconds": int(cooldown.Seconds()),
		},
		"suggested_actions": suggested,
		"agent_skill_url":   "/skill",
	}
}

func pushLogLocked(id string, li LogItem) {
	old := store.logs[id]
	old = append([]LogItem{li}, old...)
	if len(old) > 200 {
		old = old[:200]
	}
	store.logs[id] = old
}

func nextID(prefix string) string {
	store.counter++
	return fmt.Sprintf("%s_%d", prefix, store.counter)
}

func intParam(params map[string]interface{}, key string, def int) int {
	if params == nil {
		return def
	}
	v, ok := params[key]
	if !ok {
		return def
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return def
	}
}

func clamp(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func now() string { return time.Now().Format(time.RFC3339) }

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErr(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
