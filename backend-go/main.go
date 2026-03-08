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

type RegisterRequest struct {
	AgentID    string `json:"agent_id"`
	FamilyName string `json:"family_name"`
	ChildName  string `json:"child_name"`
}

type ActionRequest struct {
	AgentID        string                 `json:"agent_id"`
	ActionType     string                 `json:"action_type"`
	Params         map[string]interface{} `json:"params,omitempty"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
}

type Child struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Generation int    `json:"generation"`
	AgeMonth   int    `json:"age_month"`

	Intelligence int `json:"intelligence"`
	Discipline   int `json:"discipline"`
	Health       int `json:"health"`
	Stress       int `json:"stress"`
	SelfEsteem   int `json:"self_esteem"`
	Rebellion    int `json:"rebellion"`
	StudyScore   int `json:"study_score"`
}

type ParentSnapshot struct {
	Intelligence int `json:"intelligence"`
	Discipline   int `json:"discipline"`
	Health       int `json:"health"`
	Stress       int `json:"stress"`
	SelfEsteem   int `json:"self_esteem"`
	Rebellion    int `json:"rebellion"`
}

type Family struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	AgentID      string `json:"agent_id"`
	Generation   int    `json:"generation"`
	Turn         int    `json:"turn"`
	FamilyMoney  int    `json:"family_money"`
	ParentEnergy int    `json:"parent_energy"`
	ChildEnergy  int    `json:"child_energy"`
	ActionQuota  int    `json:"action_quota"`
	ActionsUsed  int    `json:"actions_used"`

	Child          Child           `json:"child"`
	ParentSnapshot *ParentSnapshot `json:"parent_snapshot,omitempty"`
}

type LogEntry struct {
	TS     string `json:"ts"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type Store struct {
	mu          sync.RWMutex
	families    map[string]*Family
	logs        map[string][]LogEntry
	idempotency map[string]struct{}
}

func newStore() *Store {
	return &Store{
		families:    map[string]*Family{},
		logs:        map[string][]LogEntry{},
		idempotency: map[string]struct{}{},
	}
}

var (
	store         = newStore()
	actionLimit   = 4
	adultAgeMonth = 22 * 12
	idCounter     = 1000

	reStatus   = regexp.MustCompile(`^/api/family/([^/]+)/status$`)
	reAction   = regexp.MustCompile(`^/api/family/([^/]+)/action$`)
	reNextTurn = regexp.MustCompile(`^/api/family/([^/]+)/next-turn$`)
	reNextGen  = regexp.MustCompile(`^/api/family/([^/]+)/next-generation$`)
	reHistory  = regexp.MustCompile(`^/api/family/([^/]+)/history$`)
)

func main() {
	rand.Seed(time.Now().UnixNano())

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/family/register", handleRegister)
	mux.HandleFunc("/api/leaderboard", handleLeaderboard)
	mux.HandleFunc("/api/family/", handleFamilyRoutes)

	handler := corsMiddleware(mux)
	addr := ":8080"
	log.Printf("Go backend listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
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
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "time": time.Now().Format(time.RFC3339)})
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_PARAMS", "invalid json body")
		return
	}
	if req.AgentID == "" || req.FamilyName == "" || req.ChildName == "" {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_PARAMS", "agent_id, family_name, child_name 必填")
		return
	}

	familyID := nextID("fam")
	childID := nextID("child")
	f := &Family{
		ID:           familyID,
		Name:         req.FamilyName,
		AgentID:      req.AgentID,
		Generation:   1,
		Turn:         1,
		FamilyMoney:  12000,
		ParentEnergy: 100,
		ChildEnergy:  100,
		ActionQuota:  actionLimit,
		ActionsUsed:  0,
		Child: Child{
			ID:           childID,
			Name:         req.ChildName,
			Generation:   1,
			AgeMonth:     72,
			Intelligence: randInt(40, 65),
			Discipline:   randInt(35, 60),
			Health:       randInt(45, 70),
			Stress:       randInt(20, 40),
			SelfEsteem:   randInt(35, 65),
			Rebellion:    randInt(20, 50),
			StudyScore:   randInt(20, 45),
		},
	}

	store.mu.Lock()
	store.families[f.ID] = f
	pushLogLocked(f.ID, LogEntry{TS: now(), Type: "system", Title: "家族创建成功", Detail: "欢迎来到家脉，第1代已创建"})
	store.mu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]interface{}{"family_id": f.ID, "status": formatStatus(f)})
}

func handleFamilyRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if m := reStatus.FindStringSubmatch(path); len(m) == 2 {
		handleStatus(w, r, m[1])
		return
	}
	if m := reAction.FindStringSubmatch(path); len(m) == 2 {
		handleAction(w, r, m[1])
		return
	}
	if m := reNextTurn.FindStringSubmatch(path); len(m) == 2 {
		handleNextTurn(w, r, m[1])
		return
	}
	if m := reNextGen.FindStringSubmatch(path); len(m) == 2 {
		handleNextGeneration(w, r, m[1])
		return
	}
	if m := reHistory.FindStringSubmatch(path); len(m) == 2 {
		handleHistory(w, r, m[1])
		return
	}
	writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
}

func handleStatus(w http.ResponseWriter, r *http.Request, familyID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	store.mu.RLock()
	f := store.families[familyID]
	store.mu.RUnlock()
	if f == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "family 不存在")
		return
	}
	writeJSON(w, http.StatusOK, formatStatus(f))
}

func handleAction(w http.ResponseWriter, r *http.Request, familyID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_PARAMS", "invalid json body")
		return
	}
	if req.AgentID == "" || req.ActionType == "" {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_PARAMS", "agent_id, action_type 必填")
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	f := store.families[familyID]
	if f == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "family 不存在")
		return
	}
	if f.AgentID != req.AgentID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "agent_id 不匹配")
		return
	}
	if f.ActionsUsed >= f.ActionQuota {
		writeError(w, http.StatusTooManyRequests, "QUOTA_EXCEEDED", "本回合动作次数已用尽")
		return
	}
	if req.IdempotencyKey != "" {
		key := f.ID + ":" + req.IdempotencyKey
		if _, exists := store.idempotency[key]; exists {
			writeError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "重复请求")
			return
		}
		store.idempotency[key] = struct{}{}
	}

	title, detail, errCode, errMsg := applyAction(f, req.ActionType)
	if errCode != "" {
		writeError(w, http.StatusBadRequest, errCode, errMsg)
		return
	}
	f.ActionsUsed++
	pushLogLocked(f.ID, LogEntry{TS: now(), Type: "action", Title: title, Detail: detail})
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "result": map[string]string{"title": title, "detail": detail}, "status": formatStatus(f)})
}

func handleNextTurn(w http.ResponseWriter, r *http.Request, familyID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	f := store.families[familyID]
	if f == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "family 不存在")
		return
	}

	f.Turn++
	f.ActionsUsed = 0
	f.ActionQuota = actionLimit
	f.ParentEnergy = 100
	f.ChildEnergy = 100
	f.Child.AgeMonth++
	f.Child.Stress = clamp(f.Child.Stress-randInt(0, 3), 0, 100)

	expense := 800 + f.Generation*200
	f.FamilyMoney = max(0, f.FamilyMoney-expense)

	events := []map[string]string{}
	if ev := maybeEvent(f); ev != nil {
		events = append(events, ev)
		pushLogLocked(f.ID, LogEntry{TS: now(), Type: "event", Title: ev["title"], Detail: ev["detail"]})
	}

	pushLogLocked(f.ID, LogEntry{TS: now(), Type: "system", Title: "进入下一回合", Detail: fmt.Sprintf("当前回合 %d", f.Turn)})
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "events": events, "status": formatStatus(f)})
}

func handleNextGeneration(w http.ResponseWriter, r *http.Request, familyID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	f := store.families[familyID]
	if f == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "family 不存在")
		return
	}
	if f.Child.AgeMonth < adultAgeMonth {
		writeError(w, http.StatusBadRequest, "NEXT_GEN_NOT_ALLOWED", "当前孩子未成年，无法进入下一代")
		return
	}

	f.ParentSnapshot = &ParentSnapshot{
		Intelligence: f.Child.Intelligence,
		Discipline:   f.Child.Discipline,
		Health:       f.Child.Health,
		Stress:       f.Child.Stress,
		SelfEsteem:   f.Child.SelfEsteem,
		Rebellion:    f.Child.Rebellion,
	}

	inherit := func(v int) int { return clamp(int(float64(v)*0.7)+randInt(-10, 10), 0, 100) }
	f.Generation++
	f.Turn++
	f.FamilyMoney = max(8000, int(float64(f.FamilyMoney)*0.65))
	f.ParentEnergy = 100
	f.ChildEnergy = 100
	f.ActionQuota = actionLimit
	f.ActionsUsed = 0
	f.Child = Child{
		ID:           nextID("child"),
		Name:         f.Name + "二代",
		Generation:   f.Generation,
		AgeMonth:     0,
		Intelligence: inherit(f.ParentSnapshot.Intelligence),
		Discipline:   inherit(f.ParentSnapshot.Discipline),
		Health:       inherit(f.ParentSnapshot.Health),
		Stress:       clamp(int(float64(f.ParentSnapshot.Stress)*0.35)+randInt(-5, 5), 0, 100),
		SelfEsteem:   inherit(f.ParentSnapshot.SelfEsteem),
		Rebellion:    clamp(int(float64(f.ParentSnapshot.Rebellion)*0.5)+randInt(-8, 8), 0, 100),
		StudyScore:   10,
	}

	pushLogLocked(f.ID, LogEntry{TS: now(), Type: "system", Title: "代际更替", Detail: fmt.Sprintf("已进入第 %d 代", f.Generation)})
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "status": formatStatus(f)})
}

