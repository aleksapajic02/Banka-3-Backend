package bank

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	bankpb "github.com/RAF-SI-2025/Banka-3-Backend/gen/bank"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type timeArgument struct{}

func (timeArgument) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func TestCreateAccountSuccess(t *testing.T) {
	notificationServer := &testNotificationServer{}
	notifAddr, notifStop := startNotificationTestServer(t, notificationServer)
	defer notifStop()
	t.Setenv("NOTIFICATION_GRPC_ADDR", notifAddr)

	userServer := &testUserServer{}
	userAddr, userStop := startUserTestServer(t, userServer)
	defer userStop()
	t.Setenv("USER_GRPC_ADDR", userAddr)

	server, mock, db := newGormTestServer(t)
	defer func() { _ = db.Close() }()

	createdAt := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	validUntil := time.Date(2029, 3, 19, 0, 0, 0, 0, time.UTC)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"user-email", "test@example.com",
		"employee-id", "3",
	))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM employees`)).
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(3)))

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "accounts"`)).
		WithArgs(
			sqlmock.AnyArg(),
			"checking-personal",
			int64(1),
			int64(100000),
			int64(3),
			timeArgument{},
			"EUR",
			"personal",
			"checking",
			int64(0),
			int64(500),
			int64(5000),
			nil, nil, nil, nil, nil,
		).
		WillReturnRows(sqlmockAccountRows().AddRow(
			int64(12),
			"12345678901234567890",
			"checking-personal",
			int64(1),
			int64(100000),
			int64(3),
			createdAt,
			validUntil,
			"EUR",
			true,
			"personal",
			"checking",
			int64(0),
			int64(500),
			int64(5000),
			int64(0),
			int64(0),
		))

	_, err := server.CreateAccount(ctx, &bankpb.CreateAccountRequest{
		ClientId:       1,
		Currency:       "EUR",
		Subtype:        "personal",
		AccountType:    "checking",
		InitialBalance: 1000,
		DailyLimit:     500,
		MonthlyLimit:   5000,
	})

	if err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestCreateAccountInvalidAccountType(t *testing.T) {
	server, _, db := newGormTestServer(t)
	defer func() { _ = db.Close() }()

	_, err := server.CreateAccount(context.Background(), &bankpb.CreateAccountRequest{
		ClientId:    1,
		Currency:    "EUR",
		Subtype:     "personal",
		AccountType: "invalid",
	})

	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestCreateAccountMissingMetadata(t *testing.T) {
	server, _, db := newGormTestServer(t)
	defer func() { _ = db.Close() }()

	_, err := server.CreateAccount(context.Background(), &bankpb.CreateAccountRequest{
		ClientId:    1,
		Currency:    "EUR",
		Subtype:     "personal",
		AccountType: "checking",
	})

	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestCreateAccountCreatorNotFound(t *testing.T) {
	server, mock, db := newGormTestServer(t)
	defer func() { _ = db.Close() }()

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"user-email", "test@example.com",
		"employee-id", "77",
	))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM employees`)).
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(77)))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "employees" WHERE "employees"."id" = $1 AND "employees"."deleted_at" IS NULL ORDER BY "employees"."id" LIMIT 1`)).
		WithArgs(int64(77)).
		WillReturnError(new(pgconn.PgError)) // Simulate not found

	_, err := server.CreateAccount(ctx, &bankpb.CreateAccountRequest{
		ClientId:    1,
		Currency:    "EUR",
		Subtype:     "personal",
		AccountType: "checking",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestCreateAccountCurrencyNotFound(t *testing.T) {
	server, mock, db := newGormTestServer(t)
	defer func() { _ = db.Close() }()

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"user-email", "test@example.com",
		"employee-id", "3",
	))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM employees`)).
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(3)))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "employees" WHERE "employees"."id" = $1 AND "employees"."deleted_at" IS NULL ORDER BY "employees"."id" LIMIT 1`)).
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "currencies" WHERE label = $1 AND "currencies"."deleted_at" IS NULL`)).
		WithArgs("USD").
		WillReturnError(new(pgconn.PgError)) // Simulate not found

	_, err := server.CreateAccount(ctx, &bankpb.CreateAccountRequest{
		ClientId:    1,
		Currency:    "USD",
		Subtype:     "personal",
		AccountType: "checking",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestCreateAccountDefaultValidUntilAndZeroLimitsBecomeNull(t *testing.T) {
	notificationServer := &testNotificationServer{}
	notifAddr, notifStop := startNotificationTestServer(t, notificationServer)
	defer notifStop()
	t.Setenv("NOTIFICATION_GRPC_ADDR", notifAddr)

	userServer := &testUserServer{}
	userAddr, userStop := startUserTestServer(t, userServer)
	defer userStop()
	t.Setenv("USER_GRPC_ADDR", userAddr)

	server, mock, db := newGormTestServer(t)
	defer func() { _ = db.Close() }()

	createdAt := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	validUntil := time.Date(2029, 3, 19, 0, 0, 0, 0, time.UTC)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"user-email", "test@example.com",
		"employee-id", "1",
	))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM employees`)).
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "accounts"`)).
		WithArgs(
			sqlmock.AnyArg(),
			"foreign-business",
			int64(1),
			int64(0),
			int64(1),
			timeArgument{},
			"EUR",
			"business",
			"foreign",
			int64(0),
			nil,
			nil,
			nil, nil, nil, nil, nil,
		).
		WillReturnRows(sqlmockAccountRows().AddRow(
			int64(13),
			"99999999999999999999",
			"foreign-business",
			int64(1),
			int64(0),
			int64(1),
			createdAt,
			validUntil,
			"EUR",
			true,
			"business",
			"foreign",
			int64(0),
			nil,
			nil,
			int64(0),
			int64(0),
		))

	_, err := server.CreateAccount(ctx, &bankpb.CreateAccountRequest{
		ClientId:    1,
		Currency:    "EUR",
		Subtype:     "business",
		AccountType: "foreign",
	})
	if err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestCreateAccountNumberCollisionRetryPath(t *testing.T) {
	notificationServer := &testNotificationServer{}
	notifAddr, notifStop := startNotificationTestServer(t, notificationServer)
	defer notifStop()
	t.Setenv("NOTIFICATION_GRPC_ADDR", notifAddr)

	userServer := &testUserServer{}
	userAddr, userStop := startUserTestServer(t, userServer)
	defer userStop()
	t.Setenv("USER_GRPC_ADDR", userAddr)

	server, mock, db := newGormTestServer(t)
	defer func() { _ = db.Close() }()

	createdAt := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	validUntil := time.Date(2030, 3, 19, 0, 0, 0, 0, time.UTC)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"user-email", "test@example.com",
		"employee-id", "1",
	))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM employees`)).
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "accounts"`)).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "accounts"`)).
		WillReturnRows(sqlmockAccountRows().AddRow(
			int64(99),
			"55555555555555555555",
			"checking-personal",
			int64(1),
			int64(0),
			int64(1),
			createdAt,
			validUntil,
			"EUR",
			true,
			"personal",
			"checking",
			int64(0),
			int64(0),
			int64(0),
			int64(0),
			int64(0),
		))

	_, err := server.CreateAccount(ctx, &bankpb.CreateAccountRequest{
		ClientId:    1,
		Currency:    "EUR",
		Subtype:     "personal",
		AccountType: "checking",
	})

	if err != nil {
		t.Fatalf("CreateAccount returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func sqlmockAccountRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"number",
		"name",
		"owner",
		"balance",
		"created_by",
		"created_at",
		"valid_until",
		"currency",
		"active",
		"owner_type",
		"account_type",
		"maintainance_cost",
		"daily_limit",
		"monthly_limit",
		"daily_expenditure",
		"monthly_expenditure",
	})
}
