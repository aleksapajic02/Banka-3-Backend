package user

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/pquerna/otp/totp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	notificationpb "github.com/RAF-SI-2025/Banka-3-Backend/gen/notification"
	userpb "github.com/RAF-SI-2025/Banka-3-Backend/gen/user"
)

type TOTPServer struct {
	userpb.UnimplementedTOTPServiceServer
	db                  *sql.DB
	notificationService notificationpb.NotificationServiceClient
}

func NewTotpServer(database *sql.DB, notif notificationpb.NotificationServiceClient) *TOTPServer {
	return &TOTPServer{
		db:                  database,
		notificationService: notif,
	}
}

func (s *TOTPServer) VerifyCode(_ context.Context, req *userpb.VerifyCodeRequest) (*userpb.VerifyCodeResponse, error) {
	client, err := getUserByAttribute(Client{}, s, "email", req.Email)
	userId = client.Id
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	userId := client.Id

	secret, err := s.GetSecret(userId)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.Unauthenticated, "user doesn't have TOTP set up")
		}
		return nil, err
	}
	valid, err := totp.ValidateCustom(req.Code, *secret, time.Now(), totp.ValidateOpts{
		Digits: 6,
		Period: 30,
		Skew:   1,
	})
	if err != nil {
		return nil, err
	}
	if !valid {
		passed, err := s.tryBurnBackupCode(*userId, req.Code)
		if err != nil {
			return nil, err
		}
		return &userpb.VerifyCodeResponse{
			Valid: *passed,
		}, nil
	}
	return &userpb.VerifyCodeResponse{Valid: valid}, nil
}
func (s *Server) EnrollBegin(_ context.Context, req *userpb.EnrollBeginRequest) (*userpb.EnrollBeginResponse, error) {
	client, err := getUserByAttribute(Client{}, s, "email", req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	active, err := s.status(tx, *userId)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, err
	}
	if *active {
		return nil, status.Error(20, "totp already enabled")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Banka3",
		AccountName: req.Email,
	})
	userId := client.Id

	if err != nil {
		return nil, err
	}

	secret := key.Secret()

	err = s.SetTempTOTPSecret(tx, *userId, secret)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return &userpb.EnrollBeginResponse{
		Url: key.URL(),
	}, nil
}

func generateBackupCodes(num uint64) (*[]string, error) {
	var codes []string
	for _ = range num {
		random, err := rand.Int(rand.Reader, big.NewInt(999999))
		if err != nil {
			return nil, err
		}
		code := fmt.Sprintf("%0*d", 6, random)
		codes = append(codes, code)
	}
	return &codes, nil
}

func (s *TOTPServer) EnrollConfirm(_ context.Context, req *userpb.EnrollConfirmRequest) (*userpb.EnrollConfirmResponse, error) {
	client, err := getUserByAttribute(Client{}, s, "email", req.Email)
	userId := client.Id
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	userId := client.Id

	tx, err := s.database.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	tempSecret, err := s.GetTempSecret(tx, userId)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}

	valid := totp.Validate(req.Code, *tempSecret)

	if !valid {
		return &userpb.EnrollConfirmResponse{
			Success: false,
		}, nil
	}

	err = s.EnableTOTP(tx, userId, *tempSecret)
	if err != nil {
		return nil, err
	}

	backupCodes, err := generateBackupCodes(5)
	if err != nil {
		return nil, err
	}

	err = s.InsertGeneratedCodes(tx, *userId, *backupCodes)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return &userpb.EnrollConfirmResponse{
		Success:     true,
		BackupCodes: *backupCodes,
	}, nil
}

func (s *TOTPServer) Status(_ context.Context, req *userpb.StatusRequest) (*userpb.StatusResponse, error) {
	userId, err := s.getUserIdByEmail(req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	active, err := s.status(tx, *userId)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return &userpb.StatusResponse{
		Active: *active,
	}, nil
}

func (s *Server) TOTPStatus(_ context.Context, req *userpb.TOTPStatusRequest) (*userpb.TOTPStatusResponse, error) {
	client, err := getUserByAttribute(Client{}, s, "email", req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	userId := client.Id
	active, err := s.totpStatus(userId)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) || errors.Is(err, sql.ErrNoRows) {
			return &userpb.TOTPStatusResponse{
				Active: false,
			}, nil
		}
		return nil, err
	}
	return &userpb.TOTPStatusResponse{
		Active: *active,
	}, nil
}
