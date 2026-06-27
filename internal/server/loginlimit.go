package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// loginLimiter 基于 IP 对登录失败进行限流。
// 仅统计失败的尝试，避免误伤正常浏览。
// 在 window 时间窗口内，某 IP 的失败次数达到 maxFails 后即被锁定 lockDur。
type loginLimiter struct {
	mu       sync.Mutex
	maxFails int
	window   time.Duration
	lockDur  time.Duration
	fails    map[string]*failState
}

type failState struct {
	count    int
	firstAt  time.Time // 窗口内首次失败时间
	lockedAt time.Time // 进入锁定的时间；零值表示未锁定
}

func newLoginLimiter(maxFails int, window, lockDur time.Duration) *loginLimiter {
	return &loginLimiter{
		maxFails: maxFails,
		window:   window,
		lockDur:  lockDur,
		fails:    make(map[string]*failState),
	}
}

// ipFromRequest 提取客户端 IP，优先使用 X-Forwarded-For 的首个地址。
func ipFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return net.ParseIP(strings.TrimSpace(xff[:i])).String()
			}
		}
		return net.ParseIP(strings.TrimSpace(xff)).String()
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isLocked 返回是否处于锁定状态。锁定过期会自动解除。
func (l *loginLimiter) isLocked(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	st, ok := l.fails[ip]
	if !ok || st.lockedAt.IsZero() {
		return false
	}
	if now.Sub(st.lockedAt) >= l.lockDur {
		// 锁定过期，清空计数重新开始
		delete(l.fails, ip)
		return false
	}
	return true
}

// recordFailure 记录一次失败尝试，必要时触发锁定。
func (l *loginLimiter) recordFailure(ip string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	st, ok := l.fails[ip]
	if !ok {
		st = &failState{firstAt: now}
		l.fails[ip] = st
	}
	// 窗口过期则重置计数
	if now.Sub(st.firstAt) >= l.window {
		st.count = 0
		st.firstAt = now
		st.lockedAt = time.Time{}
	}
	st.count++
	if st.count >= l.maxFails && st.lockedAt.IsZero() {
		st.lockedAt = now
	}
}

// recordSuccess 登录成功时清除该 IP 的失败记录。
func (l *loginLimiter) recordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, ip)
}
