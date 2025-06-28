package validator

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
)

type AddressValidator interface {
	ValidatorSearchAddress(address domain.AddressDomain) *rest_err.RestErr
}

type addressValidator struct{}



func NewAddressValidator() AddressValidator {
	return &addressValidator{}
}


func (a *addressValidator) ValidatorSearchAddress(address domain.AddressDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}

	if address.ZipCode == "" && address.City == "" && address.State == "" && address.Street == "" {
		causes = append(causes, rest_err.Causes{
			Field:   "invalid search address data",
			Message: "pass at least zipCode or city, state and street to search for an address",
		})
	}
	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError(
			"Invalid user data",
			causes,
		)
	}
	return nil
}
