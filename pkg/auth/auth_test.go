package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ============================================================================
// auth_test.go — auth 包单元测试
// ============================================================================

const testSecret = "test-secret-key-for-unit-tests"

func TestGenerateToken_ValidateToken_Roundtrip(t *testing.T) {
	tests := []struct {
		caseName string
		userID   uint
		role     string
		userName string
	}{
		{"普通用户", 1, "user", "张三"},
		{"管理员", 100, "admin", "管理员"},
		{"零 ID", 0, "guest", ""},
		{"大 ID", 4294967295, "user", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.caseName, func(t *testing.T) {
			token, err := GenerateToken(tt.userID, tt.role, tt.userName, testSecret)
			if err != nil {
				t.Fatalf("GenerateToken() 失败: %v", err)
			}
			if token == "" {
				t.Fatal("GenerateToken() 返回空 token")
			}

			claims, err := ValidateToken(token, testSecret)
			if err != nil {
				t.Fatalf("ValidateToken() 失败: %v", err)
			}
			if claims.UserID != tt.userID {
				t.Errorf("UserID = %d; want %d", claims.UserID, tt.userID)
			}
			if claims.Name != tt.userName {
				t.Errorf("Name = %q; want %q", claims.Name, tt.userName)
			}
			if claims.Subject != tt.userName {
				t.Errorf("Subject = %q; want %q", claims.Subject, tt.userName)
			}

			// 验证有效期
			if claims.ExpiresAt == nil {
				t.Error("ExpiresAt 不应为 nil")
			} else if claims.ExpiresAt.Time.Before(time.Now()) {
				t.Error("token 已过期")
			}
			if claims.IssuedAt == nil {
				t.Error("IssuedAt 不应为 nil")
			}
		})
	}
}

func TestValidateToken_InvalidInputs(t *testing.T) {
	validToken, _ := GenerateToken(1, "user", "test", testSecret)

	tests := []struct {
		caseName    string
		tokenString string
		secret      string
		wantErr     bool
		errContains string
	}{
		{
			caseName:    "空 token",
			tokenString: "",
			secret:      testSecret,
			wantErr:     true,
		},
		{
			caseName:    "错误密钥",
			tokenString: validToken,
			secret:      "wrong-secret",
			wantErr:     true,
		},
		{
			caseName:    "非法格式",
			tokenString: "not.a.valid.jwt.token",
			secret:      testSecret,
			wantErr:     true,
		},
		{
			caseName:    "篡改过的 token",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.tampered.xxx",
			secret:      testSecret,
			wantErr:     true,
		},
		{
			caseName:    "空密钥",
			tokenString: validToken,
			secret:      "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.caseName, func(t *testing.T) {
			_, err := ValidateToken(tt.tokenString, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v; wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateToken() error = %q; 期望包含 %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// 手动创建一个已过期的 token
	claims := &Claims{
		UserID: 1,
		Name:   "test",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("创建过期 token 失败: %v", err)
	}

	_, err = ValidateToken(tokenString, testSecret)
	if err == nil {
		t.Error("ValidateToken() 对过期 token 应返回错误")
	}
}

func TestClaims_Struct(t *testing.T) {
	claims := &Claims{
		UserID: 42,
		Name:   "测试用户",
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d; want 42", claims.UserID)
	}
	if claims.Name != "测试用户" {
		t.Errorf("Name = %q; want 测试用户", claims.Name)
	}
}
