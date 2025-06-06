package main

import (
	"ebs/src/db"
	"ebs/src/models"
	"ebs/src/types"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type TestSuite struct {
	suite.Suite
	DB    *gorm.DB
	Mock  *sqlmock.Sqlmock
	Token *string
}

type TestUserModel struct {
	models.User
}

var dbi *gorm.DB

func authMiddleware(ctx *gin.Context) {
	bearerToken := ctx.Request.Header.Get("Authorization")
	if !strings.HasPrefix(bearerToken, "Bearer") {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	reqToken := strings.Split(bearerToken, " ")[1]
	if reqToken == "" {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	claims := &types.Claims{}
	tkn, err := jwt.ParseWithClaims(reqToken, claims, func(t *jwt.Token) (any, error) {
		return jwtKey, nil
	})
	if err != nil {
		log.Printf("token error: %s\n", err.Error())
		if err == jwt.ErrSignatureInvalid || err == jwt.ErrTokenMalformed {
			ctx.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized"))
			return
		}
		ctx.AbortWithError(http.StatusUnauthorized, err)
		return
	}
	if !tkn.Valid {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var user models.User
	uid, err := strconv.Atoi(claims.Subject)
	if err != nil {
		log.Println("error parsing claims:", err.Error())
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	err = dbi.
		Model(&models.User{}).
		Where(&models.User{ID: uint(uid)}).
		First(&user).
		Error
	if err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	log.Printf("Retrieved user details: %d,%s,%s,%s\n", user.ID, user.Email, user.UID, user.TenantID.String())

	ctx.Set("id", user.ID)
	ctx.Set("tenant_id", user.TenantID.String())
	ctx.Set("email", user.Email)
	ctx.Set("uid", user.UID)
	ctx.Set("org", user.ActiveOrg)
	ctx.Set("role", user.Role)
	ctx.Set("perms", claims)
}

func (s *TestSuite) SetupSuite() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("bookabledate", eventDateTimeValidatorFunc)
		v.RegisterValidation("gtdate", gtfield)
		v.RegisterValidation("ltdate", ltfield)
		v.RegisterValidation("betweenfields", betweenfields)
	}

	d, mock := NewMockDB()
	db.NewDB(d)
	s.DB = d
	dbi = d

	err := dbi.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.Event{},
		&models.Ticket{},
		&models.Booking{},
		&models.Reservation{},
		&models.Admission{},
		&models.Transaction{},
		&models.EventSubscription{},
		&models.JobTask{},
	)
	if err != nil {
		log.Fatalf("error migration: %s", err.Error())
	}
	if err = dbi.Exec(`
	CREATE OR REPLACE FUNCTION set_tenant(tenant_id text) RETURNS void AS $$
	BEGIN
		PERFORM set_config('app.current_tenant', tenant_id, false);
	END;
	$$ LANGUAGE plpgsql;
	`).Error; err != nil {
		log.Printf("Error creating FUNCTION set_tenant: %s\n", err.Error())
	}

	s.Mock = &mock
	tenantId := uuid.New()
	uid := uuid.New()
	user := models.User{
		Email:     "someone@example.com",
		UID:       uid.String(),
		Name:      "Test User",
		ActiveOrg: 1,
		TenantID:  &tenantId,
	}

	if err := d.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Fatalf("Could not create user due to error: %s\n", err.Error())
	}
	log.Printf("Created user with ID: %d, %s", user.ID, user.Email)
	token, err := generateJWT(user.Email, user.ID, user.ActiveOrg)
	if err != nil {
		log.Fatalf("Error generating JWT token: %s\n", err.Error())
		return
	}
	s.Token = &token
}

func (s *TestSuite) TearDownSuite() {
	inner, err := s.DB.DB()
	if err != nil {
		log.Printf("Error accessing inner db instance: %s\n", err.Error())
		return
	}
	inner.Exec(`
	DELETE FROM event_subscriptions WHERE true;
	DELETE FROM job_tasks WHERE true;
	DELETE FROM admissions WHERE true;
	DELETE FROM transactions WHERE true;
	DELETE FROM reservations WHERE true;
	DELETE FROM bookings WHERE true;
	DELETE FROM tickets WHERE true;
	DELETE FROM events WHERE true;
	DELETE FROM organizations WHERE true;
	DELETE FROM users WHERE true;
	`)
	inner.Close()
}

func (s *TestSuite) SetupTest() {
}

func (s *TestSuite) TestDownTest() {

}

func NewMockDB() (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}

	testdb := "postgresql://postgres:password@localhost:5432/testdb?sslmode=disable"
	gormDB, err := gorm.Open(postgres.Open(testdb), &gorm.Config{
		ConnPool: db,
	})
	if err != nil {
		log.Fatalf("An error '%s' was not expected when opening gorm database", err)
	}

	return gormDB, mock
}

