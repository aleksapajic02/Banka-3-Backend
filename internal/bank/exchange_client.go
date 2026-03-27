package bank

import (
	"context"

	exchangepb "github.com/RAF-SI-2025/Banka-3-Backend/gen/exchange"
)

func (s *Server) callConvertMoney(ctx context.Context, from, to string, amount float64) (*exchangepb.ConversionResponse, error) {
	return s.ExchangeService.ConvertMoney(ctx, &exchangepb.ConversionRequest{
		FromCurrency: from,
		ToCurrency:   to,
		Amount:       amount,
	})
}
