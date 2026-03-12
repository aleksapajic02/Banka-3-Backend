package gateway

import (
	"banka-raf/gen/notification"
	"banka-raf/gen/user"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	userClient         user.UserServiceClient
	notificationClient notification.NotificationServiceClient
}

func connectToNotification() (*notification.NotificationServiceClient, error) {
	conn, err := grpc.NewClient("notification:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	notificationClient := notification.NewNotificationServiceClient(conn)
	return &notificationClient, nil
}

func connectToUser() (*user.UserServiceClient, error) {
	conn, err := grpc.NewClient("user:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	userClient := user.NewUserServiceClient(conn)
	return &userClient, nil
}

func NewServer() (*Server, error) {
	// TODO: replace with docker healthchecks for otherservices
	time.Sleep(time.Second * 3)

	userClient, err := connectToUser()
	if err != nil {
		return nil, err
	}

	notificationClient, err := connectToNotification()
	if err != nil {
		return nil, err
	}

	return &Server{userClient: *userClient, notificationClient: *notificationClient}, nil
}
