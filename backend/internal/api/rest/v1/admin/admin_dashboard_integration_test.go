package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupIntegrationDB creates a real SQLite DB wrapped in the database.DB interface.
func setupIntegrationDB(t *testing.T) database.DB {
	t.Helper()
	gormDB := testkit.SetupTestDB(t)
	return database.NewGormWrapper(gormDB)
}

// createAdminIntegrationContext builds a Gin context with admin user set.
func createAdminIntegrationContext(w *httptest.ResponseRecorder, adminID int64) *gin.Context {
	c, _ := gin.CreateTestContext(w)
	c.Set("admin_user_id", adminID)
	c.Set("admin_user", &user.User{ID: adminID, Email: "admin@test.com", IsSystemAdmin: true})
	return c
}

func TestAdminDashboard_Stats(t *testing.T) {
	db := setupIntegrationDB(t)
	gormDB := db.GormDB()

	// Seed data: 3 users, 2 orgs, 1 runner
	adminID := testkit.CreateUser(t, gormDB, "admin@test.com", "admin")
	testkit.CreateUser(t, gormDB, "user1@test.com", "user1")
	testkit.CreateUser(t, gormDB, "user2@test.com", "user2")

	orgID := testkit.CreateOrg(t, gormDB, "org-alpha", adminID)
	testkit.CreateOrg(t, gormDB, "org-beta", adminID)

	testkit.CreateRunner(t, gormDB, orgID, "runner-node-1")

	svc := adminservice.NewService(db)
	handler := NewDashboardHandler(svc)

	w := httptest.NewRecorder()
	c := createAdminIntegrationContext(w, adminID)
	c.Request = httptest.NewRequest("GET", "/dashboard/stats", nil)

	handler.GetStats(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	require.NoError(t, err)

	assert.Equal(t, float64(3), stats["total_users"])
	assert.Equal(t, float64(3), stats["active_users"])
	assert.Equal(t, float64(2), stats["total_organizations"])
	assert.Equal(t, float64(1), stats["total_runners"])
	assert.Equal(t, float64(1), stats["online_runners"])
}

func TestAdminUser_ToggleActive(t *testing.T) {
	db := setupIntegrationDB(t)
	gormDB := db.GormDB()

	adminID := testkit.CreateUser(t, gormDB, "admin@test.com", "admin")
	targetID := testkit.CreateUser(t, gormDB, "target@test.com", "target")

	svc := adminservice.NewService(db)
	handler := NewUserHandler(svc)

	// Step 1: Disable user
	w := httptest.NewRecorder()
	c := createAdminIntegrationContext(w, adminID)
	c.Request = httptest.NewRequest("POST", "/users/"+itoa(targetID)+"/disable", nil)
	c.Params = gin.Params{{Key: "id", Value: itoa(targetID)}}

	handler.DisableUser(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var disableResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &disableResp))
	assert.Equal(t, false, disableResp["is_active"])

	// Step 2: Enable user
	w = httptest.NewRecorder()
	c = createAdminIntegrationContext(w, adminID)
	c.Request = httptest.NewRequest("POST", "/users/"+itoa(targetID)+"/enable", nil)
	c.Params = gin.Params{{Key: "id", Value: itoa(targetID)}}

	handler.EnableUser(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var enableResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &enableResp))
	assert.Equal(t, true, enableResp["is_active"])
}

func TestAdminUser_DisableSelfPrevented(t *testing.T) {
	db := setupIntegrationDB(t)
	gormDB := db.GormDB()

	adminID := testkit.CreateUser(t, gormDB, "admin@test.com", "admin")

	svc := adminservice.NewService(db)
	handler := NewUserHandler(svc)

	w := httptest.NewRecorder()
	c := createAdminIntegrationContext(w, adminID)
	c.Request = httptest.NewRequest("POST", "/users/"+itoa(adminID)+"/disable", nil)
	c.Params = gin.Params{{Key: "id", Value: itoa(adminID)}}

	handler.DisableUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminUser_GrantRevokeAdmin(t *testing.T) {
	db := setupIntegrationDB(t)
	gormDB := db.GormDB()

	adminID := testkit.CreateUser(t, gormDB, "admin@test.com", "admin")
	targetID := testkit.CreateUser(t, gormDB, "target@test.com", "target")

	svc := adminservice.NewService(db)
	handler := NewUserHandler(svc)

	// Step 1: Grant admin
	w := httptest.NewRecorder()
	c := createAdminIntegrationContext(w, adminID)
	c.Request = httptest.NewRequest("POST", "/users/"+itoa(targetID)+"/grant-admin", nil)
	c.Params = gin.Params{{Key: "id", Value: itoa(targetID)}}

	handler.GrantAdmin(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var grantResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &grantResp))
	assert.Equal(t, true, grantResp["is_system_admin"])

	// Step 2: Revoke admin
	w = httptest.NewRecorder()
	c = createAdminIntegrationContext(w, adminID)
	c.Request = httptest.NewRequest("POST", "/users/"+itoa(targetID)+"/revoke-admin", nil)
	c.Params = gin.Params{{Key: "id", Value: itoa(targetID)}}

	handler.RevokeAdmin(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var revokeResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &revokeResp))
	assert.Equal(t, false, revokeResp["is_system_admin"])
}

func TestAdminUser_RevokeSelfPrevented(t *testing.T) {
	db := setupIntegrationDB(t)
	gormDB := db.GormDB()

	adminID := testkit.CreateUser(t, gormDB, "admin@test.com", "admin")

	svc := adminservice.NewService(db)
	handler := NewUserHandler(svc)

	w := httptest.NewRecorder()
	c := createAdminIntegrationContext(w, adminID)
	c.Request = httptest.NewRequest("POST", "/users/"+itoa(adminID)+"/revoke-admin", nil)
	c.Params = gin.Params{{Key: "id", Value: itoa(adminID)}}

	handler.RevokeAdmin(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// itoa converts int64 to string — avoids importing strconv in test.
func itoa(n int64) string {
	return fmt.Sprintf("%d", n)
}
