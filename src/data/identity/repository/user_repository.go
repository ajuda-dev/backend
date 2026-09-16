package repository

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	userentity "github.com/ajuda-dev/backend/src/data/identity/entity"
	notificationrepo "github.com/ajuda-dev/backend/src/data/notification/repository"
	skillentity "github.com/ajuda-dev/backend/src/data/skill/entity"
	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
	notificationdomain "github.com/ajuda-dev/backend/src/service/notification/domain"
	skilldomain "github.com/ajuda-dev/backend/src/service/skill/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type UserFilter struct {
	SkillName string
	Name      string
	Email     string
}

type UserRepository interface {
	CreateUser(user *userdomain.UserDomain) (*userdomain.UserDomain, *rest_err.RestErr)
	GetUserByEmail(email string) (*userdomain.UserDomain, *rest_err.RestErr)
	FindById(id string) (*userdomain.UserDomain, *rest_err.RestErr)
	FindAll(filter UserFilter, page int, limit int) (*userdomain.PageableUser, *rest_err.RestErr)
	Update(id string, user *userdomain.UserDomain) (*userdomain.UserDomain, *rest_err.RestErr)
	UpdatePassword(id string, hashedPassword string) *rest_err.RestErr
	MarkEmailVerified(id string, at time.Time) *rest_err.RestErr
	SoftDeleteById(id string) *rest_err.RestErr
}

type userRepository struct {
	database *gorm.DB
	outbox   notificationrepo.OutboxEventRepository
}

// FindById implements UserRepository.
func (u *userRepository) FindById(id string) (*userdomain.UserDomain, *rest_err.RestErr) {
	var userEntity userentity.UserEntity
	if err := u.database.Where("id = ?", id).First(&userEntity).Error; err != nil {
		return handlerErrorDataBase(err)
	}
	return userEntity.ToDomainUser(), nil
}

func (u *userRepository) FindAll(filter UserFilter, page int, limit int) (*userdomain.PageableUser, *rest_err.RestErr) {
	var users []userentity.UserEntity
	query := u.database.Model(&userentity.UserEntity{})

	// Caminho A (com skill): mantém o mecanismo atual — join + match exato do nome.
	if filter.SkillName != "" {
		query = query.Joins("JOIN skill_users ON skill_users.user_id = users.id").
			Joins("JOIN skills ON skills.id = skill_users.skill_id").
			Where("skills.name = ? AND skills.deleted_at IS NULL", filter.SkillName)
	}
	// Caminho B (sem skill): sem join — lista todos os usuários ativos (soft delete já
	// vem do Model(&UserEntity{})); quem não tem skill aparece com "skills": [].

	name := strings.ToLower(strings.TrimSpace(filter.Name))
	if name != "" {
		search := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(name)
		// unaccent nos DOIS lados (padrão do community): quem digita "josé" acha "jose" e vice-versa.
		query = query.Where("unaccent(LOWER(users.name)) LIKE unaccent(?)", "%"+search+"%")
	}
	email := strings.ToLower(strings.TrimSpace(filter.Email))
	if email != "" {
		// match exato e case-insensitive; e-mail é ASCII por validação (service/validation).
		query = query.Where("LOWER(users.email) = ?", email)
	}

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	result := query.Order("users.name").Offset(offset).Limit(limit + 1).Find(&users)
	if result.Error != nil {
		return nil, rest_err.NewInternalServerError(result.Error.Error())
	}

	hasNext := len(users) > limit
	if hasNext {
		users = users[:limit]
	}

	pageable := &userdomain.PageableUser{
		HasNext: hasNext,
		Data:    make([]*userdomain.UserDomain, 0, len(users)),
	}
	if len(users) == 0 {
		return pageable, nil
	}

	ids := make([]string, len(users))
	for i := range users {
		ids[i] = users[i].Id
	}
	var skillUsers []skillentity.SkillUserEntity
	if err := u.database.Preload("Skill").
		Joins("JOIN skills ON skills.id = skill_users.skill_id").
		Where("skill_users.user_id IN ?", ids).
		Order("skills.name").
		Find(&skillUsers).Error; err != nil {
		return nil, rest_err.NewInternalServerError(err.Error())
	}

	skillsByUser := make(map[string][]skilldomain.SkillDomain, len(users))
	for _, skillUser := range skillentity.ToSkillUserDomainList(skillUsers) {
		if skillUser.Skill != nil {
			skillsByUser[skillUser.UserId] = append(skillsByUser[skillUser.UserId], *skillUser.Skill)
		}
	}
	for i := range users {
		userDomain := users[i].ToDomainUser()
		userDomain.Skills = skillsByUser[users[i].Id]
		pageable.Data = append(pageable.Data, userDomain)
	}
	return pageable, nil
}

