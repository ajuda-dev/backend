package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
)

// CommunityConfigVisibility declara as únicas chaves aceitas na coluna jsonb
// config_visibility da comunidade. Chaves fora desta lista são descartadas na
// leitura (Scan) e na escrita (Value). Não há shareWithCommunity: os links
// são sempre públicos.
type CommunityConfigVisibility struct {
	Github    *communitydomain.CommunityLink `json:"github,omitempty"`
	Linkedin  *communitydomain.CommunityLink `json:"linkedin,omitempty"`
	Otherlink *communitydomain.CommunityLink `json:"otherlink,omitempty"`
	Photo     *communitydomain.CommunityLink `json:"photo,omitempty"`
}

func (c CommunityConfigVisibility) IsEmpty() bool {
	return c.Github == nil && c.Linkedin == nil && c.Otherlink == nil && c.Photo == nil
}

func (c CommunityConfigVisibility) Value() (driver.Value, error) {
	if c.IsEmpty() {
		return nil, nil
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

func (c *CommunityConfigVisibility) Scan(value any) error {
	if value == nil {
		*c = CommunityConfigVisibility{}
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported type for config_visibility: %T", v)
	}
	return json.Unmarshal(raw, c)
}

func (c CommunityConfigVisibility) ToDomain() communitydomain.CommunityLinks {
	config := communitydomain.CommunityLinks{}
	if c.Github != nil {
		config[communitydomain.LinkKeyGithub] = *c.Github
	}
	if c.Linkedin != nil {
		config[communitydomain.LinkKeyLinkedin] = *c.Linkedin
	}
	if c.Otherlink != nil {
		config[communitydomain.LinkKeyOtherlink] = *c.Otherlink
	}
	if c.Photo != nil {
		config[communitydomain.LinkKeyPhoto] = *c.Photo
	}
	return config
}

func CommunityConfigVisibilityFromDomain(config communitydomain.CommunityLinks) CommunityConfigVisibility {
	entityConfig := CommunityConfigVisibility{}
	if item, ok := config[communitydomain.LinkKeyGithub]; ok {
		github := item
		entityConfig.Github = &github
	}
	if item, ok := config[communitydomain.LinkKeyLinkedin]; ok {
		linkedin := item
		entityConfig.Linkedin = &linkedin
	}
	if item, ok := config[communitydomain.LinkKeyOtherlink]; ok {
		otherlink := item
		entityConfig.Otherlink = &otherlink
	}
	if item, ok := config[communitydomain.LinkKeyPhoto]; ok {
		photo := item
		entityConfig.Photo = &photo
	}
	return entityConfig
}
