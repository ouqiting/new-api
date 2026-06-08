package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func configureModelRequestRateLimitTest(t *testing.T) {
	t.Helper()

	oldRedisEnabled := common.RedisEnabled
	oldEnabled := setting.ModelRequestRateLimitEnabled
	oldDuration := setting.ModelRequestRateLimitDurationMinutes
	oldTotalCount := setting.ModelRequestRateLimitCount
	oldSuccessCount := setting.ModelRequestRateLimitSuccessCount

	setting.ModelRequestRateLimitMutex.Lock()
	oldGroup := setting.ModelRequestRateLimitGroup
	setting.ModelRequestRateLimitGroup = map[string][2]int{}
	setting.ModelRequestRateLimitMutex.Unlock()

	common.RedisEnabled = false
	setting.ModelRequestRateLimitEnabled = true
	setting.ModelRequestRateLimitDurationMinutes = 1
	setting.ModelRequestRateLimitCount = 1
	setting.ModelRequestRateLimitSuccessCount = 1000
	inMemoryRateLimiter = common.InMemoryRateLimiter{}

	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
		setting.ModelRequestRateLimitEnabled = oldEnabled
		setting.ModelRequestRateLimitDurationMinutes = oldDuration
		setting.ModelRequestRateLimitCount = oldTotalCount
		setting.ModelRequestRateLimitSuccessCount = oldSuccessCount
		setting.ModelRequestRateLimitMutex.Lock()
		setting.ModelRequestRateLimitGroup = oldGroup
		setting.ModelRequestRateLimitMutex.Unlock()
		inMemoryRateLimiter = common.InMemoryRateLimiter{}
	})
}

func performModelRateLimitRequest(userID int, role *int) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/chat/completions", func(c *gin.Context) {
		c.Set("id", userID)
		if role != nil {
			c.Set("role", *role)
		}
		c.Next()
	}, ModelRequestRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestModelRequestRateLimitSkipsRootUser(t *testing.T) {
	configureModelRequestRateLimitTest(t)

	role := common.RoleRootUser
	require.Equal(t, http.StatusOK, performModelRateLimitRequest(1, &role).Code)
	require.Equal(t, http.StatusOK, performModelRateLimitRequest(1, &role).Code)
}

func TestModelRequestRateLimitStillLimitsCommonUser(t *testing.T) {
	configureModelRequestRateLimitTest(t)

	role := common.RoleCommonUser
	require.Equal(t, http.StatusOK, performModelRateLimitRequest(2, &role).Code)
	require.Equal(t, http.StatusTooManyRequests, performModelRateLimitRequest(2, &role).Code)
}

func TestModelRequestRateLimitFallsBackToRootUserLookup(t *testing.T) {
	configureModelRequestRateLimitTest(t)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
	})

	require.NoError(t, model.DB.Create(&model.User{
		Id:       3,
		Username: "root-rate-limit-test",
		Password: "password",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
	}).Error)

	require.Equal(t, http.StatusOK, performModelRateLimitRequest(3, nil).Code)
	require.Equal(t, http.StatusOK, performModelRateLimitRequest(3, nil).Code)
}
