package grpc

import (
	grpc_user "GolangTemplateProject/internal/api-grpc/user"
	"google.golang.org/grpc"
)

type Service struct {
	grpc_user.UnimplementedUserServer
}

func Registry(server *grpc.Server) {
	grpc_user.RegisterUserServer(server, newService())
}

func newService() *Service {
	return &Service{}
}
