package main

import (
	"context"
	"ebs/src/db"
	"ebs/src/lib"
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

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/stripe/stripe-go/v82"
	"github.com/tidwall/gjson"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TestSuite struct {
	suite.Suite
	DB     *gorm.DB
	Mock   *sqlmock.Sqlmock
	Token  *string
	UserId *uint
	UID    *string
}

type TestUserModel struct {
	models.User
}

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
	db := db.GetDb()
	err = db.
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

func deleteAllTables() {
	db := db.GetDb()
	db.Exec(`
	TRUNCATE users CASCADE;
	TRUNCATE organizations CASCADE;
	TRUNCATE teams CASCADE;
	TRUNCATE team_members CASCADE;
	TRUNCATE events CASCADE;
	TRUNCATE tickets CASCADE;
	TRUNCATE bookings CASCADE;
	TRUNCATE reservations CASCADE;
	TRUNCATE transactions CASCADE;
	TRUNCATE event_subscriptions CASCADE;
	TRUNCATE admissions CASCADE;
	TRUNCATE job_tasks CASCADE;
	`)
}

type MockFirebaseApp struct {
	*firebase.App
}

type MockFirebaseAuth struct {
	*auth.Client
}

type MockUser models.User

func (c *MockFirebaseAuth) CreateUser(ctx context.Context, user *auth.UserToCreate) (*auth.UserRecord, error) {
	newUser := &auth.UserRecord{
		EmailVerified: true,
		UserInfo: &auth.UserInfo{
			DisplayName: "test user",
			Email:       "user@test.local",
			PhoneNumber: "",
			PhotoURL:    "",
			ProviderID:  "",
			UID:         uuid.NewString(),
		},
	}
	return newUser, nil
}

func (c MockFirebaseAuth) GetUserByEmail(ctx context.Context, email string) (*auth.UserRecord, error) {
	newUser := &auth.UserRecord{
		EmailVerified: true,
		UserInfo: &auth.UserInfo{
			DisplayName: "test user",
			Email:       email,
			PhoneNumber: "",
			PhotoURL:    "",
			ProviderID:  "",
			UID:         uuid.NewString(),
		},
	}
	return newUser, nil
}

func (s *TestSuite) SetupSuite() {
	os.Setenv("API_SECRET", "secret")
	os.Setenv("JWT_SECRET", "secret")
	os.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "127.0.0.1:9099")

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("bookabledate", eventDateTimeValidatorFunc)
		v.RegisterValidation("gtdate", gtfield)
		v.RegisterValidation("ltdate", ltfield)
		v.RegisterValidation("betweenfields", betweenfields)
	}

	d, mock := NewMockDB()
	db.NewDB(d)
	s.DB = d

	err := d.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.Team{},
		&models.TeamMember{},
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
	if err = d.Exec(`
	CREATE OR REPLACE FUNCTION set_tenant(tenant_id text) RETURNS void AS $$
	BEGIN
		PERFORM set_config('app.current_tenant', tenant_id, false);
	END;
	$$ LANGUAGE plpgsql;
	`).Error; err != nil {
		log.Printf("Error creating FUNCTION set_tenant: %s\n", err.Error())
	}

	// Setup Mock Stripe API
	sc := stripe.NewClient("sk_test_123", stripe.WithBackends(
		&stripe.Backends{
			API: stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
				URL: stripe.String("http://localhost:12111/v1"),
			}),
			Connect: stripe.GetBackendWithConfig(stripe.ConnectBackend, &stripe.BackendConfig{
				URL: stripe.String("http://localhost:12111/v1"),
			}),
		},
	))
	lib.NewStripeClient(sc)

	app, _ := firebase.NewApp(context.Background(), &firebase.Config{
		ProjectID: "projectId",
	})
	lib.NewFirebaseApp(app)

	s.Mock = &mock
}

func createFirebaseUser(s *TestSuite, email string) (*string, error) {
	fauth, _ := lib.GetFirebaseAuth()
	nuser := new(auth.UserToCreate)
	nuser.
		Email(email).
		DisplayName(email)
	au, err := fauth.GetUserByEmail(context.Background(), email)
	if err != nil {
		log.Print(err)
		if !strings.HasPrefix(err.Error(), "no user exists") {
			return nil, err
		}
	}
	if au != nil {
		return &au.UID, nil
	}
	cuser, err := fauth.CreateUser(context.Background(), nuser)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	log.Printf("Created FirebaseUser [%s] with ID: %s\n", email, cuser.UID)
	s.UID = &cuser.UID
	return &cuser.UID, nil
}

