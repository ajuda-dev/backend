package controller

import (
	"strconv"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
)

type CommunityController interface {
	RegisterCommunity() fiber.Handler
	GetAllCommunities() fiber.Handler
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
			logger.Error("erro", err)
			return cf.Status(err.Code).JSON(err)
		}
		return cf.Status(fiber.StatusCreated).JSON(registerCommunityDto.FromDomain(address))
	}
}



func (c *communityController) GetAllCommunities() fiber.Handler {
	return func(cf *fiber.Ctx) error {
		page, _ := strconv.Atoi(cf.Query("page", "1"))
    limit, _ := strconv.Atoi(cf.Query("limit", "10"))

	addressIDStr := cf.Query("address_id")
		var addressID int = 0
		if addressIDStr != "" {
			if idParsed, err := strconv.ParseUint(addressIDStr, 10, 64); err == nil {
					addressID = int(idParsed)
			}
		}
		result, e := c.communityService.GetAll(addressID, page, limit)
		if e != nil {
			logger.Error("error: ", e)
			return cf.Status(e.Code).JSON(e)
		}

		dtoResult := dto.PageableCommunityDto{}.FromDomain(*result)
		return cf.Status(fiber.StatusOK).JSON(dtoResult)
	}
}
