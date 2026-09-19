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
	} else if len(community.Description) > 500 {
		causes = append(causes, rest_err.Causes{
			Field:   "description",
			Message: "description must have at most 500 characters",
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
	causes = append(causes, validateCommunityLinks(community.ConfigVisibility)...)

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

	if name == "" && description == "" && addressId == "" && len(community.ConfigVisibility) == 0 {
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
	if description != "" && len(community.Description) > 500 {
		causes = append(causes, rest_err.Causes{
			Field:   "description",
			Message: "description must have at most 500 characters",
		})
	}
	if addressId != "" && !uuidv7.IsValidString(addressId) {
		causes = append(causes, rest_err.Causes{
			Field:   "address_id",
			Message: "AddressId is not valid",
		})
	}
	causes = append(causes, validateCommunityLinks(community.ConfigVisibility)...)
	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError("Invalid community data", causes)
	}
	return nil
}

func validateCommunityLinks(config communitydomain.CommunityLinks) []rest_err.Causes {
	causes := []rest_err.Causes{}
	for key, item := range config {
		field := "config_visibility." + key
		value := strings.TrimSpace(item.Value)
		switch key {
		case communitydomain.LinkKeyGithub, communitydomain.LinkKeyLinkedin, communitydomain.LinkKeyOtherlink, communitydomain.LinkKeyPhoto:
			if value == "" {
				continue
			}
			if !validation.IsValidHTTPURL(value) {
				causes = append(causes, rest_err.Causes{
					Field:   field + ".value",
					Message: "value must be a valid http or https url",
				})
			} else if len(value) > 500 {
				causes = append(causes, rest_err.Causes{
					Field:   field + ".value",
					Message: "value must have at most 500 characters",
				})
			}
		default:
			causes = append(causes, rest_err.Causes{
				Field:   field,
				Message: "unsupported visibility key",
			})
		}
	}
	return causes
}
