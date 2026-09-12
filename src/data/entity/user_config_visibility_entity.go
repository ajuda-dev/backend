package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/ajuda-dev/backend/src/service/domain"
)

// UserConfigVisibility declara explicitamente as únicas chaves aceitas na coluna
// jsonb config_visibility. Qualquer chave fora desta lista é descartada na
// leitura (Scan) e na escrita (Value), mesmo que exista no banco.
type UserConfigVisibility struct {
	Github    *domain.VisibilityConfig `json:"github,omitempty"`
	Linkedin  *domain.VisibilityConfig `json:"linkedin,omitempty"`
	Otherlink *domain.VisibilityConfig `json:"otherlink,omitempty"`
	Photo     *domain.VisibilityConfig `json:"photo,omitempty"`
	Email     *domain.VisibilityConfig `json:"email,omitempty"`
	Phone     *domain.VisibilityConfig `json:"phone,omitempty"`
}

func (c UserConfigVisibility) IsEmpty() bool {
	return c.Github == nil && c.Linkedin == nil && c.Otherlink == nil &&
		c.Photo == nil && c.Email == nil && c.Phone == nil
}

func (c UserConfigVisibility) Value() (driver.Value, error) {
	if c.IsEmpty() {
		return nil, nil
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

func (c *UserConfigVisibility) Scan(value any) error {
	if value == nil {
		*c = UserConfigVisibility{}
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

func (c UserConfigVisibility) ToDomain() domain.ConfigVisibility {
	config := domain.ConfigVisibility{}
	if c.Github != nil {
		config[domain.VisibilityKeyGithub] = *c.Github
	}
	if c.Linkedin != nil {
		config[domain.VisibilityKeyLinkedin] = *c.Linkedin
	}
	if c.Otherlink != nil {
		config[domain.VisibilityKeyOtherlink] = *c.Otherlink
	}
	if c.Photo != nil {
		config[domain.VisibilityKeyPhoto] = *c.Photo
	}
	if c.Email != nil {
		config[domain.VisibilityKeyEmail] = *c.Email
	}
	if c.Phone != nil {
		config[domain.VisibilityKeyPhone] = *c.Phone
	}
	return config
}

func UserConfigVisibilityFromDomain(config domain.ConfigVisibility) UserConfigVisibility {
	entityConfig := UserConfigVisibility{}
	if item, ok := config[domain.VisibilityKeyGithub]; ok {
		github := item
		entityConfig.Github = &github
	}
	if item, ok := config[domain.VisibilityKeyLinkedin]; ok {
		linkedin := item
		entityConfig.Linkedin = &linkedin
	}
	if item, ok := config[domain.VisibilityKeyOtherlink]; ok {
		otherlink := item
		entityConfig.Otherlink = &otherlink
	}
	if item, ok := config[domain.VisibilityKeyPhoto]; ok {
		photo := item
		entityConfig.Photo = &photo
	}
	if item, ok := config[domain.VisibilityKeyEmail]; ok {
		email := item
		entityConfig.Email = &email
	}
	if item, ok := config[domain.VisibilityKeyPhone]; ok {
		phone := item
		entityConfig.Phone = &phone
	}
	return entityConfig
}
