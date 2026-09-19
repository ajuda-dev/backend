package entity

import (
	"encoding/json"
	"testing"

	communitydomain "github.com/ajuda-dev/backend/src/service/community/domain"
)

func TestCommunityConfigVisibilityScanDropsUnknownKeys(t *testing.T) {
	var config CommunityConfigVisibility
	raw := `{
		"github":  { "value": "https://github.com/ajudadev" },
		"twitter": { "value": "https://x.com/ajudadev" },
		"email":   { "value": "vazamento" }
	}`
	if err := config.Scan([]byte(raw)); err != nil {
		t.Fatalf("erro ao ler o jsonb: %v", err)
	}

	if config.Github == nil || config.Github.Value != "https://github.com/ajudadev" {
		t.Errorf("esperava github lido do jsonb, recebeu %+v", config.Github)
	}
	persisted := config.ToDomain()
	if _, exists := persisted["twitter"]; exists {
		t.Errorf("esperava a chave desconhecida 'twitter' descartada na leitura, recebeu %+v", persisted)
	}
	if _, exists := persisted["email"]; exists {
		t.Errorf("esperava a chave desconhecida 'email' descartada na leitura, recebeu %+v", persisted)
	}
	if len(persisted) != 1 {
		t.Errorf("esperava apenas 1 chave após o Scan, recebeu %+v", persisted)
	}
}

func TestCommunityConfigVisibilityFromDomainDropsUnknownKeys(t *testing.T) {
	config := CommunityConfigVisibilityFromDomain(communitydomain.CommunityLinks{
		communitydomain.LinkKeyGithub:    {Value: "https://github.com/ajudadev"},
		communitydomain.LinkKeyLinkedin:  {Value: "https://www.linkedin.com/company/ajudadev"},
		communitydomain.LinkKeyOtherlink: {Value: "https://ajudadev.dev"},
		communitydomain.LinkKeyPhoto:     {Value: "https://avatars.githubusercontent.com/u/1"},
		"twitter":                        {Value: "https://x.com/ajudadev"},
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
		communitydomain.LinkKeyGithub, communitydomain.LinkKeyLinkedin,
		communitydomain.LinkKeyOtherlink, communitydomain.LinkKeyPhoto,
	} {
		if _, exists := persisted[key]; !exists {
			t.Errorf("esperava a chave '%s' gravada, recebeu %s", key, serialized)
		}
	}
	if len(persisted) != 4 {
		t.Errorf("esperava exatamente 4 chaves gravadas, recebeu %s", serialized)
	}
}

func TestCommunityConfigVisibilityValueNilWhenEmpty(t *testing.T) {
	var config CommunityConfigVisibility
	raw, err := config.Value()
	if err != nil {
		t.Fatalf("erro ao serializar a coluna vazia: %v", err)
	}
	if raw != nil {
		t.Errorf("esperava NULL para a config vazia, recebeu %v", raw)
	}
}

func TestCommunityConfigVisibilityScanNullKeepsEmpty(t *testing.T) {
	config := CommunityConfigVisibility{Github: &communitydomain.CommunityLink{Value: "https://github.com/ajudadev"}}
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

func TestCommunityConfigVisibilityRoundTrip(t *testing.T) {
	var config CommunityConfigVisibility
	raw := `{"photo": {"value": "https://foto.dev/comunidade.png"}}`
	if err := config.Scan([]byte(raw)); err != nil {
		t.Fatalf("erro ao ler o jsonb: %v", err)
	}

	persisted, ok := config.ToDomain()[communitydomain.LinkKeyPhoto]
	if !ok {
		t.Fatalf("esperava a chave photo no domínio, recebeu %+v", config.ToDomain())
	}
	if persisted.Value != "https://foto.dev/comunidade.png" {
		t.Errorf("esperava photo persistida, recebeu %+v", persisted)
	}
}
