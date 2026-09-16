package identity

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	communityrepo "github.com/ajuda-dev/backend/src/data/community/repository"
	eventrepo "github.com/ajuda-dev/backend/src/data/event/repository"
	userrepo "github.com/ajuda-dev/backend/src/data/identity/repository"
	notificationrepo "github.com/ajuda-dev/backend/src/data/notification/repository"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	identityvalidator "github.com/ajuda-dev/backend/src/service/identity/validator"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
	"golang.org/x/crypto/bcrypt"
)

func NewUserService(userRepository userrepo.UserRepository, validator identityvalidator.UserValidator, authService AuthService,
	communityRepository communityrepo.CommunityRepository,
	eventRepository eventrepo.EventRepository,
	eventUserRepository eventrepo.EventUserRepository,
	communityUserRepository communityrepo.CommunityUserRepository,
	outboxEventRepository notificationrepo.OutboxEventRepository,
	emailCodeRepository userrepo.EmailCodeRepository) UserService {
	cfg := EmailCodeConfigFromEnv()
	return &userService{
		userRepository:          userRepository,
		validator:               validator,
		authService:             authService,
		communityRepository:     communityRepository,
		eventRepository:         eventRepository,
		eventUserRepository:     eventUserRepository,
		communityUserRepository: communityUserRepository,
		outboxEventRepository:   outboxEventRepository,
		emailCodeRepository:     emailCodeRepository,
		emailCodeCfg:            cfg,
		rateLimiter:             newEmailCodeRateLimiter(cfg),
	}
}

type UserService interface {
	CreateUser(user *userdomain.UserDomain) (*userdomain.UserDomain, string, *rest_err.RestErr)
	FindById(id string) (*userdomain.UserDomain, *rest_err.RestErr)
	GetUserById(targetId string, requesterId string) (*userdomain.UserDomain, *rest_err.RestErr)
	GetAllUsers(filter userrepo.UserFilter, page int, limit int) (*userdomain.PageableUser, *rest_err.RestErr)
	UpdateUser(targetId string, requesterId string, changes *userdomain.UserDomain) (*userdomain.UserDomain, *rest_err.RestErr)
	DeleteUser(targetId string, requesterId string) *rest_err.RestErr
	VerifyEmail(userId, code string) (*userdomain.UserDomain, *rest_err.RestErr)
	ResendVerification(userId string) *rest_err.RestErr
}

type userService struct {
	userRepository          userrepo.UserRepository
	validator               identityvalidator.UserValidator
	authService             AuthService
	communityRepository     communityrepo.CommunityRepository
	eventRepository         eventrepo.EventRepository
	eventUserRepository     eventrepo.EventUserRepository
	communityUserRepository communityrepo.CommunityUserRepository
	outboxEventRepository   notificationrepo.OutboxEventRepository
	emailCodeRepository     userrepo.EmailCodeRepository
	emailCodeCfg            EmailCodeConfig
	rateLimiter             *emailCodeRateLimiter
}

// FindById implements UserService.
func (u *userService) FindById(id string) (*userdomain.UserDomain, *rest_err.RestErr) {
	return u.userRepository.FindById(id)
}

// GetUserById implements UserService.
func (u *userService) GetUserById(targetId string, requesterId string) (*userdomain.UserDomain, *rest_err.RestErr) {
	requester, err := AuthenticatedUser(u, requesterId)
	if err != nil {
		return nil, err
	}
	target, err := u.userRepository.FindById(targetId)
	if err != nil {
		return nil, err
	}
	ApplyVisibilityFilter(target, requester)
	return target, nil
}

