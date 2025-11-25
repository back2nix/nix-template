package grpc

import (
	"context"

	"chat/internal/application"
	// Предполагаем, что используется helloworld.proto как временный контракт
	// так как в дереве файлов нет chat.proto
	pb "chat/pkg/proto/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedGreeterServer
	// ИСПРАВЛЕНО: Ссылка на новый Command Handler
	postMessageHandler *application.PostMessageHandler
}

// NewServer создает gRPC сервер с внедренными зависимостями
// options (например, для трассировки) можно передать через grpc.ServerOption
func NewServer(postMessageHandler *application.PostMessageHandler, opts ...grpc.ServerOption) *grpc.Server {
	s := grpc.NewServer(opts...)
	srv := &Server{
		postMessageHandler: postMessageHandler,
	}
	pb.RegisterGreeterServer(s, srv)
	return s
}

// SayHello - реализует метод gRPC (пока что используем старый контракт Greeter).
// Мы адаптируем запрос "SayHello" в команду "PostMessage".
func (s *Server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "Name (content) is required")
	}

	// Адаптер: Превращаем DTO в Domain Command
	// Т.к. HelloRequest имеет только Name, используем его как Content.
	// AuthorID пока хардкодим как "grpc-user" или берем из метаданных (контекста), если нужно.
	cmd := application.PostMessageCommand{
		AuthorID: "grpc-user",
		Content:  req.Name,
	}

	// Вызов Application Layer (CQRS Command Side)
	resultMsg, err := s.postMessageHandler.Handle(ctx, cmd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to post message: %v", err)
	}

	return &pb.HelloReply{
		Message: resultMsg,
	}, nil
}
