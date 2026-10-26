// Package auth — 认证基础件：手写 HS256 JWT（禁三方 jwt 库，version-lock 裁定）、
// 密码校验（plain: 直等 + bcrypt 兜底）、Bearer/租户守卫（镜像 rails 同门）。
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// env 必需四件（fail-fast，无兜底——suite-hard-rules §1）。
func jwtEnv() (key, issuer, audience string, ttl int, err error) {
	key = os.Getenv("JWT_SIGNING_KEY")
	if key == "" {
		return "", "", "", 0, errors.New("缺少环境变量 JWT_SIGNING_KEY（禁默认值兜底）")
	}
	issuer = os.Getenv("JWT_ISSUER")
	if issuer == "" {
		return "", "", "", 0, errors.New("缺少环境变量 JWT_ISSUER（禁默认值兜底）")
	}
	audience = os.Getenv("JWT_AUDIENCE")
	if audience == "" {
		return "", "", "", 0, errors.New("缺少环境变量 JWT_AUDIENCE（禁默认值兜底）")
	}
	raw := os.Getenv("JWT_TTL_SECONDS")
	ttl, err = strconv.Atoi(raw)
	if err != nil || ttl <= 0 {
		return "", "", "", 0, fmt.Errorf("JWT_TTL_SECONDS 非法 %q: %w", raw, err)
	}
	return key, issuer, audience, ttl, nil
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// TTLSeconds 取 JWT_TTL_SECONDS（login payload expires_in 用；fail-fast 同 encode）。
func TTLSeconds() (int, error) {
	_, _, _, ttl, err := jwtEnv()
	return ttl, err
}

func hs256(key string, signingInput string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(signingInput))
	return b64(mac.Sum(nil))
}

// Encode 签发 HS256：payload 注入 iat/exp/iss/aud（rails Auth::JwtVerifier.encode 同形）。
func Encode(payload map[string]any) (string, error) {
	key, issuer, audience, ttl, err := jwtEnv()
	if err != nil {
		return "", err
	}
	now := time.Now().Unix()
	full := make(map[string]any, len(payload)+4)
	for k, v := range payload {
		full[k] = v
	}
	full["iat"] = now
	full["exp"] = now + int64(ttl)
	full["iss"] = issuer
	full["aud"] = audience

	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(full)
	if err != nil {
		return "", err
	}
	signingInput := b64(header) + "." + b64(body)
	return signingInput + "." + hs256(key, signingInput), nil
}

// Verify 验签 HS256：结构三段、header alg 恒等 HS256（alg 混淆拒）、签名 HMAC 比对、
// exp/iss/aud 全验。任一不过返回带语义的错误（上层统一转 401）。
func Verify(token string) (map[string]any, error) {
	key, issuer, audience, _, err := jwtEnv()
	if err != nil {
		return nil, err
	}
	parts := splitSegments(token)
	if len(parts) != 3 {
		return nil, errors.New("token 段数非 3")
	}
	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("header 解码失败: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		return nil, fmt.Errorf("header 解析失败: %w", err)
	}
	if header.Alg != "HS256" {
		return nil, fmt.Errorf("alg %q 非法（仅接受 HS256）", header.Alg)
	}
	signingInput := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(hs256(key, signingInput)), []byte(parts[2])) {
		return nil, errors.New("签名校验失败")
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("payload 解码失败: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadRaw, &claims); err != nil {
		return nil, fmt.Errorf("payload 解析失败: %w", err)
	}
	if exp, ok := numField(claims, "exp"); ok && time.Now().Unix() >= int64(exp) {
		return nil, errors.New("token 已过期")
	}
	if iss, _ := claims["iss"].(string); iss != issuer {
		return nil, fmt.Errorf("iss %q 不符", iss)
	}
	if aud, _ := claims["aud"].(string); aud != audience {
		return nil, fmt.Errorf("aud %q 不符", aud)
	}
	return claims, nil
}

func splitSegments(tok string) []string {
	segs := []string{}
	cur := []rune{}
	for _, r := range tok {
		if r == '.' {
			segs = append(segs, string(cur))
			cur = []rune{}
			continue
		}
		cur = append(cur, r)
	}
	return append(segs, string(cur))
}

// numField 取数值 claim（json 反序列化数字落 float64）。
func numField(claims map[string]any, key string) (float64, bool) {
	v, ok := claims[key].(float64)
	return v, ok
}
