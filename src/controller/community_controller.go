package controller

import (
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
)

type CommunityController interface {
	RegisterCommunity() fiber.Handler
}

type communityController struct {
	communityService service.CommunityService
}



func NewCommunityController(communityService service.CommunityService) CommunityController {
	return &communityController{
		communityService: communityService,
	}
}


// RegisterCommunity implements CommunityController.
func (c *communityController) RegisterCommunity() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		var registerCommunityDto dto.RegisterCommunityDto
		if err := cf.BodyParser(&registerCommunityDto); err != nil {
			logger.Error("erro body request", err)
			return cf.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
	
		address, err := c.communityService.CreateCommunity(registerCommunityDto.ToDomain())
		if err != nil {
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(registerCommunityDto.FromDomain(address))
	}
}

