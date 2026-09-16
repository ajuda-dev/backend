package validator

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	addressdomain "github.com/ajuda-dev/backend/src/service/address/domain"
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
	"github.com/ajuda-dev/backend/src/service/validation"
	"github.com/samborkent/uuidv7"
)

type CommunityValidator interface {
	ValidatorRegisterCommunity(community communitydomain.CommunityDomain) *rest_err.RestErr
	ValidateUpdateCommunity(community communitydomain.CommunityDomain) *rest_err.RestErr
}

type communityValidator struct{}

func NewCommunityValidator() CommunityValidator {
	return &communityValidator{}
}

func (c *communityValidator) ValidatorRegisterCommunity(community communitydomain.CommunityDomain) *rest_err.RestErr {
	causes := []rest_err.Causes{}
	if !validation.IsValidName(community.Name, true) {
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
	if community.Address == (addressdomain.AddressDomain{}) || !uuidv7.IsValidString(community.Address.Id) {
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

func (c *communityValidator) ValidateUpdateCommunity(community communitydomain.CommunityDomain) *rest_err.RestErr {
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
	if name != "" && !validation.IsValidName(community.Name, true) {
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
