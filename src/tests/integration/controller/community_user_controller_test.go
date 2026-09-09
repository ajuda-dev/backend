package controller_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/samborkent/uuidv7"
)

func cleanupMembershipTest() {
	cleanCommunityUsersTable()
	cleanCommunityTable()
	cleanUsersTable()
	cleanAddressesTable()
}

func createCommunityUserForTest(t *testing.T, email string) *domain.UserDomain {
	t.Helper()
	user, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "membro",
		Email:    email,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create member user: %v", createErr)
	}
	return user
}

func createCommunityOwnedByForTest(t *testing.T, name string, ownerEmail string) string {
	t.Helper()
	owner, createErr := userRepository.CreateUser(&domain.UserDomain{
		Name:     "dono",
		Email:    ownerEmail,
		Password: "123456",
	})
	if createErr != nil {
		t.Fatalf("failed to create owner user: %v", createErr)
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
		Owner:       *owner,
		Address:     *address,
	})
	if cErr != nil {
		t.Fatalf("failed to create community: %v", cErr)
	}
	return community.Id
}

func newJoinCommunityRequest(communityId string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/v1/community/"+communityId+"/join", nil)
}

func newLeaveCommunityRequest(communityId string) *http.Request {
	return httptest.NewRequest(http.MethodDelete, "/v1/community/"+communityId+"/leave", nil)
}

