package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"io"

	"github.com/ajuda-dev/backend/src/config/rest_err"
	"github.com/ajuda-dev/backend/src/service/domain"
)

// ViaCepResponse represents the JSON structure returned by ViaCEP API
type ViaCepResponse struct {
	Cep        string `json:"cep"`
	Logradouro string `json:"logradouro"`
	Localidade string `json:"localidade"`
	Uf         string `json:"uf"`
	Erro       bool   `json:"erro,omitempty"`
}

var (
	baseUrl = "https://viacep.com.br/ws"
)

type AddressSearchClient interface {
	SearchAddress(address domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr)
}

type addressSearchClient struct {
}

func NewAddressSearchClient() AddressSearchClient {
	return &addressSearchClient{}
}

func (a *addressSearchClient) SearchAddress(address domain.AddressDomain) (*domain.AddressDomain, *rest_err.RestErr) {
	url := buildUrl(address)
	resp, err := http.Get(url)
	if err != nil {
		return nil, rest_err.NewInternalServerError("Erro ao buscar endereço: " + err.Error())
	}
	defer resp.Body.Close()
	var data ViaCepResponse
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, rest_err.NewInternalServerError("error reading response body: " + err.Error())
	}

	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		var dataArr []ViaCepResponse
		if err := json.Unmarshal(bodyBytes, &dataArr); err != nil || len(dataArr) == 0 {
			return nil, rest_err.NewInternalServerError("error decoding response viacep: " + err.Error())
		}
		data = dataArr[0]
	}
	if data.Erro {
		return nil, rest_err.NewBadRequestError("invalid search address data")
	}
	
	return viaCepResponseToAddressDomain(data), nil
}

func viaCepResponseToAddressDomain(data ViaCepResponse) *domain.AddressDomain {
	return &domain.AddressDomain{
		City:    data.Localidade,
		State:   data.Uf,
		Street:  data.Logradouro,
		ZipCode: data.Cep,
	}
}

func buildUrl(address domain.AddressDomain) string {
	if address.ZipCode != "" {
		return fmt.Sprintf("%s/%s/json/", baseUrl, address.ZipCode)
	}
	return  fmt.Sprintf("%s/%s/%s/%s/json/", baseUrl, address.State, address.City, address.Street)
}
