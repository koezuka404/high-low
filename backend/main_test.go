package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"backend/model"
	"backend/usecase"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMain_SuccessPath(t *testing.T) {
	origRun := runFn
	origFatal := fatalFn
	defer func() {
		runFn = origRun
		fatalFn = origFatal
	}()

	runFn = func() error { return nil }
	fatalCalled := false
	fatalFn = func(err error) { fatalCalled = true }

	main()
	if fatalCalled {
		t.Fatal("fatal should not be called")
	}
}

func TestMain_FatalOnError(t *testing.T) {
	origRun := runFn
	origFatal := fatalFn
	origLogErr := logErrFn
	origExit := exitFn
	defer func() {
		runFn = origRun
		fatalFn = origFatal
		logErrFn = origLogErr
		exitFn = origExit
	}()

	runFn = func() error { return errors.New("boom") }
	var got error
	logErrFn = func(err error) { got = err }
	exitCode := 0
	exitFn = func(code int) { exitCode = code }

	main()
	if got == nil || got.Error() != "boom" {
		t.Fatalf("unexpected error: %v", got)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestDefaultLogAndExitHelpers_CanBeCovered(t *testing.T) {
	origLogErr := logErrFn
	origExit := exitFn
	origOsExit := osExitImpl
	defer func() {
		logErrFn = origLogErr
		exitFn = origExit
		osExitImpl = origOsExit
	}()

	logErrDefault(errors.New("x"))

	called := 0
	code := 0
	osExitImpl = func(c int) { called++; code = c }
	exitDefault(7)
	if called != 1 || code != 7 {
		t.Fatalf("unexpected exitDefault call: called=%d code=%d", called, code)
	}
}

func TestRunWithDeps_DBError(t *testing.T) {
	errBoom := errors.New("db error")
	err := runWithDeps(runDeps{
		getenv: func(string) string { return "" },
		newDB: func() (*gorm.DB, error) { return nil, errBoom },
		autoMigrate: func(db *gorm.DB) error {
			t.Fatal("autoMigrate should not be called")
			return nil
		},
		newRedis: func() (redis.UniversalClient, error) { return nil, nil },
		start:    func(e *echo.Echo) error { return nil },
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestRunWithDeps_MigrateError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	errBoom := errors.New("migrate error")
	err = runWithDeps(runDeps{
		getenv: func(string) string { return "" },
		newDB:  func() (*gorm.DB, error) { return db, nil },
		autoMigrate: func(db *gorm.DB) error {
			return errBoom
		},
		newRedis: func() (redis.UniversalClient, error) { return nil, nil },
		start:    func(e *echo.Echo) error { return nil },
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestRunWithDeps_Success_ParsesRateLimitEnv(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Game{}, &model.GameRoundLog{}); err != nil {
		t.Fatal(err)
	}

	env := map[string]string{
		"RATE_LIMIT_CAPACITY":    "30",
		"RATE_LIMIT_REFILL_RATE": "6.5",
		"RATE_LIMIT_TOKEN_COST":  "2",
		"RATE_LIMIT_TTL_SEC":     "120",
	}
	var gotParams *struct {
		capacity   float64
		refillRate float64
		tokenCost  float64
		ttlSec     int64
	}

	err = runWithDeps(runDeps{
		getenv: func(k string) string { return env[k] },
		newDB:  func() (*gorm.DB, error) { return db, nil },
		autoMigrate: func(db *gorm.DB) error {
			return nil
		},
		newRedis: func() (redis.UniversalClient, error) { return nil, nil },
		start:    func(e *echo.Echo) error { return nil },
		onRateLimitParams: func(p usecase.RateLimitParams) {
			gotParams = &struct {
				capacity   float64
				refillRate float64
				tokenCost  float64
				ttlSec     int64
			}{p.Capacity, p.RefillRate, p.TokenCost, p.TTLSec}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotParams == nil {
		t.Fatal("expected params hook called")
	}
	if gotParams.capacity != 30 || gotParams.refillRate != 6.5 || gotParams.tokenCost != 2 || gotParams.ttlSec != 120 {
		t.Fatalf("unexpected params: %+v", *gotParams)
	}
}

func TestRunWithDeps_IgnoresInvalidRateLimitEnv(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Game{}, &model.GameRoundLog{}); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{
		"RATE_LIMIT_CAPACITY":    "x",
		"RATE_LIMIT_REFILL_RATE": "-1",
		"RATE_LIMIT_TOKEN_COST":  "0",
		"RATE_LIMIT_TTL_SEC":     "bad",
	}
	var got usecase.RateLimitParams
	err = runWithDeps(runDeps{
		getenv: func(k string) string { return env[k] },
		newDB:  func() (*gorm.DB, error) { return db, nil },
		autoMigrate: func(db *gorm.DB) error {
			return nil
		},
		newRedis: func() (redis.UniversalClient, error) { return nil, nil },
		start:    func(e *echo.Echo) error { return nil },
		onRateLimitParams: func(p usecase.RateLimitParams) {
			got = p
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Capacity != 20 || got.RefillRate != 5 || got.TokenCost != 1 || got.TTLSec != 60 {
		t.Fatalf("unexpected defaults: %+v", got)
	}
}

func TestRunWithDeps_StartErrorPropagates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Game{}, &model.GameRoundLog{}); err != nil {
		t.Fatal(err)
	}
	errBoom := errors.New("start error")
	err = runWithDeps(runDeps{
		getenv:     func(string) string { return "" },
		newDB:      func() (*gorm.DB, error) { return db, nil },
		autoMigrate: func(db *gorm.DB) error { return nil },
		newRedis:   func() (redis.UniversalClient, error) { return nil, nil },
		start:      func(e *echo.Echo) error { return errBoom },
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestDefaultRunDeps_HasNowStart(t *testing.T) {
	origGetenv := getenvFn
	origNewDB := newDBFn
	origMigrate := autoMigrateFn
	origNewRedis := newRedisFn
	origStart := startFn
	defer func() {
		getenvFn = origGetenv
		newDBFn = origNewDB
		autoMigrateFn = origMigrate
		newRedisFn = origNewRedis
		startFn = origStart
	}()

	getenvFn = func(key string) string {
		if key == "X" {
			return "Y"
		}
		return ""
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	newDBFn = func() (*gorm.DB, error) { return db, nil }
	migrated := false
	autoMigrateFn = func(db *gorm.DB) error { migrated = true; return nil }
	newRedisFn = func() (redis.UniversalClient, error) { return nil, nil }
	started := false
	startFn = func(e *echo.Echo) error { started = true; return nil }

	d := defaultRunDeps()
	if d.getenv("X") != "Y" {
		t.Fatalf("unexpected getenv result")
	}
	gotDB, err := d.newDB()
	if err != nil || gotDB == nil {
		t.Fatalf("unexpected newDB: %v %v", gotDB, err)
	}
	if err := d.autoMigrate(gotDB); err != nil {
		t.Fatal(err)
	}
	if !migrated {
		t.Fatal("expected autoMigrate called")
	}
	if _, err := d.newRedis(); err != nil {
		t.Fatal(err)
	}
	if err := d.start(echo.New()); err != nil {
		t.Fatal(err)
	}
	if !started {
		t.Fatal("expected start called")
	}

	_ = time.Now()
}

func TestDefaultImplementations_AutoMigrateAndStart(t *testing.T) {
	origStartAddr := startAddrFn
	defer func() { startAddrFn = origStartAddr }()

	startAddrFn = func() string { return "127.0.0.1:0" }
	e := echo.New()

	done := make(chan error, 1)
	go func() { done <- startFn(e) }()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = e.Shutdown(ctx)

	err := <-done
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("unexpected startFn err: %v", err)
	}

	db, err2 := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err2 != nil {
		t.Fatal(err2)
	}
	if err2 := autoMigrateFn(db); err2 != nil {
		t.Fatalf("unexpected autoMigrateFn err: %v", err2)
	}
}

func TestDefaultTopLevelFuncBodies_AreCovered(t *testing.T) {
	origGetenv := getenvFn
	origNewDB := newDBFn
	origMigrate := autoMigrateFn
	origNewRedis := newRedisFn
	origStartAddr := startAddrFn
	origStart := startFn
	defer func() {
		getenvFn = origGetenv
		newDBFn = origNewDB
		autoMigrateFn = origMigrate
		newRedisFn = origNewRedis
		startAddrFn = origStartAddr
		startFn = origStart
	}()

	if startAddrFn() != ":8080" {
		t.Fatalf("unexpected default addr: %q", startAddrFn())
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	getenvFn = func(string) string { return "" }
	newDBFn = func() (*gorm.DB, error) { return db, nil }
	autoMigrateFn = func(db *gorm.DB) error { return nil }
	newRedisFn = func() (redis.UniversalClient, error) { return nil, nil }
	startFn = func(e *echo.Echo) error { return nil }

	if err := runFn(); err != nil {
		t.Fatalf("unexpected runFn err: %v", err)
	}
}