func joinCommunityViaApi(t *testing.T, app *fiber.App, communityId string, token string) dto.CommunityUserDto {
	t.Helper()
	resp, err := doAuthedRequest(app, newJoinCommunityRequest(communityId), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusCreated {
		var respBody rest_err.RestErr
		json.NewDecoder(resp.Body).Decode(&respBody)
		t.Fatalf("esperava 201 ao entrar na comunidade, recebeu %d (body: %+v)", resp.StatusCode, respBody)
	}
	var respDto dto.CommunityUserDto
	if err := json.NewDecoder(resp.Body).Decode(&respDto); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	return respDto
}

func membershipCountByCommunity(t *testing.T, communityId string) int64 {
	t.Helper()
	count, err := communityUserRepository.CountByCommunity(communityId)
	if err != nil {
		t.Fatalf("failed to count memberships: %v", err)
	}
	return count
}

func membershipCountByUser(t *testing.T, userId string) int64 {
	t.Helper()
	count, err := communityUserRepository.CountByUserId(userId)
	if err != nil {
		t.Fatalf("failed to count memberships: %v", err)
	}
	return count
}

func TestJoinCommunitySuccess(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	communityId := createCommunityForTest(t, "Join Community Test")
	member := createCommunityUserForTest(t, "join_member@ajuda.dev")

	respDto := joinCommunityViaApi(t, app, communityId, validTokenFor(t, member.Id))

	if !uuidv7.IsValidString(respDto.Id) {
		t.Errorf("esperava id uuid v7 válido, recebeu '%s'", respDto.Id)
	}
	if respDto.CommunityId != communityId {
		t.Errorf("esperava community_id '%s', recebeu '%s'", communityId, respDto.CommunityId)
	}
	if respDto.UserId != member.Id {
		t.Errorf("esperava user_id '%s', recebeu '%s'", member.Id, respDto.UserId)
	}
	if count := membershipCountByCommunity(t, communityId); count != 1 {
		t.Errorf("esperava 1 membership na comunidade, recebeu %d", count)
	}
}

func TestJoinCommunityRejectsDuplicate(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	communityId := createCommunityForTest(t, "Duplicate Join Test")
	member := createCommunityUserForTest(t, "join_duplicate@ajuda.dev")
	token := validTokenFor(t, member.Id)

	joinCommunityViaApi(t, app, communityId, token)

	resp, err := doAuthedRequest(app, newJoinCommunityRequest(communityId), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("esperava 400 no join duplicado, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "Invalid membership data" {
		t.Errorf("esperava message 'Invalid membership data', recebeu '%s'", respBody.Message)
	}
	causes := getCauseByField("community_id", respBody.Causes)
	if len(causes) == 0 || causes[0] != "user is already a member of this community" {
		t.Errorf("esperava cause 'user is already a member of this community', recebeu %+v", respBody.Causes)
	}
	if count := membershipCountByCommunity(t, communityId); count != 1 {
		t.Errorf("esperava 1 membership na comunidade, recebeu %d", count)
	}
}

func TestJoinCommunityRejectsInvalidOrNonexistentCommunity(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	member := createCommunityUserForTest(t, "join_invalid@ajuda.dev")
	token := validTokenFor(t, member.Id)

	resp, err := doAuthedRequest(app, newJoinCommunityRequest("abc"), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 com id inválido, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	causes := getCauseByField("id", respBody.Causes)
	if len(causes) == 0 {
		t.Errorf("esperava cause para o campo 'id', recebeu %+v", respBody.Causes)
	}

	resp, err = doAuthedRequest(app, newJoinCommunityRequest(uuidv7.New().String()), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 em comunidade inexistente, recebeu %d", resp.StatusCode)
	}

	communityId := createCommunityForTest(t, "Join Soft Deleted Test")
	if delErr := communityRepository.SoftDeleteById(communityId); delErr != nil {
		t.Fatalf("failed to soft delete community: %v", delErr)
	}
	resp, err = doAuthedRequest(app, newJoinCommunityRequest(communityId), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 em comunidade soft-deletada, recebeu %d", resp.StatusCode)
	}
}

func TestJoinCommunityRejectsDeletedAuthenticatedUser(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	communityId := createCommunityForTest(t, "Join Deleted User Test")
	member := createCommunityUserForTest(t, "join_deleted_user@ajuda.dev")
	token := validTokenFor(t, member.Id)

	if err := db.Exec("DELETE FROM users WHERE id = ?", member.Id).Error; err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	resp, err := doAuthedRequest(app, newJoinCommunityRequest(communityId), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("esperava 401 para usuário do token inexistente no banco, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "invalid authenticated user" {
		t.Errorf("esperava message 'invalid authenticated user', recebeu '%s'", respBody.Message)
	}
}

func TestCommunityUsersRoutesRequireToken(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	communityId := uuidv7.New().String()

	resp, err := app.Test(newJoinCommunityRequest(communityId))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no join sem token, recebeu %d", resp.StatusCode)
	}

	resp, err = app.Test(newLeaveCommunityRequest(communityId))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("esperava 401 no leave sem token, recebeu %d", resp.StatusCode)
	}
}

func TestLeaveCommunitySuccess(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	communityId := createCommunityForTest(t, "Leave Community Test")
	member := createCommunityUserForTest(t, "leave_member@ajuda.dev")
	token := validTokenFor(t, member.Id)

	joinCommunityViaApi(t, app, communityId, token)

	resp, err := doAuthedRequest(app, newLeaveCommunityRequest(communityId), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Errorf("esperava 204 ao sair da comunidade, recebeu %d", resp.StatusCode)
	}
	if count := membershipCountByCommunity(t, communityId); count != 0 {
		t.Errorf("esperava 0 membership após leave, recebeu %d", count)
	}

	joinCommunityViaApi(t, app, communityId, token)
	if count := membershipCountByCommunity(t, communityId); count != 1 {
		t.Errorf("esperava 1 membership após novo join, recebeu %d", count)
	}
}

func TestLeaveCommunityWithoutMembership(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	communityId := createCommunityForTest(t, "Leave Without Membership Test")
	stranger := createCommunityUserForTest(t, "leave_stranger@ajuda.dev")

	resp, err := doAuthedRequest(app, newLeaveCommunityRequest(communityId), validTokenFor(t, stranger.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 ao sair sem membership, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	if respBody.Message != "membership not found" {
		t.Errorf("esperava message 'membership not found', recebeu '%s'", respBody.Message)
	}
}

func TestLeaveCommunityAsOwnerReturnsNotFound(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	communityId := createCommunityForTest(t, "Owner Leave Test")
	community, findErr := communityRepository.FindById(communityId)
	if findErr != nil {
		t.Fatalf("failed to find community: %v", findErr)
	}

	resp, err := doAuthedRequest(app, newLeaveCommunityRequest(communityId), validTokenFor(t, community.Owner.Id))
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 ao owner sair da própria comunidade, recebeu %d", resp.StatusCode)
	}
}

func TestLeaveCommunityRejectsInvalidOrNonexistentCommunity(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	member := createCommunityUserForTest(t, "leave_invalid@ajuda.dev")
	token := validTokenFor(t, member.Id)

	resp, err := doAuthedRequest(app, newLeaveCommunityRequest("abc"), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("esperava 400 com id inválido, recebeu %d", resp.StatusCode)
	}
	var respBody rest_err.RestErr
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("erro ao decodificar body: %v", err)
	}
	causes := getCauseByField("id", respBody.Causes)
	if len(causes) == 0 {
		t.Errorf("esperava cause para o campo 'id', recebeu %+v", respBody.Causes)
	}

	resp, err = doAuthedRequest(app, newLeaveCommunityRequest(uuidv7.New().String()), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 em comunidade inexistente, recebeu %d", resp.StatusCode)
	}

	communityId := createCommunityForTest(t, "Leave Soft Deleted Test")
	joinCommunityViaApi(t, app, communityId, token)
	if delErr := communityRepository.SoftDeleteById(communityId); delErr != nil {
		t.Fatalf("failed to soft delete community: %v", delErr)
	}
	resp, err = doAuthedRequest(app, newLeaveCommunityRequest(communityId), token)
	if err != nil {
		t.Fatalf("erro ao executar requisição: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("esperava 404 em comunidade soft-deletada, recebeu %d", resp.StatusCode)
	}
}

func TestCountByUserIdCountsOnlyActiveCommunityMemberships(t *testing.T) {
	t.Cleanup(cleanupMembershipTest)

	app := setupApp()
	communityOne := createCommunityOwnedByForTest(t, "Counts Community One", "counts_owner1@ajuda.dev")
	communityTwo := createCommunityOwnedByForTest(t, "Counts Community Two", "counts_owner2@ajuda.dev")
	member := createCommunityUserForTest(t, "counts_member@ajuda.dev")
	token := validTokenFor(t, member.Id)

	joinCommunityViaApi(t, app, communityOne, token)
	joinCommunityViaApi(t, app, communityTwo, token)

	if count := membershipCountByUser(t, member.Id); count != 2 {
		t.Errorf("esperava 2 memberships ativos, recebeu %d", count)
	}

	if delErr := communityRepository.SoftDeleteById(communityOne); delErr != nil {
		t.Fatalf("failed to soft delete community: %v", delErr)
	}

	if count := membershipCountByUser(t, member.Id); count != 1 {
		t.Errorf("esperava 1 membership após soft delete da comunidade, recebeu %d", count)
	}
	if count := membershipCountByCommunity(t, communityOne); count != 1 {
		t.Errorf("esperava membership preservada na comunidade soft-deletada, recebeu %d", count)
	}
}
