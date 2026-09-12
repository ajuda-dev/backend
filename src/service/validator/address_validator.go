package validator

import (
	"strings"

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
			Field:   "address",
			Message: "pass at least zipCode or city, state and street to search for an address",
		})
	}
	if street := strings.TrimSpace(address.Street); street != "" && len(street) > 200 {
		causes = append(causes, rest_err.Causes{
			Field:   "street",
			Message: "street must have at most 200 characters",
		})
	}
	if number := strings.TrimSpace(address.Number); number != "" && len(number) > 20 {
		causes = append(causes, rest_err.Causes{
			Field:   "number",
			Message: "number must have at most 20 characters",
		})
	}
	if complement := strings.TrimSpace(address.Complement); complement != "" && len(complement) > 60 {
		causes = append(causes, rest_err.Causes{
			Field:   "complement",
			Message: "complement must have at most 60 characters",
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
