package errors

import (
	"errors"
	"sort"

	"github.com/nenormalka/freya/types/errors/grpc"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	Unknown Code = iota + 1
	InvalidArgument
	NotFound
	AlreadyExists
	PermissionDenied
	Internal
	Unavailable
)

type (
	Error struct {
		Code    Code
		err     error
		Details map[string]string
	}

	Code int32
)

func NewInvalidError(err error) *Error {
	return NewError(InvalidArgument, err)
}

func NewNotFoundError(err error) *Error {
	return NewError(NotFound, err)
}

func NewAlreadyExistsError(err error) *Error {
	return NewError(AlreadyExists, err)
}

func NewPermissionDeniedError(err error) *Error {
	return NewError(PermissionDenied, err)
}

func NewInternalError(err error) *Error {
	return NewError(Internal, err)
}

func NewUnavailableError(err error) *Error {
	return NewError(Unavailable, err)
}

func NewUnknownError(err error) *Error {
	return NewError(Unknown, err)
}

func NewError(code Code, err error) *Error {
	return &Error{
		Code:    code,
		err:     err,
		Details: make(map[string]string),
	}
}

func (e Error) Error() string {
	if e.err == nil {
		return ""
	}

	return e.err.Error()
}

func (e *Error) SetDetails(details map[string]string) *Error {
	e.Details = details
	return e
}

func (e *Error) AddDetail(key, value string) *Error {
	e.Details[key] = value
	return e
}

func (e *Error) AddDetails(details map[string]string) *Error {
	if len(e.Details) == 0 {
		return e
	}

	for key, value := range details {
		e.Details[key] = value
	}

	return e
}

func (e *Error) DetailsToGRPCDetails() proto.Message {
	if e == nil {
		return nil
	}

	switch e.Code {
	case InvalidArgument:
		return e.detailsForInvalidArgument()
	default:
		return e.detailsDefault()
	}
}

func (e *Error) CodeToGRPCCode() codes.Code {
	if e == nil {
		return codes.Unknown
	}

	switch e.Code {
	case InvalidArgument:
		return codes.InvalidArgument
	case NotFound:
		return codes.NotFound
	case AlreadyExists:
		return codes.AlreadyExists
	case PermissionDenied:
		return codes.PermissionDenied
	case Internal:
		return codes.Internal
	case Unavailable:
		return codes.Unavailable
	default:
		return codes.Unknown
	}
}

func (e *Error) detailsDefault() proto.Message {
	if e == nil || len(e.Details) == 0 {
		return nil
	}

	return &grpc.ErrorInfo{Reason: e.Error(), Metadata: e.Details}
}

func (e *Error) detailsForInvalidArgument() proto.Message {
	if e == nil || len(e.Details) == 0 {
		return nil
	}

	fields := make([]*grpc.BadRequest_FieldViolation, 0, len(e.Details))
	for k, v := range e.Details {
		fields = append(fields, &grpc.BadRequest_FieldViolation{
			Field:       k,
			Description: v,
		})
	}

	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Field < fields[j].Field
	})

	return &grpc.BadRequest{FieldViolations: fields}
}

func ErrorToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	var e *Error
	if !errors.As(err, &e) || e == nil {
		return err
	}

	st := status.New(e.CodeToGRPCCode(), err.Error())

	details := e.DetailsToGRPCDetails()
	if details != nil {
		var errS error
		st, errS = st.WithDetails(details)
		if errS != nil {
			return err
		}
	}

	return st.Err()
}