func normalizeSkillName(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

// GetAllUsers implements UserService.
func (u *userService) GetAllUsers(filter userrepo.UserFilter, page int, limit int) (*userdomain.PageableUser, *rest_err.RestErr) {
	skillName := normalizeSkillName(filter.SkillName)
	if len(skillName) > 50 {
		return nil, rest_err.NewBadRequestValidationError(
			"Invalid query params",
			[]rest_err.Causes{{
				Field:   "skill",
				Message: "Skill name is not valid",
			}})
	}
	return u.userRepository.FindAll(userrepo.UserFilter{
		SkillName: skillName,
		Name:      filter.Name,
		Email:     filter.Email,
	}, page, limit)
}

// CreateUser implements UserService.
func (u *userService) CreateUser(user *userdomain.UserDomain) (*userdomain.UserDomain, string, *rest_err.RestErr) {
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

func (u *userService) VerifyEmail(userId, code string) (*userdomain.UserDomain, *rest_err.RestErr) {
	user, err := u.userRepository.FindById(userId)
	if err != nil {
		return nil, err
	}
	if user.EmailVerified() {
		return user, nil
	}

	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, rest_err.NewBadRequestValidationError("Invalid request", []rest_err.Causes{
			{Field: "code", Message: "Code cannot be empty"},
		})
	}

	now := time.Now()
	if rateErr := u.rateLimiter.AllowVerificationAttempt(user.Id, now); rateErr != nil {
		return nil, rateErr
	}

	invalid := rest_err.NewUnauthorizedError("invalid or expired code")
	stored, findErr := u.emailCodeRepository.FindActive(user.Id, userdomain.EmailCodePurposeConfirm, now)
	expected := strings.Repeat("0", 64)
	if stored != nil {
		expected = stored.CodeHash
	}
	if findErr != nil && findErr.Code != rest_err.NOT_FOUND {
		return nil, findErr
	}
	if !constantTimeEqualCode(hashEmailConfirmCode(code, u.emailCodeCfg.Secret), expected) || stored == nil {
		u.rateLimiter.RecordFailedAttempt(user.Id, now)
		return nil, invalid
	}
	if consumeErr := u.emailCodeRepository.MarkConsumed(stored.Id); consumeErr != nil {
		return nil, consumeErr
	}
	if markErr := u.userRepository.MarkEmailVerified(user.Id, now); markErr != nil {
		return nil, markErr
	}
	u.rateLimiter.ClearAttempts(user.Id)
	return u.userRepository.FindById(user.Id)
}

func (u *userService) ResendVerification(userId string) *rest_err.RestErr {
	user, err := u.userRepository.FindById(userId)
	if err != nil {
		return err
	}
	if user.EmailVerified() {
		return nil
	}
	if u.outboxEventRepository == nil {
		return rest_err.NewInternalServerError("outbox is not configured")
	}

	now := time.Now()
	if rateErr := u.rateLimiter.AllowVerificationSend(user.Id, now); rateErr != nil {
		return rateErr
	}

	payload, _ := json.Marshal(map[string]string{
		"email": user.Email,
		"name":  user.Name,
	})
	return u.outboxEventRepository.Create(nil, &notificationdomain.OutboxEventDomain{
		Type:    notificationdomain.OutboxTypeCreatedAccount,
		UserId:  user.Id,
		Payload: payload,
		Status:  notificationdomain.OutboxStatusPending,
	})
}

// UpdateUser implements UserService.
func (u *userService) UpdateUser(targetId string, requesterId string, changes *userdomain.UserDomain) (*userdomain.UserDomain, *rest_err.RestErr) {
	requester, err := AuthenticatedUser(u, requesterId)
	if err != nil {
		return nil, err
	}
	if requester.Id != targetId && requester.Role != userdomain.UserRoleAdmin {
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
		merged[userdomain.VisibilityKeyEmail] = userdomain.VisibilityConfig{
			Value:              current.Email,
			ShareWithCommunity: merged[userdomain.VisibilityKeyEmail].ShareWithCommunity,
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
	requester, err := AuthenticatedUser(u, requesterId)
	if err != nil {
		return err
	}
	if requester.Role != userdomain.UserRoleAdmin {
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
