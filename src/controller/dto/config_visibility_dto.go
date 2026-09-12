package dto

import "github.com/ajuda-dev/backend/src/service/domain"

type VisibilityConfigDto struct {
	Value              string `json:"value"`
	ShareWithCommunity bool   `json:"shareWithCommunity"`
}

type ConfigVisibilityDto map[string]VisibilityConfigDto

func (c ConfigVisibilityDto) ToDomain() domain.ConfigVisibility {
	if c == nil {
		return nil
	}
	config := domain.ConfigVisibility{}
	for key, item := range c {
		config[key] = domain.VisibilityConfig{
			Value:              item.Value,
			ShareWithCommunity: item.ShareWithCommunity,
		}
	}
	return config
}

func ConfigVisibilityDtoFromDomain(config domain.ConfigVisibility) ConfigVisibilityDto {
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
