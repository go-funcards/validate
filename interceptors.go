package validate

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ValidatorUnaryServerInterceptor(v *Validator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		err := reqValidation(v, req)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func DefaultValidatorUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return ValidatorUnaryServerInterceptor(Default)
}

func ValidatorStreamServerInterceptor(v *Validator) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &serverStream{ServerStream: stream, v: v})
	}
}

func DefaultValidatorStreamServerInterceptor() grpc.StreamServerInterceptor {
	return ValidatorStreamServerInterceptor(Default)
}

type serverStream struct {
	grpc.ServerStream
	v *Validator
}

func (s *serverStream) RecvMsg(m interface{}) error {
	err := reqValidation(s.v, m)
	if err != nil {
		return err
	}
	return s.ServerStream.RecvMsg(m)
}

func reqValidation(v *Validator, req any) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "Request struct is required")
	}

	err := v.ValidateStruct(req)
	if err != nil {
		switch tmp := err.(type) {
		case SliceValidateError:
			br := new(errdetails.BadRequest)
			for _, items := range tmp {
				if vErrors, ok := items.(validator.ValidationErrors); ok {
					addValidationErrors(br, vErrors)
				}
			}
			st, err1 := status.New(codes.InvalidArgument, tmp.Error()).WithDetails(br)
			if err1 != nil {
				panic(fmt.Sprintf("Unexpected error attaching metadata: %v", err1))
			}
			err = st.Err()
		case validator.ValidationErrors:
			br := new(errdetails.BadRequest)
			addValidationErrors(br, tmp)
			st, err1 := status.New(codes.InvalidArgument, tmp.Error()).WithDetails(br)
			if err1 != nil {
				panic(fmt.Sprintf("Unexpected error attaching metadata: %v", err1))
			}
			err = st.Err()
		}
	}
	return err
}

func addValidationErrors(br *errdetails.BadRequest, vErrors validator.ValidationErrors) {
	for _, ve := range vErrors {
		v := &errdetails.BadRequest_FieldViolation{
			Field:       ve.StructField(),
			Description: ve.Error(),
		}
		br.FieldViolations = append(br.FieldViolations, v)
	}
}
