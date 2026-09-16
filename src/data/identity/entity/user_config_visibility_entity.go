package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

// UserConfigVisibility declara explicitamente as únicas chaves aceitas na coluna
// jsonb config_visibility. Qualquer chave fora desta lista é descartada na
// leitura (Scan) e na escrita (Value), mesmo que exista no banco.
type UserConfigVisibility struct {
	Github    *userdomain.VisibilityConfig `json:"github,omitempty"`
	Linkedin  *userdomain.VisibilityConfig `json:"linkedin,omitempty"`
	Otherlink *userdomain.VisibilityConfig `json:"otherlink,omitempty"`
	Photo     *userdomain.VisibilityConfig `json:"photo,omitempty"`
	Email     *userdomain.VisibilityConfig `json:"email,omitempty"`
	Phone     *userdomain.VisibilityConfig `json:"phone,omitempty"`
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

func (c UserConfigVisibility) ToDomain() userdomain.ConfigVisibility {
	config := userdomain.ConfigVisibility{}
	if c.Github != nil {
		config[userdomain.VisibilityKeyGithub] = *c.Github
	}
	if c.Linkedin != nil {
		config[userdomain.VisibilityKeyLinkedin] = *c.Linkedin
	}
	if c.Otherlink != nil {
		config[userdomain.VisibilityKeyOtherlink] = *c.Otherlink
	}
	if c.Photo != nil {
		config[userdomain.VisibilityKeyPhoto] = *c.Photo
	}
	if c.Email != nil {
		config[userdomain.VisibilityKeyEmail] = *c.Email
	}
	if c.Phone != nil {
		config[userdomain.VisibilityKeyPhone] = *c.Phone
	}
	return config
}

func UserConfigVisibilityFromDomain(config userdomain.ConfigVisibility) UserConfigVisibility {
	entityConfig := UserConfigVisibility{}
	if item, ok := config[userdomain.VisibilityKeyGithub]; ok {
		github := item
		entityConfig.Github = &github
	}
	if item, ok := config[userdomain.VisibilityKeyLinkedin]; ok {
		linkedin := item
		entityConfig.Linkedin = &linkedin
	}
	if item, ok := config[userdomain.VisibilityKeyOtherlink]; ok {
		otherlink := item
		entityConfig.Otherlink = &otherlink
	}
	if item, ok := config[userdomain.VisibilityKeyPhoto]; ok {
		photo := item
		entityConfig.Photo = &photo
	}
	if item, ok := config[userdomain.VisibilityKeyEmail]; ok {
		email := item
		entityConfig.Email = &email
	}
	if item, ok := config[userdomain.VisibilityKeyPhone]; ok {
		phone := item
		entityConfig.Phone = &phone
	}
	return entityConfig
}
