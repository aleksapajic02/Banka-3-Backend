package bank

import (
	"fmt"

	bankpb "github.com/RAF-SI-2025/Banka-3-Backend/gen/bank"
)

func (s *Server) GetActiveAccountsByOwnerID(ownerID uint64) ([]Account, error) {
	var accounts []Account
	result := s.db_gorm.Where(&Account{Owner: int64(ownerID), Active: true}).
		Order("balance DESC").
		Find(&accounts)
	return accounts, result.Error
}

func (s *Server) GetAccountsForEmployee(firstName, lastName, accountNumber string) ([]Account, error) {
	var accounts []Account
	query := s.db_gorm.Model(&Account{})

	if accountNumber != "" {
		query = query.Where("number = ?", accountNumber)
	}

	if firstName != "" || lastName != "" {
		query = query.Joins("JOIN clients ON clients.id = accounts.owner")
		if firstName != "" {
			query = query.Where("clients.first_name ILIKE ?", firstName+"%")
		}
		if lastName != "" {
			query = query.Where("clients.last_name ILIKE ?", lastName+"%")
		}
	}

	result := query.Find(&accounts)
	return accounts, result.Error
}

func (s *Server) GetAccountByNumber(accNumber string) (*Account, error) {
	var acc Account
	result := s.db_gorm.Where(&Account{Number: accNumber}).First(&acc)
	if result.Error != nil {
		return nil, result.Error
	}
	return &acc, nil
}

func (s *Server) GetCompanyByOwnerID(ownerID int64) (*Company, error) {
	var company Company
	result := s.db_gorm.Where(&Company{Owner_id: ownerID}).First(&company)
	if result.Error != nil {
		return nil, result.Error
	}
	return &company, nil
}

func (s *Server) GetFilteredTransactions(accNumber string, date string, amount int64, status string) ([]*bankpb.ClientTransaction, error) {
	var pbTransactions []*bankpb.ClientTransaction

	var payments []Payment
	payQuery := s.db_gorm.Model(&Payment{}).Where("from_account = ? OR to_account = ?", accNumber, accNumber)
	if date != "" {
		payQuery = payQuery.Where("DATE(timestamp) = ?", date)
	}
	if amount > 0 {
		payQuery = payQuery.Where("start_amount = ?", amount)
	}
	// Dodata opciona filtracija po statusu ako je prosleđen
	if status != "" {
		payQuery = payQuery.Where("status = ?", status)
	}
	payQuery.Order("timestamp DESC").Find(&payments)

	for _, p := range payments {
		pbTransactions = append(pbTransactions, &bankpb.ClientTransaction{
			FromAccount:     p.From_account,
			ToAccount:       p.To_account,
			InitialAmount:   float64(p.Start_amount),
			FinalAmount:     float64(p.End_amount),
			Fee:             float64(p.Commission),
			PaymentCode:     fmt.Sprintf("%d", p.Transaction_code),
			ReferenceNumber: p.Call_number,
			Purpose:         p.Reason,
			Status:          p.Status, // Koristi se status iz baze
			Timestamp:       p.Timestamp.Unix(),
		})
	}

	var transfers []Transfer
	transQuery := s.db_gorm.Model(&Transfer{}).Where("from_account = ? OR to_account = ?", accNumber, accNumber)
	if date != "" {
		transQuery = transQuery.Where("DATE(timestamp) = ?", date)
	}
	if amount > 0 {
		transQuery = transQuery.Where("start_amount = ?", amount)
	}
	transQuery.Order("timestamp DESC").Find(&transfers)

	for _, t := range transfers {
		pbTransactions = append(pbTransactions, &bankpb.ClientTransaction{
			FromAccount:   t.From_account,
			ToAccount:     t.To_account,
			InitialAmount: float64(t.Start_amount),
			FinalAmount:   float64(t.End_amount),
			Fee:           float64(t.Commission),
			Status:        "realized", // Pretpostavka za transfer dok se ne doda status u Transfer model
			Timestamp:     t.Timestamp.Unix(),
		})
	}

	return pbTransactions, nil
}
