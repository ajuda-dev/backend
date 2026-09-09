package validator

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
)

type CommunityValidator interface {
	ValidatorRegisterCommunity(community domain.CommunityDomain) *rest_err.RestErr
}

type communityValidator struct{}

func NewCommunityValidator() CommunityValidator {
	return &communityValidator{}
}

func (c *communityValidator) ValidatorRegisterCommunity(community domain.CommunityDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}
	if !isValidName(community.Name, true) {
		causes = append(causes, rest_err.Causes{
			Field:   "name",
			Message: "Name is not valid",
		})
	}
	if community.Description == "" {
		causes = append(causes, rest_err.Causes{
			Field:   "description",
			Message: "Description is not valid",
		})
	}
	if community.Address == (domain.AddressDomain{}) || !uuidv7.IsValidString(community.Address.Id) {
		causes = append(causes, rest_err.Causes{
			Field:   "addressId",
			Message: "AddressId is not valid",
		})
	}

	if community.Owner.Id == "" || !uuidv7.IsValidString(community.Owner.Id) {
		causes = append(causes, rest_err.Causes{
			Field:   "ownerId",
			Message: "OwnerId is not valid",
		})
	}

	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError(
			"Invalid community data",
			causes,
		)
	}
	return nil
}