func NewUserRepository(db *gorm.DB, outbox notificationrepo.OutboxEventRepository) UserRepository {
	return &userRepository{
		database: db,
		outbox:   outbox,
	}
}

func (u *userRepository) CreateUser(user *userdomain.UserDomain) (*userdomain.UserDomain, *rest_err.RestErr) {
	userEntity := userentity.FromDomainUser(user)
	userEntity.Id = uuidv7.New().String()
	enqueueCreatedAccount := u.outbox != nil && userEntity.EmailVerifiedAt == nil
	if !enqueueCreatedAccount && userEntity.EmailVerifiedAt == nil {
		now := time.Now()
		userEntity.EmailVerifiedAt = &now
	}

	if !enqueueCreatedAccount {
		if err := u.database.Create(userEntity).Error; err != nil {
			return &userdomain.UserDomain{}, rest_err.NewInternalServerError(err.Error())
		}
		if userEntity.Role == "" {
			userEntity.Role = userdomain.UserRoleUser
		}
		return userEntity.ToDomainUser(), nil
	}

	txErr := u.database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(userEntity).Error; err != nil {
			return rest_err.NewInternalServerError(err.Error())
		}
		payload, _ := json.Marshal(map[string]string{
			"email": userEntity.Email,
			"name":  userEntity.Name,
		})
		outboxEvent := &notificationdomain.OutboxEventDomain{
			Type:    notificationdomain.OutboxTypeCreatedAccount,
			UserId:  userEntity.Id,
			Payload: payload,
			Status:  notificationdomain.OutboxStatusPending,
		}
		if err := u.outbox.Create(tx, outboxEvent); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		return nil, toRestErr(txErr)
	}
	if userEntity.Role == "" {
		userEntity.Role = userdomain.UserRoleUser
	}
	return userEntity.ToDomainUser(), nil
}
func (u *userRepository) GetUserByEmail(email string) (*userdomain.UserDomain, *rest_err.RestErr) {
	var userEntity userentity.UserEntity
	if err := u.database.Where("email = ?", email).First(&userEntity).Error; err != nil {
		return handlerErrorDataBase(err)
	}
	return userEntity.ToDomainUser(), nil
}

func (u *userRepository) Update(id string, user *userdomain.UserDomain) (*userdomain.UserDomain, *rest_err.RestErr) {
	// map (e não struct) porque `Updates` com struct ignora campos zero, e aqui
	// "vazio" significa "não alterar". Hoje só o nome é atualizável; o padrão
	// fica pronto para o plano de troca de e-mail/senha sem reescrita.
	fields := map[string]interface{}{}
	if user.Name != "" {
		fields["name"] = user.Name
	}
	if user.Description != "" {
		fields["description"] = user.Description
	}
	if user.ConfigVisibility != nil {
		fields["config_visibility"] = userentity.UserConfigVisibilityFromDomain(user.ConfigVisibility)
	}
	result := u.database.Model(&userentity.UserEntity{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(fields)
	if result.Error != nil {
		return nil, rest_err.NewInternalServerError("Error updating user: " + result.Error.Error())
	}
	return u.FindById(id)
}

func (u *userRepository) UpdatePassword(id string, hashedPassword string) *rest_err.RestErr {
	result := u.database.Model(&userentity.UserEntity{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("password", hashedPassword)
	if result.Error != nil {
		return rest_err.NewInternalServerError("Error updating password: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return rest_err.NewNotFoundError("User not found")
	}
	return nil
}

func (u *userRepository) MarkEmailVerified(id string, at time.Time) *rest_err.RestErr {
	result := u.database.Model(&userentity.UserEntity{}).
		Where("id = ? AND deleted_at IS NULL AND email_verified_at IS NULL", id).
		Update("email_verified_at", at)
	if result.Error != nil {
		return rest_err.NewInternalServerError("Error verifying email: " + result.Error.Error())
	}
	return nil
}

func (u *userRepository) SoftDeleteById(id string) *rest_err.RestErr {
	result := u.database.Where("id = ?", id).Delete(&userentity.UserEntity{})
	if result.Error != nil {
		return rest_err.NewInternalServerError("Error deleting user: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return rest_err.NewNotFoundError("User not found")
	}
	return nil
}

func handlerErrorDataBase(err error) (*userdomain.UserDomain, *rest_err.RestErr) {
	if err == gorm.ErrRecordNotFound {
		return nil, rest_err.NewNotFoundError("User not found")
	}
	return nil, rest_err.NewInternalServerError(err.Error())
}
