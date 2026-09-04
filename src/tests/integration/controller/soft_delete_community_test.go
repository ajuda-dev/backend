package controller_test

import (
	"strconv"
	"testing"

	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
)

func itoa(n uint) string {
	return strconv.FormatUint(uint64(n), 10)
}

func createCommunityForTest(t *testing.T, name string) uint {
	t.Helper()
	user, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "teste",
		Email:    testEmail,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}

	address, aErr := addressRepository.CreateAddress(&domain.AddressDomain{
		City:    "test_city",
		State:   "test_state",
		Street:  "test_street",
		ZipCode: "test_zip_code",
	})
	if aErr != nil {
		t.Fatalf("failed to create address: %v", aErr)
	}

	community, cErr := communityRepository.CreateCommunity(&domain.CommunityDomain{
		Name:        name,
		Description: "comunidade de teste",
		Owner:       *user,
		Address:     *address,
	})
	if cErr != nil {
		t.Fatalf("failed to create community: %v", cErr)
	}
	return community.Id
}

func TestSoftDeleteCommunityRemovesFromListing(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	id := createCommunityForTest(t, "Dev Mode Community")

	if delErr := communityRepository.SoftDeleteById(id); delErr != nil {
		t.Fatalf("failed to soft delete community: %v", delErr)
	}

	page, findErr := communityRepository.FindAll(0, 1, 10)
	if findErr != nil {
		t.Fatalf("failed to list communities: %v", findErr)
	}
	if len(page.Data) != 0 {
		t.Errorf("esperava nenhuma comunidade na listagem, recebeu %d", len(page.Data))
	}

	_, findByIdErr := communityRepository.FindById(id)
	if findByIdErr == nil || findByIdErr.Code != fiber.StatusNotFound {
		t.Errorf("esperava 404 no FindById, recebeu %+v", findByIdErr)
	}
}

func TestSoftDeletedCommunityAllowsNameReuse(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	id := createCommunityForTest(t, "Dev Mode")

	if delErr := communityRepository.SoftDeleteById(id); delErr != nil {
		t.Fatalf("failed to soft delete community: %v", delErr)
	}

	app := setupApp()
	user, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "teste",
		Email:    testEmail,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create user: %v", createErr)
	}
	address, aErr := addressRepository.CreateAddress(&domain.AddressDomain{
		City:    "test_city",
		State:   "test_state",
		Street:  "test_street",
		ZipCode: "test_zip_code",
	})
	if aErr != nil {
		t.Fatalf("failed to create address: %v", aErr)
	}

	body := []byte(`{
		"address_id": ` + itoa(address.Id) + `,
		"owner_id": "` + user.Id + `",
		"name": "Dev Mode",
		"description": "comunidade recriada"
	}`)
	req := newCommunityRegisterRequest(body)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("esperava 201 ao recriar com nome repetido, recebeu %d", resp.StatusCode)
	}
}

func TestRestoreCommunity(t *testing.T) {
	t.Cleanup(cleanAddressesTable)
	t.Cleanup(cleanUsersTable)
	t.Cleanup(cleanCommunityTable)

	id := createCommunityForTest(t, "Dev Mode Community")

	if delErr := communityRepository.SoftDeleteById(id); delErr != nil {
		t.Fatalf("failed to soft delete community: %v", delErr)
	}

	if err := db.Unscoped().Model(&entity.CommunityEntity{}).Where("id = ?", id).Update("deleted_at", nil).Error; err != nil {
		t.Fatalf("failed to restore community: %v", err)
	}

	community, findErr := communityRepository.FindById(id)
	if findErr != nil {
		t.Fatalf("esperava achar a comunidade restaurada, recebeu %v", findErr)
	}
	if community.Id != id {
		t.Errorf("esperava comunidade de id %d, recebeu %d", id, community.Id)
	}
}