const (
	secret = "secret"
	origin = "http://localhost:3000"
)

func (s *TestSuite) TestPingRoute() {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), 200, w.Code)
}

func (s *TestSuite) TestMaintenanceMode() {
	os.Setenv("MAINTENANCE_MODE", "true")

	router := setupRouter()
	router = maintenanceModeMiddleware(router)
	apiv1Group(router)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), 503, w.Code)
}

func (s *TestSuite) TestAuthRoutes() {
	router := setupRouter()
	guestAuthRoutes(router)

	w := httptest.NewRecorder()

	jbody := map[string]any{
		"email": "someone@example.com",
	}
	sbody, _ := json.Marshal(&jbody)
	loginReq, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(string(sbody)))
	loginReq.Header.Set("x-secret", secret)
	loginReq.Header.Set("origin", origin)
	router.ServeHTTP(w, loginReq)

	assert.Equal(s.T(), 404, w.Code)
	rbytes, err := io.ReadAll(w.Body)
	assert.Nil(s.T(), err)
	assert.Greaterf(s.T(), len(rbytes), 0, "Empty response")

	w = httptest.NewRecorder()

	registerReq, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(string(sbody)))
	registerReq.Header.Set("x-secret", secret)
	registerReq.Header.Set("origin", origin)
	router.ServeHTTP(w, registerReq)

	assert.Equal(s.T(), 400, w.Code)
}

func (s *TestSuite) TestEvents() {
	router := setupRouter()
	apiv1 := apiv1Group(router)
	apiv1.Use(authMiddleware)
	eventHandlers(apiv1)

	token := *s.Token
	s.Run("Should return list of Event with 200 status", func() {
		w := httptest.NewRecorder()
		listReq, _ := http.NewRequest("GET", "/api/v1/events", nil)
		listReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		listReq.Header.Set("x-secret", secret)
		listReq.Header.Set("origin", origin)
		router.ServeHTTP(w, listReq)
		resbytes, err := io.ReadAll(w.Body)
		if err == nil {
			var resmap map[string]any
			if err := json.Unmarshal(resbytes, &resmap); err != nil {
				log.Printf("Error decoding response: %s\n", err.Error())
			} else {
				assert.NotNil(s.T(), resmap, "Response body returned nil")
			}
		} else {
			log.Printf("Error reading response body: %s\n", err.Error())
		}

		assert.Equal(s.T(), 200, w.Code)
	})

	s.Run("Should return a 400 error response", func() {
		w := httptest.NewRecorder()
		reqBody := types.CreateEventRequestBody{
			Title: "test event",
		}
		rbytes, err := json.Marshal(&reqBody)
		assert.Nil(s.T(), err)
		eventReq, err := http.NewRequest("POST", "/api/v1/events", strings.NewReader(string(rbytes)))
		assert.Nil(s.T(), err)
		eventReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		eventReq.Header.Set("x-secret", secret)
		eventReq.Header.Set("origin", origin)
		router.ServeHTTP(w, eventReq)

		assert.Equal(s.T(), 400, w.Code)

		rbytes, err = io.ReadAll(w.Body)
		assert.Nil(s.T(), err)
		sjson := string(rbytes)
		errMsg := gjson.Get(sjson, "error").String()

		assert.NotNil(s.T(), errMsg)
	})
}

func TestRunner(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
