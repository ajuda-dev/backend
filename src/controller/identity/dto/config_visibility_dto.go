package dto

import (
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

type VisibilityConfigDto struct {
	Value              string `json:"value"`
	ShareWithCommunity bool   `json:"shareWithCommunity"`
}

type ConfigVisibilityDto map[string]VisibilityConfigDto

func (c ConfigVisibilityDto) ToDomain() userdomain.ConfigVisibility {
	if c == nil {
		return nil
	}
	config := userdomain.ConfigVisibility{}
	for key, item := range c {
		config[key] = userdomain.VisibilityConfig{
			Value:              item.Value,
			ShareWithCommunity: item.ShareWithCommunity,
		}
	}
	return config
}

func ConfigVisibilityDtoFromDomain(config userdomain.ConfigVisibility) ConfigVisibilityDto {
	if config == nil {
		return nil
	}
	dtoConfig := ConfigVisibilityDto{}
	for key, item := range config {
		dtoConfig[key] = VisibilityConfigDto{
			Value:              item.Value,
			ShareWithCommunity: item.ShareWithCommunity,
		}
	}
	return dtoConfig
}
