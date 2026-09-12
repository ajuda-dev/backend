package validator

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
)

type CommunityValidator interface {
	ValidatorRegisterCommunity(community domain.CommunityDomain) *rest_err.RestErr
	ValidateUpdateCommunity(community domain.CommunityDomain) *rest_err.RestErr
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

func (c *communityValidator) ValidateUpdateCommunity(community domain.CommunityDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}

	name := strings.TrimSpace(community.Name)
	description := strings.TrimSpace(community.Description)
	addressId := strings.TrimSpace(community.Address.Id)

	if name == "" && description == "" && addressId == "" {
		causes = append(causes, rest_err.Causes{
			Field:   "body",
			Message: "provide at least one field to update",
		})
	}
	if name != "" && !isValidName(community.Name, true) {
		causes = append(causes, rest_err.Causes{
			Field:   "name",
			Message: "Name is not valid",
		})
	}
	if addressId != "" && !uuidv7.IsValidString(addressId) {
		causes = append(causes, rest_err.Causes{
			Field:   "address_id",
			Message: "AddressId is not valid",
		})
	}
	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError("Invalid community data", causes)
	}
	return nil
}
