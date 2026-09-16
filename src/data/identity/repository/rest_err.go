package repository

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
)

func toRestErr(err error) *rest_err.RestErr {
	if err == nil {
		return nil
	}
	if restErr, ok := err.(*rest_err.RestErr); ok {
		return restErr
	}
	return rest_err.NewInternalServerError(err.Error())
}
