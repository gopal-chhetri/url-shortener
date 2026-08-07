package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newDemoQuotaCtx(t *testing.T, ip string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/api/v1/shorten", nil)
	ctx.Request.RemoteAddr = ip + ":1234"
	return ctx
}

func TestDemoQuota_InMemory_LimitPerIP(t *testing.T) {
	q := NewDemoQuota(nil, 3, 24)

	// Each request generates a fresh cookie, so the IP identity is what limits
	// a cookie-clearing client: 3 allowed, 4th blocked.
	for i := 0; i < 3; i++ {
		ctx := newDemoQuotaCtx(t, "10.0.0.1")
		allowed, remaining := q.Take(ctx)
		assert.True(t, allowed, "request %d should be allowed", i+1)
		assert.GreaterOrEqual(t, remaining, 0)
	}

	ctx := newDemoQuotaCtx(t, "10.0.0.1")
	allowed, _ := q.Take(ctx)
	assert.False(t, allowed, "4th request from same IP should be blocked")

	ctx2 := newDemoQuotaCtx(t, "10.0.0.2")
	allowed2, _ := q.Take(ctx2)
	assert.True(t, allowed2, "different IP has its own quota")
}

func TestDemoQuota_ReleaseReturnsCapacity(t *testing.T) {
	q := NewDemoQuota(nil, 3, 60)

	for i := 0; i < 3; i++ {
		ctx := newDemoQuotaCtx(t, "10.0.0.5")
		assert.True(t, func() bool { ok, _ := q.Take(ctx); return ok }())
	}

	ctxFull := newDemoQuotaCtx(t, "10.0.0.5")
	allowed, _ := q.Take(ctxFull)
	assert.False(t, allowed)

	// Release one, now it should be allowed again
	q.Release(ctxFull)
	ctxAgain := newDemoQuotaCtx(t, "10.0.0.5")
	allowed, _ = q.Take(ctxAgain)
	assert.True(t, allowed)
}

func TestDemoQuota_EnsureCookieStagesPersistentValue(t *testing.T) {
	q := NewDemoQuota(nil, 3, 24)
	ctx := newDemoQuotaCtx(t, "10.0.0.9")
	q.Take(ctx)
	q.EnsureCookie(ctx)

	got, ok := ctx.Get("demo_cookie")
	assert.True(t, ok, "demo cookie should be staged on context")
	assert.NotEmpty(t, got)
}
