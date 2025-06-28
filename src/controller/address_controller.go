package controller

import (
	"github.com/ajuda-dev/backend/src/controller/dto"
	"github.com/ajuda-dev/backend/src/service"
	"github.com/gofiber/fiber/v2"
)

type AddressController interface {
	RegisterAddress() fiber.Handler
}

type addressController struct {
	addressService service.AddressService
}


func NewAddressController(addressService service.AddressService) AddressController {
	return &addressController{
		addressService: addressService,
	}
}

func (a *addressController) RegisterAddress() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var addressDto dto.AddressDto
		if err := c.BodyParser(&addressDto); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Não foi possível processar o corpo da requisição",
			})
		}
		address, err := a.addressService.CreateAddress(addressDto.ToDomain())
		if err != nil {
			return c.Status(err.Code).JSON(err)
		}
		return c.Status(fiber.StatusCreated).JSON(addressDto.FromDomain(address))
	}
}
