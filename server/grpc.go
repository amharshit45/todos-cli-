package server

import (
	"context"
	"errors"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/amharshit45/todos-cli-/gen/todopb"
	"github.com/amharshit45/todos-cli-/todo"
)

type Server struct {
	todopb.UnimplementedTodoServiceServer
	store todo.Storage
}

func New(store todo.Storage) *Server {
	return &Server{store: store}
}

func (s *Server) Add(ctx context.Context, req *todopb.AddRequest) (*todopb.AddResponse, error) {
	if err := s.store.Add(ctx, req.GetTitle(), req.GetDescription()); err != nil {
		return nil, domainToGRPCError(err)
	}
	return &todopb.AddResponse{}, nil
}

func (s *Server) List(ctx context.Context, _ *todopb.ListRequest) (*todopb.ListResponse, error) {
	todos, err := s.store.List(ctx)
	if err != nil {
		return nil, domainToGRPCError(err)
	}
	pbTodos := make([]*todopb.Todo, len(todos))
	for i, t := range todos {
		pbTodos[i] = &todopb.Todo{
			Id:          int32(t.ID),
			Title:       t.Title,
			Description: t.Description,
			Completed:   t.Completed,
		}
	}
	return &todopb.ListResponse{Todos: pbTodos}, nil
}

func (s *Server) Delete(ctx context.Context, req *todopb.DeleteRequest) (*todopb.DeleteResponse, error) {
	if err := s.store.Delete(ctx, int(req.GetId())); err != nil {
		return nil, domainToGRPCError(err)
	}
	return &todopb.DeleteResponse{}, nil
}

func (s *Server) SetCompleted(ctx context.Context, req *todopb.SetCompletedRequest) (*todopb.SetCompletedResponse, error) {
	if err := s.store.SetCompleted(ctx, int(req.GetId()), req.GetCompleted()); err != nil {
		return nil, domainToGRPCError(err)
	}
	return &todopb.SetCompletedResponse{}, nil
}

func (s *Server) EditTitle(ctx context.Context, req *todopb.EditTitleRequest) (*todopb.EditTitleResponse, error) {
	if err := s.store.EditTitle(ctx, int(req.GetId()), req.GetTitle()); err != nil {
		return nil, domainToGRPCError(err)
	}
	return &todopb.EditTitleResponse{}, nil
}

func (s *Server) EditDescription(ctx context.Context, req *todopb.EditDescriptionRequest) (*todopb.EditDescriptionResponse, error) {
	if err := s.store.EditDescription(ctx, int(req.GetId()), req.GetDescription()); err != nil {
		return nil, domainToGRPCError(err)
	}
	return &todopb.EditDescriptionResponse{}, nil
}

func domainToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	var code codes.Code
	switch {
	case errors.Is(err, todo.ErrNotFound):
		code = codes.NotFound
	case errors.Is(err, todo.ErrAlreadyCompleted),
		errors.Is(err, todo.ErrAlreadyIncomplete),
		errors.Is(err, todo.ErrTitleUnchanged),
		errors.Is(err, todo.ErrDescriptionUnchanged):
		code = codes.FailedPrecondition
	case errors.Is(err, todo.ErrInvalidID),
		errors.Is(err, todo.ErrEmptyTitle),
		errors.Is(err, todo.ErrTitleTooLong),
		errors.Is(err, todo.ErrDescriptionTooLong):
		code = codes.InvalidArgument
	default:
		log.Printf("internal error: %v", err)
		return status.Error(codes.Internal, "internal server error")
	}

	return status.Error(code, err.Error())
}
