package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"auth-service/internal/domain"
	"auth-service/internal/usecase"
	"auth-service/pkg/bcrypt"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// --- fakes -----------------------------------------------------------------

type fakeUsers struct {
	mu    sync.Mutex
	users map[uint]*domain.User
}

func (f *fakeUsers) Create(u *domain.User) error { f.users[u.ID] = u; return nil }
func (f *fakeUsers) Update(u *domain.User) error { f.users[u.ID] = u; return nil }
func (f *fakeUsers) FindByID(id uint) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.users[id]; ok {
		c := *u
		return &c, nil
	}
	return nil, errors.New("not found")
}
func (f *fakeUsers) FindByEmail(email string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Email == email {
			c := *u
			return &c, nil
		}
	}
	return nil, errors.New("not found")
}
func (f *fakeUsers) FindByRole(string) ([]domain.User, error) { return nil, nil }

// fakeSessions mirrors the semantics of the real repository: unique tokens,
// a per-user cap that evicts the oldest, atomic rotation, ownership-checked
// single-session revocation.
type fakeSessions struct {
	mu   sync.Mutex
	seq  uint
	rows []domain.RefreshToken
}

func (f *fakeSessions) Find(token string) (*domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.Token == token {
			c := r
			return &c, nil
		}
	}
	return nil, errors.New("record not found")
}

func (f *fakeSessions) insertLocked(t *domain.RefreshToken) error {
	for _, r := range f.rows {
		if r.Token == t.Token {
			return errors.New("duplicate key value violates unique constraint")
		}
	}
	f.seq++
	c := *t
	c.ID = f.seq
	f.rows = append(f.rows, c)
	return nil
}

func (f *fakeSessions) Create(t *domain.RefreshToken, maxSessions int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.insertLocked(t); err != nil {
		return err
	}
	if maxSessions > 0 {
		var mine []int
		for i, r := range f.rows {
			if r.UserID == t.UserID {
				mine = append(mine, i)
			}
		}
		if extra := len(mine) - maxSessions; extra > 0 {
			drop := map[int]bool{}
			for _, i := range mine[:extra] { // rows are appended in creation order
				drop[i] = true
			}
			var kept []domain.RefreshToken
			for i, r := range f.rows {
				if !drop[i] {
					kept = append(kept, r)
				}
			}
			f.rows = kept
		}
	}
	return nil
}

func (f *fakeSessions) Rotate(old string, next *domain.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, r := range f.rows {
		if r.Token == old {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			return f.insertLocked(next)
		}
	}
	return domain.ErrTokenNotFound
}

