package address

import (
	"regexp"
	"strconv"

	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/ajuda-dev/backend/src/config/rest_err"
	addressdto "github.com/ajuda-dev/backend/src/controller/address/dto"
	addressrepo "github.com/ajuda-dev/backend/src/data/address/repository"
	"github.com/ajuda-dev/backend/src/service/address"
	"github.com/gofiber/fiber/v2"
)

var hasDigit = regexp.MustCompile(`[0-9]`)

type AddressController interface {
	RegisterAddress() fiber.Handler
	GetAllAddresses() fiber.Handler
}

type addressController struct {
	addressService address.AddressService
}

func NewAddressController(addressService address.AddressService) AddressController {
	return &addressController{
		addressService: addressService,
	}
}

// RegisterAddress godoc
// @Summary      Registra um novo endereço
// @Description  Cria um novo endereço no sistema
// @Tags         addresses
// @Accept       json
// @Produce      json
// @Param        address  body  addressdto.AddressDto  true  "Dados do endereço"
// @Success      201   {object}  addressdto.AddressDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/address/register [post]
func (a *addressController) RegisterAddress() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var addressDto addressdto.AddressDto
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

// GetAllAddresses godoc
// @Summary      Lista endereços
// @Description  Retorna os endereços com paginação e filtros opcionais por cidade (busca parcial, case-insensitive e accent-insensitive), estado (UF exata) e CEP (igualdade pelos dígitos, aceita com ou sem hífen)
// @Tags         addresses
// @Accept       json
// @Produce      json
// @Param        page      query   int     false  "Página"
// @Param        limit     query   int     false  "Limite"
// @Param        city      query   string  false  "Cidade (busca parcial, case-insensitive, ignora acentos)"
// @Param        state     query   string  false  "UF (igualdade exata, ex. SP)"
// @Param        zip_code  query   string  false  "CEP (igualdade pelos dígitos, aceita com ou sem hífen)"
// @Success      200   {object}  addressdto.PageableAddressDto
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /v1/address [get]
func (a *addressController) GetAllAddresses() fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, _ := strconv.Atoi(c.Query("page", "1"))
		limit, _ := strconv.Atoi(c.Query("limit", "10"))

		filter := addressrepo.AddressFilter{
			City:    c.Query("city"),
			State:   c.Query("state"),
			ZipCode: c.Query("zip_code"),
		}
		if filter.ZipCode != "" && !hasDigit.MatchString(filter.ZipCode) {
			return c.Status(fiber.StatusBadRequest).JSON(rest_err.NewBadRequestValidationError(
				"Invalid query params",
				[]rest_err.Causes{{Field: "zip_code", Message: "zip_code must contain digits"}}))
		}

		result, e := a.addressService.GetAll(filter, page, limit)
		if e != nil {
			logger.Error("error: ", e)
			return c.Status(e.Code).JSON(e)
		}
		return c.Status(fiber.StatusOK).JSON(addressdto.PageableAddressDto{}.FromDomain(*result))
	}
}
