package dto

import (
	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
)

type CommunityLinkDto struct {
	Value string `json:"value"`
}

type CommunityLinksDto map[string]CommunityLinkDto

func (c CommunityLinksDto) ToDomain() communitydomain.CommunityLinks {
	if c == nil {
		return nil
	}
	config := communitydomain.CommunityLinks{}
	for key, item := range c {
		config[key] = communitydomain.CommunityLink{Value: item.Value}
	}
	return config
}

func CommunityLinksDtoFromDomain(config communitydomain.CommunityLinks) CommunityLinksDto {
	if len(config) == 0 {
		return nil
	}
	dtoConfig := CommunityLinksDto{}
	for key, item := range config {
		dtoConfig[key] = CommunityLinkDto{Value: item.Value}
	}
	return dtoConfig
}
