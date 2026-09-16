package entity

import (
	"encoding/json"
	"testing"

	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

func TestUserConfigVisibilityScanDropsUnknownKeys(t *testing.T) {
	var config UserConfigVisibility
	raw := `{
		"github":   { "value": "https://github.com/lucas", "shareWithCommunity": true },
		"twitter":  { "value": "https://x.com/lucas", "shareWithCommunity": true },
		"password": { "value": "vazamento", "shareWithCommunity": true }
	}`
	if err := config.Scan([]byte(raw)); err != nil {
		t.Fatalf("erro ao ler o jsonb: %v", err)
	}

	if config.Github == nil || config.Github.Value != "https://github.com/lucas" {
		t.Errorf("esperava github lido do jsonb, recebeu %+v", config.Github)
	}
	if !config.Github.ShareWithCommunity {
		t.Errorf("esperava github com shareWithCommunity true, recebeu %+v", config.Github)
	}
	persisted := config.ToDomain()
	if _, exists := persisted["twitter"]; exists {
		t.Errorf("esperava a chave desconhecida 'twitter' descartada na leitura, recebeu %+v", persisted)
	}
	if _, exists := persisted["password"]; exists {
		t.Errorf("esperava a chave desconhecida 'password' descartada na leitura, recebeu %+v", persisted)
	}
	if len(persisted) != 1 {
		t.Errorf("esperava apenas 1 chave após o Scan, recebeu %+v", persisted)
	}
}

func TestUserConfigVisibilityFromDomainDropsUnknownKeys(t *testing.T) {
	config := UserConfigVisibilityFromDomain(userdomain.ConfigVisibility{
		userdomain.VisibilityKeyGithub:    {Value: "https://github.com/lucas", ShareWithCommunity: true},
		userdomain.VisibilityKeyLinkedin:  {Value: "https://www.linkedin.com/in/devrocha/"},
		userdomain.VisibilityKeyOtherlink: {Value: "https://linktr.ee/devrocha"},
		userdomain.VisibilityKeyPhoto:     {Value: "https://avatars.githubusercontent.com/u/33586465"},
		userdomain.VisibilityKeyEmail:     {Value: "lucas@ajuda.dev"},
		userdomain.VisibilityKeyPhone:     {Value: "+55 (11) 99999-9999"},
		"twitter":                         {Value: "https://x.com/lucas"},
	})

	raw, err := config.Value()
	if err != nil {
		t.Fatalf("erro ao serializar a coluna: %v", err)
	}
	serialized, ok := raw.(string)
	if !ok {
		t.Fatalf("esperava string como valor gravado, recebeu %T", raw)
	}

	var persisted map[string]any
	if err := json.Unmarshal([]byte(serialized), &persisted); err != nil {
		t.Fatalf("erro ao decodificar o valor gravado: %v", err)
	}
	if _, exists := persisted["twitter"]; exists {
		t.Errorf("esperava a chave desconhecida descartada na escrita, recebeu %s", serialized)
	}
	for _, key := range []string{
		userdomain.VisibilityKeyGithub, userdomain.VisibilityKeyLinkedin, userdomain.VisibilityKeyOtherlink,
		userdomain.VisibilityKeyPhoto, userdomain.VisibilityKeyEmail, userdomain.VisibilityKeyPhone,
	} {
		if _, exists := persisted[key]; !exists {
			t.Errorf("esperava a chave '%s' gravada, recebeu %s", key, serialized)
		}
	}
	if len(persisted) != 6 {
		t.Errorf("esperava exatamente 6 chaves gravadas, recebeu %s", serialized)
	}
}

func TestUserConfigVisibilityValueNilWhenEmpty(t *testing.T) {
	var config UserConfigVisibility
	raw, err := config.Value()
	if err != nil {
		t.Fatalf("erro ao serializar a coluna vazia: %v", err)
	}
	if raw != nil {
		t.Errorf("esperava NULL para a config vazia, recebeu %v", raw)
	}
}

func TestUserConfigVisibilityScanNullKeepsEmpty(t *testing.T) {
	config := UserConfigVisibility{Github: &userdomain.VisibilityConfig{Value: "https://github.com/lucas"}}
	if err := config.Scan(nil); err != nil {
		t.Fatalf("erro ao ler NULL: %v", err)
	}
	raw, err := config.Value()
	if err != nil {
		t.Fatalf("erro ao serializar após ler NULL: %v", err)
	}
	if raw != nil {
		t.Errorf("esperava config vazia após ler NULL, recebeu %v", raw)
	}
}

func TestUserConfigVisibilityRoundTrip(t *testing.T) {
	var config UserConfigVisibility
	raw := `{"photo": {"value": "https://foto.dev/lucas.png", "shareWithCommunity": true}}`
	if err := config.Scan([]byte(raw)); err != nil {
		t.Fatalf("erro ao ler o jsonb: %v", err)
	}

	persisted, ok := config.ToDomain()[userdomain.VisibilityKeyPhoto]
	if !ok {
		t.Fatalf("esperava a chave photo no domínio, recebeu %+v", config.ToDomain())
	}
	if persisted.Value != "https://foto.dev/lucas.png" || !persisted.ShareWithCommunity {
		t.Errorf("esperava photo com valor e share persistidos, recebeu %+v", persisted)
	}
}