func handleHistory(w http.ResponseWriter, r *http.Request, familyID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	store.mu.RLock()
	f := store.families[familyID]
	if f == nil {
		store.mu.RUnlock()
		writeError(w, http.StatusNotFound, "NOT_FOUND", "family 不存在")
		return
	}
	logs := store.logs[familyID]
	store.mu.RUnlock()

	limit := 30
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			if v < 1 {
				v = 1
			}
			if v > 100 {
				v = 100
			}
			limit = v
		}
	}
	if len(logs) > limit {
		logs = logs[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": logs})
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	type item struct {
		FamilyID   string `json:"family_id"`
		FamilyName string `json:"family_name"`
		Generation int    `json:"generation"`
		Turn       int    `json:"turn"`
		Score      int    `json:"score"`
		ChildName  string `json:"child_name"`
		StudyScore int    `json:"study_score"`
	}
	items := []item{}

	store.mu.RLock()
	for _, f := range store.families {
		c := f.Child
		score := int(float64(c.StudyScore)*0.35 +
			float64(c.Intelligence)*0.15 +
			float64(c.Discipline)*0.15 +
			float64(c.Health)*0.10 +
			float64(c.SelfEsteem)*0.10 +
			float64(100-c.Stress)*0.10 +
			float64(100-c.Rebellion)*0.05)
		items = append(items, item{
			FamilyID:   f.ID,
			FamilyName: f.Name,
			Generation: f.Generation,
			Turn:       f.Turn,
			Score:      score,
			ChildName:  c.Name,
			StudyScore: c.StudyScore,
		})
	}
	store.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	if len(items) > 50 {
		items = items[:50]
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(items), "items": items})
}

