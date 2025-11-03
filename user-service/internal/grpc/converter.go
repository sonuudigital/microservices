package grpc

import (
	"strconv"
	"time"

	userv1 "github.com/sonuudigital/microservices/gen/user/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CachedUser struct {
	*userv1.User
	HashedPassword string
}

func userToMap(u *userv1.User, hashedPassword string) map[string]any {
	return map[string]any{
		"id":             u.Id,
		"username":       u.Username,
		"hashedPassword": hashedPassword,
		"email":          u.Email,
		"createdAt":      u.CreatedAt.AsTime().Unix(),
		"updatedAt":      u.UpdatedAt.AsTime().Unix(),
	}
}

func mapToUser(data map[string]string) (*CachedUser, error) {
	createdAt, err := strconv.ParseInt(data["createdAt"], 10, 64)
	if err != nil {
		return nil, err
	}

	updatedAt, err := strconv.ParseInt(data["updatedAt"], 10, 64)
	if err != nil {
		return nil, err
	}

	return &CachedUser{
		User: &userv1.User{
			Id:        data["id"],
			Username:  data["username"],
			Email:     data["email"],
			CreatedAt: timestamppb.New(time.Unix(createdAt, 0)),
			UpdatedAt: timestamppb.New(time.Unix(updatedAt, 0)),
		},
		HashedPassword: data["hashedPassword"],
	}, nil
}