func (f *fakeSessions) DeleteSession(userID uint, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, r := range f.rows {
		if r.UserID == userID && r.Token == token {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeSessions) DeleteByUser(userID uint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	var kept []domain.RefreshToken
	for _, r := range f.rows {
		if r.UserID != userID {
			kept = append(kept, r)
		}
	}
	f.rows = kept
	return nil
}

// --- harness ---------------------------------------------------------------

const testPassword = "password123"

var testPasswordHash string

func init() {
	h, err := bcrypt.Hash(testPassword)
	if err != nil {
		panic(err)
	}
	testPasswordHash = h
}

type rig struct {
	t        *testing.T
	router   *gin.Engine
	sessions *fakeSessions
	secret   []byte
}

func newRig(t *testing.T, maxSessions int) *rig {
	t.Helper()
	gin.SetMode(gin.TestMode)

	users := &fakeUsers{users: map[uint]*domain.User{
		1: {ID: 1, Email: "alice@example.com", Password: testPasswordHash, Role: "user"},
		2: {ID: 2, Email: "bob@example.com", Password: testPasswordHash, Role: "user"},
	}}
	sessions := &fakeSessions{}
	secret := []byte("test-secret")
	h := NewAuthHandler(usecase.NewAuthUsecase(users), sessions, secret, nil, maxSessions)

	r := gin.New()
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/logout", h.Logout)
	r.POST("/change-password", h.ChangePassword)
	return &rig{t: t, router: r, sessions: sessions, secret: secret}
}

func (g *rig) do(path, body, userID string) (int, map[string]any) {
	g.t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("X-User-ID", userID) // set by the gateway after JWT validation
	}
	w := httptest.NewRecorder()
	g.router.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func (g *rig) login(email string) string {
	g.t.Helper()
	code, out := g.do("/login", `{"email":"`+email+`","password":"`+testPassword+`"}`, "")
	if code != http.StatusOK {
		g.t.Fatalf("login failed: %d %v", code, out)
	}
	return out["data"].(map[string]any)["refresh_token"].(string)
}

func (g *rig) refresh(token string) (int, string) {
	g.t.Helper()
	code, out := g.do("/refresh", `{"refresh_token":"`+token+`"}`, "")
	if code != http.StatusOK {
		return code, ""
	}
	return code, out["data"].(map[string]any)["refresh_token"].(string)
}

func (g *rig) mustRefresh(token string) string {
	g.t.Helper()
	code, next := g.refresh(token)
	if code != http.StatusOK {
		g.t.Fatalf("expected the session to refresh, got %d", code)
	}
	return next
}

func (g *rig) mustBeRejected(token string) {
	g.t.Helper()
	if code, _ := g.refresh(token); code != http.StatusUnauthorized {
		g.t.Fatalf("expected 401 for a revoked/used refresh token, got %d", code)
	}
}

// --- tests -----------------------------------------------------------------

// The reported bug: signing in on a second device signed the first one out.
func TestSessions_TwoDevicesStaySignedIn(t *testing.T) {
	g := newRig(t, 5)
	deviceA := g.login("alice@example.com")
	deviceB := g.login("alice@example.com")

	g.mustRefresh(deviceA)
	g.mustRefresh(deviceB)
}

// Every refresh used to delete ALL of the user's tokens, so two devices
// signed each other out every couple of hours.
func TestSessions_RefreshingOneDeviceLeavesTheOthersSignedIn(t *testing.T) {
	g := newRig(t, 5)
	deviceA := g.login("alice@example.com")
	deviceB := g.login("alice@example.com")

	newB := g.mustRefresh(deviceB)
	g.mustBeRejected(deviceB) // B's old token is used up (rotation)...
	g.mustRefresh(deviceA)    // ...but A is untouched
	g.mustRefresh(newB)
}

func TestSessions_ReplayedRefreshTokenIsRejected(t *testing.T) {
	g := newRig(t, 5)
	old := g.login("alice@example.com")
	g.mustRefresh(old)
	g.mustBeRejected(old)
}

func TestSessions_ConcurrentRefreshOfOneTokenSucceedsOnce(t *testing.T) {
	g := newRig(t, 5)
	token := g.login("alice@example.com")

	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if code, _ := g.refresh(token); code == http.StatusOK {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("exactly one concurrent refresh of the same token may win, %d did", wins)
	}
}

func TestSessions_LogoutWithRefreshTokenEndsOnlyThatDevice(t *testing.T) {
	g := newRig(t, 5)
	deviceA := g.login("alice@example.com")
	deviceB := g.login("alice@example.com")

	if code, _ := g.do("/logout", `{"refresh_token":"`+deviceB+`"}`, "1"); code != http.StatusOK {
		t.Fatalf("logout failed: %d", code)
	}
	g.mustRefresh(deviceA)
	g.mustBeRejected(deviceB)
}

// Clients that send no body (the current Flutter app) keep getting "sign out
// everywhere".
func TestSessions_LogoutWithoutBodyEndsEveryDevice(t *testing.T) {
	g := newRig(t, 5)
	deviceA := g.login("alice@example.com")
	deviceB := g.login("alice@example.com")

	if code, _ := g.do("/logout", ``, "1"); code != http.StatusOK {
		t.Fatalf("logout failed: %d", code)
	}
	g.mustBeRejected(deviceA)
	g.mustBeRejected(deviceB)
}

func TestSessions_LogoutCannotRevokeSomeoneElsesSession(t *testing.T) {
	g := newRig(t, 5)
	aliceSession := g.login("alice@example.com")

	// Bob presents Alice's refresh token to /logout.
	if code, _ := g.do("/logout", `{"refresh_token":"`+aliceSession+`"}`, "2"); code != http.StatusOK {
		t.Fatalf("logout failed: %d", code)
	}
	g.mustRefresh(aliceSession) // still valid
}

func TestSessions_CapEvictsTheOldestSession(t *testing.T) {
	g := newRig(t, 3)
	first := g.login("alice@example.com")
	second := g.login("alice@example.com")
	third := g.login("alice@example.com")
	fourth := g.login("alice@example.com") // over the cap of 3

	g.mustBeRejected(first)
	g.mustRefresh(second)
	g.mustRefresh(third)
	g.mustRefresh(fourth)
}

func TestSessions_CapIsPerUser(t *testing.T) {
	g := newRig(t, 1)
	alice := g.login("alice@example.com")
	g.login("bob@example.com") // must not evict Alice's session
	g.mustRefresh(alice)
}

// Without a jti, two refresh tokens issued to one user in the same second are
// byte-identical and collide on the unique column.
func TestSessions_TokensAreUniqueEvenWithinOneSecond(t *testing.T) {
	g := newRig(t, 50)
	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		tok := g.login("alice@example.com")
		if seen[tok] {
			t.Fatal("two logins produced the identical refresh token")
		}
		seen[tok] = true
	}
}

func TestSessions_ChangePasswordSignsEveryDeviceOut(t *testing.T) {
	g := newRig(t, 5)
	deviceA := g.login("alice@example.com")
	deviceB := g.login("alice@example.com")

	code, out := g.do("/change-password", `{"old_password":"`+testPassword+`","new_password":"password456"}`, "1")
	if code != http.StatusOK {
		t.Fatalf("change-password failed: %d %v", code, out)
	}
	g.mustBeRejected(deviceA)
	g.mustBeRejected(deviceB)
}

func TestSessions_ChangePasswordWithWrongOldPasswordKeepsSessions(t *testing.T) {
	g := newRig(t, 5)
	deviceA := g.login("alice@example.com")

	if code, _ := g.do("/change-password", `{"old_password":"nope-nope","new_password":"password456"}`, "1"); code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", code)
	}
	g.mustRefresh(deviceA)
}

// A session stored for one user must not be usable through a token that
// claims to be another user's.
func TestSessions_RefreshRejectsATokenWhoseSessionBelongsToAnotherUser(t *testing.T) {
	g := newRig(t, 5)
	claimsAlice := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 1, "role": "user", "exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := claimsAlice.SignedString(g.secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.sessions.Create(&domain.RefreshToken{UserID: 2, Token: signed}, 5); err != nil {
		t.Fatal(err)
	}
	g.mustBeRejected(signed)
}
