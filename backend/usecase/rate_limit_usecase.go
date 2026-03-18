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

func EmailKeyPartForAuthRateLimit(rawEmail string) (string, error) {
	s := strings.TrimSpace(rawEmail)
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

func IPKeyPartAndCostForAuthRateLimit(rawIP string, defaultTokenCost float64) (keyPart string, tokenCost float64) {
	if defaultTokenCost < 0 {
		defaultTokenCost = 0
	}
	s := strings.TrimSpace(rawIP)
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
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), defaultTokenCost
	}

	return rateLimitIPKeyUnknown, 0
}

func EnforceAuthRateLimit(ctx context.Context, rl RateLimiter, rlp RateLimitParams, rawIP string, rawEmail string, ipKeyPrefix string, emailKeyPrefix string) (emailKeyPart string, err error) {
	emailNorm, err := EmailKeyPartForAuthRateLimit(rawEmail)
	if err != nil {
		return "", err
	}
	if rl == nil {
		return emailNorm, nil
	}

	now := float64(time.Now().Unix())

	ipSuffix, ipCost := IPKeyPartAndCostForAuthRateLimit(rawIP, rlp.TokenCost)
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
