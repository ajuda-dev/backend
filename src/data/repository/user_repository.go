package repository

import (
	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/data/entity"
	"github.com/ajuda-dev/backend/src/service/domain"
	"github.com/samborkent/uuidv7"
	"gorm.io/gorm"
)

type UserFilter struct {
	SkillName string
}

type UserRepository interface {
	CreateUser(user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr)
	GetUserByEmail(email string) (*domain.UserDomain, *rest_err.RestErr)
	FindById(id string) (*domain.UserDomain, *rest_err.RestErr)
	FindAll(filter UserFilter, page int, limit int) (*domain.PageableUser, *rest_err.RestErr)
	Update(id string, user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr)
	SoftDeleteById(id string) *rest_err.RestErr
}

type userRepository struct {
	database *gorm.DB
}

// FindById implements UserRepository.
func (u *userRepository) FindById(id string) (*domain.UserDomain, *rest_err.RestErr) {
	var userEntity entity.UserEntity
	if err := u.database.Where("id = ?", id).First(&userEntity).Error; err != nil {
		return handlerErrorDataBase(err)
	}
	return userEntity.ToDomainUser(), nil
}

func (u *userRepository) FindAll(filter UserFilter, page int, limit int) (*domain.PageableUser, *rest_err.RestErr) {
	var users []entity.UserEntity
	query := u.database.Model(&entity.UserEntity{})
	query = query.Joins("JOIN skill_users ON skill_users.user_id = users.id").
		Joins("JOIN skills ON skills.id = skill_users.skill_id").
		Where("skills.name = ? AND skills.deleted_at IS NULL", filter.SkillName)

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

	pageable := &domain.PageableUser{
		HasNext: hasNext,
		Data:    make([]*domain.UserDomain, 0, len(users)),
	}
	if len(users) == 0 {
		return pageable, nil
	}

	ids := make([]string, len(users))
	for i := range users {
		ids[i] = users[i].Id
	}
	var skillUsers []entity.SkillUserEntity
	if err := u.database.Preload("Skill").
		Joins("JOIN skills ON skills.id = skill_users.skill_id").
		Where("skill_users.user_id IN ?", ids).
		Order("skills.name").
		Find(&skillUsers).Error; err != nil {
		return nil, rest_err.NewInternalServerError(err.Error())
	}

	skillsByUser := make(map[string][]domain.SkillDomain, len(users))
	for _, skillUser := range entity.ToSkillUserDomainList(skillUsers) {
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

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		database: db,
	}
}

func (u *userRepository) CreateUser(user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr) {
	var userEntity = entity.FromDomainUser(user)
	userEntity.Id = uuidv7.New().String()

	if err := u.database.Create(&userEntity).Error; err != nil {
		return &domain.UserDomain{}, rest_err.NewInternalServerError(err.Error())
	}
	if userEntity.Role == "" {
		userEntity.Role = domain.UserRoleUser
	}

	return userEntity.ToDomainUser(), nil
}
func (u *userRepository) GetUserByEmail(email string) (*domain.UserDomain, *rest_err.RestErr) {
	var userEntity entity.UserEntity
	if err := u.database.Where("email = ?", email).First(&userEntity).Error; err != nil {
		return handlerErrorDataBase(err)
	}
	return userEntity.ToDomainUser(), nil
}

func (u *userRepository) Update(id string, user *domain.UserDomain) (*domain.UserDomain, *rest_err.RestErr) {
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
		fields["config_visibility"] = entity.UserConfigVisibilityFromDomain(user.ConfigVisibility)
	}
	result := u.database.Model(&entity.UserEntity{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(fields)
	if result.Error != nil {
		return nil, rest_err.NewInternalServerError("Error updating user: " + result.Error.Error())
	}
	return u.FindById(id)
}

func (u *userRepository) SoftDeleteById(id string) *rest_err.RestErr {
	result := u.database.Where("id = ?", id).Delete(&entity.UserEntity{})
	if result.Error != nil {
		return rest_err.NewInternalServerError("Error deleting user: " + result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return rest_err.NewNotFoundError("User not found")
	}
	return nil
}

func handlerErrorDataBase(err error) (*domain.UserDomain, *rest_err.RestErr) {
	if err == gorm.ErrRecordNotFound {
		return nil, rest_err.NewNotFoundError("User not found")
	}
	return nil, rest_err.NewInternalServerError(err.Error())
}
