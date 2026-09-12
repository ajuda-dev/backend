package service

import (
	"strings"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/repository"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/ajuda-dev/backend/src/service/validator"
	"golang.org/x/crypto/bcrypt"
)

func NewUserService(userRepository repository.UserRepository, validator validator.UserValidator, authService AuthService,
	communityRepository repository.CommunityRepository,
	eventRepository repository.EventRepository,
	eventUserRepository repository.EventUserRepository,
	communityUserRepository repository.CommunityUserRepository) UserService {
	return &userService{
		userRepository:         userRepository,
		validator:              validator,
		authService:            authService,
		communityRepository:    communityRepository,
		eventRepository:        eventRepository,
		eventUserRepository:    eventUserRepository,
		communityUserRepository: communityUserRepository,
	}
}

type UserService interface {
	CreateUser(user *domain.UserDomain) (*domain.UserDomain, string, *rest_err.RestErr)
	FindById(id string) (*domain.UserDomain, *rest_err.RestErr)
	GetAllUsers(filter repository.UserFilter, page int, limit int) (*domain.PageableUser, *rest_err.RestErr)
	UpdateUser(targetId string, requesterId string, changes *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr)
	DeleteUser(targetId string, requesterId string) *rest_err.RestErr
}

type userService struct {
	userRepository         repository.UserRepository
	validator              validator.UserValidator
	authService            AuthService
	communityRepository    repository.CommunityRepository
	eventRepository        repository.EventRepository
	eventUserRepository    repository.EventUserRepository
	communityUserRepository repository.CommunityUserRepository
}

// FindById implements UserService.
func (u *userService) FindById(id string) (*domain.UserDomain, *rest_err.RestErr) {
	return u.userRepository.FindById(id)
}

// GetAllUsers implements UserService.
func (u *userService) GetAllUsers(filter repository.UserFilter, page int, limit int) (*domain.PageableUser, *rest_err.RestErr) {
	skillName := normalizeSkillName(filter.SkillName)
	if skillName == "" {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid query params",
			[]rest_err.Causes{{
				Field:   "skill",
				Message: "query param skill is required",
			}})
	}
	if len(skillName) > 50 {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid query params",
			[]rest_err.Causes{{
				Field:   "skill",
				Message: "Skill name is not valid",
			}})
	}
	return u.userRepository.FindAll(repository.UserFilter{SkillName: skillName}, page, limit)
}

// CreateUser implements UserService.
func (u *userService) CreateUser(user *domain.UserDomain) (*domain.UserDomain, string, *rest_err.RestErr) {
	err := u.validator.ValidateRegisterUser(*user)
	if err != nil {
		return nil, "", err
	}
	existingUser, err := u.userRepository.GetUserByEmail(user.Email)
	if err != nil && err.Code != rest_err.NOT_FOUND {
		return nil, "", rest_err.NewInternalServerError(err.Error())
	}

	if existingUser != nil {
		return nil, "", rest_err.NewBadRequestValidationError(
			"Invalid user data",
			[]rest_err.Causes{
				{
					Field:   "email",
					Message: "Email already exists",
				},
			})
	}
	password, errH := hashPassword(user.Password)
	if errH != nil {
		return nil, "", rest_err.NewInternalServerError(errH.Error())
	}
	user.Password = password
	user, err = u.userRepository.CreateUser(user)
	if err != nil {
		return nil, "", err
	}
	token, err := u.authService.CreateToken(user)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

// UpdateUser implements UserService.
func (u *userService) UpdateUser(targetId string, requesterId string, changes *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr) {
	requester, err := authenticatedUser(u, requesterId)
	if err != nil {
		return nil, err
	}
	if requester.Id != targetId && requester.Role != domain.UserRoleAdmin {
		return nil, rest_err.NewForbiddenError("only the user themselves or an admin can update this user")
	}
	current, err := u.userRepository.FindById(targetId)
	if err != nil {
		return nil, err
	}
	if err := u.validator.ValidateUpdateUser(*changes); err != nil {
		return nil, err
	}
	if changes.ConfigVisibility != nil {
		merged := current.ConfigVisibility
		for key, item := range changes.ConfigVisibility {
			merged[key] = item
		}
		merged[domain.VisibilityKeyEmail] = domain.VisibilityConfig{
			Value:              current.Email,
			ShareWithCommunity: merged[domain.VisibilityKeyEmail].ShareWithCommunity,
		}
		changes.ConfigVisibility = merged
	}
	if changes.Description != "" {
		changes.Description = strings.TrimSpace(changes.Description)
	}
	return u.userRepository.Update(targetId, changes)
}

// DeleteUser implements UserService.
func (u *userService) DeleteUser(targetId string, requesterId string) *rest_err.RestErr {
	requester, err := authenticatedUser(u, requesterId)
	if err != nil {
		return err
	}
	if requester.Role != domain.UserRoleAdmin {
		return rest_err.NewForbiddenError("only admins can delete users")
	}
	if _, err := u.userRepository.FindById(targetId); err != nil {
		return err
	}

	causes := []rest_err.Causes{}
	communityCount, countErr := u.communityRepository.CountByOwnerId(targetId)
	if countErr != nil {
		return countErr
	}
	if communityCount > 0 {
		causes = append(causes, rest_err.Causes{
			Field:   "community",
			Message: "is owner of an active community",
		})
	}
	eventCount, countErr := u.eventRepository.CountByOwnerId(targetId)
	if countErr != nil {
		return countErr
	}
	if eventCount > 0 {
		causes = append(causes, rest_err.Causes{
			Field:   "event",
			Message: "is owner of an active event",
		})
	}
	participationCount, countErr := u.eventUserRepository.CountActiveByUserId(targetId)
	if countErr != nil {
		return countErr
	}
	if participationCount > 0 {
		causes = append(causes, rest_err.Causes{
			Field:   "event_users",
			Message: "has an active participation in an event",
		})
	}
	membershipCount, countErr := u.communityUserRepository.CountByUserId(targetId)
	if countErr != nil {
		return countErr
	}
	if membershipCount > 0 {
		causes = append(causes, rest_err.Causes{
			Field:   "community_users",
			Message: "is a member of an active community",
		})
	}
	if len(causes) > 0 {
		return rest_err.NewBadRequestValidationError("cannot delete user with active associations", causes)
	}
	return u.userRepository.SoftDeleteById(targetId)
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// func checkPasswordHash(password, hash string) bool {
// 	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
// 	return err == nil
// }
