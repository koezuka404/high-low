package usecase

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

const rateLimitIPKeyUnknown = "unknown"

var (
	emailLocalPartPattern  = `[a-zA-Z0-9](?:[a-zA-Z0-9!#$%&'*+/=?^_{|}~\-.]*[a-zA-Z0-9])?`
	emailDomainPartPattern = `(?:[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}`
	rateLimitEmailPattern  = regexp.MustCompile(`^` + emailLocalPartPattern + `@` + emailDomainPartPattern + `$`)
)

func NormalizeEmailForRateLimit(email string) (string, error) {
	s := strings.TrimSpace(email)
	if s == "" {
		return "", fmt.Errorf("email is required")
	}
	if strings.Count(s, "@") != 1 {
		return "", fmt.Errorf("invalid email format")
	}
	if !rateLimitEmailPattern.MatchString(s) {
		return "", fmt.Errorf("invalid email format")
	}
	return strings.ToLower(s), nil
}

func RateLimitIPKeyAndCost(raw string, defaultTokenCost float64) (suffix string, tokenCost float64) {
	if defaultTokenCost < 0 {
		defaultTokenCost = 0
	}
	s := strings.TrimSpace(raw)
	if s == "" {
		return rateLimitIPKeyUnknown, 0
	}

	if strings.HasPrefix(s, "[") {
		if end := strings.Index(s, "]"); end > 1 {
			inner := s[1:end]
			if ip := net.ParseIP(inner); ip != nil {
				return ip.String(), defaultTokenCost
			}
		}
	}

	host := s
	if h, _, err := net.SplitHostPort(s); err == nil {
		host = h
		if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") && len(host) >= 2 {
			host = host[1 : len(host)-1]
		}
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), defaultTokenCost
	}

	return rateLimitIPKeyUnknown, 0
}

func ConsumeAuthRateLimit(ctx context.Context, rl RateLimiter, rlp RateLimitParams, rawIP string, rawEmail string, ipKeyPrefix string, emailKeyPrefix string) (normalizedEmail string, err error) {
	emailNorm, err := NormalizeEmailForRateLimit(rawEmail)
	if err != nil {
		return "", err
	}
	if rl == nil {
		return emailNorm, nil
	}

	now := float64(time.Now().Unix())

	ipSuffix, ipCost := RateLimitIPKeyAndCost(rawIP, rlp.TokenCost)
	allowed, retryAfterSec, err := rl.ConsumeToken(ctx, ipKeyPrefix+ipSuffix, now, rlp.Capacity, rlp.RefillRate, ipCost, rlp.TTLSec)
	if err != nil {
		return "", fmt.Errorf("rate limit check: %w", err)
	}
	if !allowed {
		return "", &RateLimitError{RetryAfterSec: retryAfterSec}
	}

	allowed, retryAfterSec, err = rl.ConsumeToken(ctx, emailKeyPrefix+emailNorm, now, rlp.Capacity, rlp.RefillRate, rlp.TokenCost, rlp.TTLSec)
	if err != nil {
		return "", fmt.Errorf("rate limit check: %w", err)
	}
	if !allowed {
		return "", &RateLimitError{RetryAfterSec: retryAfterSec}
	}

	return emailNorm, nil
}