func createUser(s *TestSuite, email string, uid string) (*models.User, error) {
	user := models.User{
		Email: email,
		Name:  email,
		UID:   uid,
	}
	org := models.Organization{
		Name:         "test",
		ContactEmail: email,
	}

	db := db.GetDb()
	ss := db.Session(&gorm.Session{
		SkipHooks: true,
	})
	if err := ss.Transaction(func(tx *gorm.DB) error {
		if err := tx.FirstOrCreate(&user).Error; err != nil {
			return err
		}
		org.OwnerID = user.ID
		if err := tx.FirstOrCreate(&org).Error; err != nil {
			return err
		}
		user.ActiveOrg = org.ID
		if err := tx.Where(&user).Updates(&user).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Fatalf("Could not create user due to error: %s\n", err.Error())
	}
	log.Printf("Created user with ID: %d, %s", user.ID, user.Email)
	s.UserId = &user.ID

	return &user, nil
}

func deleteUser(s *TestSuite, email string) error {
	db := db.GetDb()
	ss := db.Session(&gorm.Session{
		AllowGlobalUpdate: true,
		SkipHooks:         true,
	})
	return ss.Transaction(func(tx *gorm.DB) error {
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.
				Unscoped().
				Select(clause.Associations).
				Where("email = ?", email).
				Delete(&models.User{Email: email}).
				Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			log.Printf("Could not delete user [%s] from database: %s\n", email, err.Error())
			return err
		}
		return nil
	})
}

func deleteFirebaseUser(email string) error {
	fauth, _ := lib.GetFirebaseAuth()
	user, err := fauth.GetUserByEmail(context.Background(), email)
	if err != nil {
		return err
	}
	return fauth.DeleteUser(context.Background(), user.UID)
}

func deleteTestUser(s *TestSuite, email string, fuser bool) error {
	if fuser {
		return deleteFirebaseUser(email)
	}
	if err := deleteUser(s, email); err != nil {
		log.Printf("Could not delete user [%s] from database: %s\n", email, err.Error())
		return err
	}
	return nil
}

func (s *TestSuite) TearDownSuite() {
	deleteAllTables()
	os.Unsetenv("API_SECRET")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("FIREBASE_AUTH_EMULATOR_HOST")
}

func (s *TestSuite) SetupTest() {
	f, _ := createFirebaseUser(s, email)
	createUser(s, email, *f)
}

func (s *TestSuite) TearDownTest() {
	deleteTestUser(s, email, true)
}

func NewMockDB() (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}

	testdb := "postgresql://postgres:password@localhost:5432/testdb?sslmode=disable"
	gormDB, err := gorm.Open(postgres.Open(testdb), &gorm.Config{
		ConnPool:                                 db,
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
		PropagateUnscoped:                        true,
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

const (
	email = "someone@company.test"
)

func (s *TestSuite) TestUserNotFound() {
	deleteTestUser(s, email, true)

	router := setupRouter()
	guestAuthRoutes(router)
	jbody := map[string]any{
		"email": email,
	}
	w := httptest.NewRecorder()

	sbody, _ := json.Marshal(&jbody)
	loginReq, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(string(sbody)))
	loginReq.Header.Set("x-secret", secret)
	loginReq.Header.Set("origin", origin)
	router.ServeHTTP(w, loginReq)
	assert.Equal(s.T(), 404, w.Code)
}

func (s *TestSuite) TestLogin() {
	router := setupRouter()
	guestAuthRoutes(router)

	w := httptest.NewRecorder()
	jbody := map[string]any{
		"email": email,
	}
	sbody, _ := json.Marshal(&jbody)
	loginReq, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(string(sbody)))
	loginReq.Header.Set("x-secret", secret)
	loginReq.Header.Set("origin", origin)
	router.ServeHTTP(w, loginReq)

	assert.Equal(s.T(), 200, w.Code)

	rbytes, err := io.ReadAll(w.Body)
	assert.Nil(s.T(), err)
	assert.Greaterf(s.T(), len(rbytes), 0, "Empty response")
	var response struct {
		Token *string `json:"token"`
	}
	err = json.Unmarshal(rbytes, &response)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), response.Token)
	s.Token = response.Token
}

func (s *TestSuite) TestRegisterUser() {
	deleteUser(s, email)

	router := setupRouter()
	guestAuthRoutes(router)

	w := httptest.NewRecorder()

	jbody := map[string]any{
		"email": email,
	}
	sbody, _ := json.Marshal(&jbody)

	registerReq, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(string(sbody)))
	registerReq.Header.Set("x-secret", secret)
	registerReq.Header.Set("origin", origin)
	router.ServeHTTP(w, registerReq)

	assert.Equal(s.T(), 200, w.Code)
}

func (s *TestSuite) TestRegisterAnotherUser() {
	email := "anotheruser@company.test"
	_, err := createFirebaseUser(s, email)
	assert.Nil(s.T(), err)
	defer deleteTestUser(s, email, true)

	router := setupRouter()
	guestAuthRoutes(router)

	w := httptest.NewRecorder()

	jbody := map[string]any{
		"email": email,
	}
	sbody, _ := json.Marshal(&jbody)

	registerReq, _ := http.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(string(sbody)))
	registerReq.Header.Set("x-secret", secret)
	registerReq.Header.Set("origin", origin)
	router.ServeHTTP(w, registerReq)
	bres, _ := io.ReadAll(w.Body)
	sres := string(bres)
	errString := gjson.Get(sres, "error").String()
	assert.Equal(s.T(), errString, "")
	assert.Equal(s.T(), 200, w.Code)
}

func (s *TestSuite) TestEvents() {
	token, err := generateJWT(email, *s.UserId, 1)
	assert.Nil(s.T(), err)

	router := setupRouter()
	apiv1 := apiv1Group(router)
	apiv1.Use(authMiddleware)
	eventHandlers(apiv1)

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