func applyAction(f *Family, action string) (title string, detail string, errCode string, errMsg string) {
	c := &f.Child
	switch strings.ToLower(action) {
	case "study":
		if f.ChildEnergy < 12 || f.ParentEnergy < 8 {
			return "", "", "INSUFFICIENT_RESOURCES", "精力不足，无法学习"
		}
		f.ChildEnergy -= 12
		f.ParentEnergy -= 8
		c.StudyScore = clamp(c.StudyScore+randInt(3, 6), 0, 100)
		c.Intelligence = clamp(c.Intelligence+randInt(1, 3), 0, 100)
		c.Stress = clamp(c.Stress+randInt(4, 8), 0, 100)
		c.SelfEsteem = clamp(c.SelfEsteem+randInt(-1, 1), 0, 100)
		return "安排学习", "学业提升，但压力上升", "", ""
	case "exercise":
		if f.ChildEnergy < 10 || f.ParentEnergy < 4 {
			return "", "", "INSUFFICIENT_RESOURCES", "精力不足，无法锻炼"
		}
		f.ChildEnergy -= 10
		f.ParentEnergy -= 4
		c.Health = clamp(c.Health+randInt(2, 5), 0, 100)
		c.Stress = clamp(c.Stress-randInt(3, 6), 0, 100)
		c.Rebellion = clamp(c.Rebellion-randInt(0, 2), 0, 100)
		return "体育锻炼", "健康提升，压力下降", "", ""
	case "rest":
		f.ChildEnergy = clamp(f.ChildEnergy+randInt(15, 22), 0, 100)
		f.ParentEnergy = clamp(f.ParentEnergy+randInt(8, 12), 0, 100)
		c.Stress = clamp(c.Stress-randInt(5, 9), 0, 100)
		c.SelfEsteem = clamp(c.SelfEsteem+randInt(1, 3), 0, 100)
		return "休息调整", "精力恢复，状态改善", "", ""
	case "talk":
		if f.ParentEnergy < 8 {
			return "", "", "INSUFFICIENT_RESOURCES", "家长精力不足，无法沟通"
		}
		f.ParentEnergy -= 8
		c.SelfEsteem = clamp(c.SelfEsteem+randInt(2, 5), 0, 100)
		c.Rebellion = clamp(c.Rebellion-randInt(2, 5), 0, 100)
		c.Stress = clamp(c.Stress-randInt(1, 3), 0, 100)
		return "亲子沟通", "自尊提升，叛逆下降", "", ""
	case "class":
		cost := 1500
		if f.FamilyMoney < cost {
			return "", "", "INSUFFICIENT_RESOURCES", "家庭资金不足，无法报班"
		}
		f.FamilyMoney -= cost
		f.ParentEnergy = clamp(f.ParentEnergy-6, 0, 100)
		c.StudyScore = clamp(c.StudyScore+randInt(4, 8), 0, 100)
		c.Intelligence = clamp(c.Intelligence+randInt(1, 2), 0, 100)
		c.Stress = clamp(c.Stress+randInt(1, 4), 0, 100)
		return "参加补习班", fmt.Sprintf("花费 %d，学业提升", cost), "", ""
	case "discipline":
		if f.ParentEnergy < 10 {
			return "", "", "INSUFFICIENT_RESOURCES", "家长精力不足，无法管教"
		}
		f.ParentEnergy -= 10
		c.Discipline = clamp(c.Discipline+randInt(2, 4), 0, 100)
		c.Rebellion = clamp(c.Rebellion+randInt(-1, 4), 0, 100)
		c.SelfEsteem = clamp(c.SelfEsteem-randInt(1, 4), 0, 100)
		c.Stress = clamp(c.Stress+randInt(2, 5), 0, 100)
		return "严格管教", "自律提升，但心理负担加重", "", ""
	default:
		return "", "", "INVALID_ACTION", "不支持的 action_type"
	}
}

func maybeEvent(f *Family) map[string]string {
	roll := rand.Float64()
	c := &f.Child
	if roll < 0.15 {
		gain := randInt(1000, 3000)
		f.FamilyMoney += gain
		return map[string]string{"title": "奖学金", "detail": fmt.Sprintf("孩子获得奖学金 +%d", gain)}
	}
	if roll < 0.25 {
		c.Stress = clamp(c.Stress+randInt(6, 12), 0, 100)
		c.SelfEsteem = clamp(c.SelfEsteem-randInt(2, 5), 0, 100)
		return map[string]string{"title": "考试失利", "detail": "考试失利，压力上升"}
	}
	if roll < 0.32 {
		f.FamilyMoney = max(0, f.FamilyMoney-2000)
		c.Health = clamp(c.Health-randInt(3, 8), 0, 100)
		return map[string]string{"title": "意外生病", "detail": "医疗开销增加，健康下滑"}
	}
	return nil
}

func formatStatus(f *Family) map[string]interface{} {
	suggestions := []string{}
	if f.Child.Stress > 70 {
		suggestions = []string{"rest", "talk", "exercise"}
	} else if f.Child.StudyScore < 40 {
		suggestions = []string{"study", "class"}
	} else {
		suggestions = []string{"study", "exercise", "talk"}
	}

	return map[string]interface{}{
		"family": map[string]interface{}{
			"family_id":  f.ID,
			"name":       f.Name,
			"generation": f.Generation,
			"turn":       f.Turn,
		},
		"resources": map[string]interface{}{
			"family_money":  f.FamilyMoney,
			"parent_energy": f.ParentEnergy,
			"child_energy":  f.ChildEnergy,
			"action_quota":  f.ActionQuota,
			"actions_used":  f.ActionsUsed,
		},
		"child":             f.Child,
		"suggested_actions": suggestions,
	}
}

func pushLogLocked(familyID string, entry LogEntry) {
	logs := store.logs[familyID]
	logs = append([]LogEntry{entry}, logs...)
	if len(logs) > 200 {
		logs = logs[:200]
	}
	store.logs[familyID] = logs
}

func now() string {
	return time.Now().Format(time.RFC3339)
}

func nextID(prefix string) string {
	idCounter++
	return fmt.Sprintf("%s_%d", prefix, idCounter)
}

func randInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
