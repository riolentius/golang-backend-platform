package dashboard

import "time"

const LowStockThreshold = 10

type RecentTransaction struct {
	ID            string    `json:"id"`
	CustomerName  string    `json:"customerName"`
	TotalAmount   string    `json:"totalAmount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	PaymentStatus string    `json:"paymentStatus"`
	CreatedAt     time.Time `json:"createdAt"`
}

type LowStockItem struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	SKU            *string `json:"sku,omitempty"`
	StockOnHand    int     `json:"stockOnHand"`
	StockReserved  int     `json:"stockReserved"`
	AvailableStock int     `json:"availableStock"`
}

type LowStockSummary struct {
	HasLowStock bool           `json:"hasLowStock"`
	Items       []LowStockItem `json:"items"`
}

type Summary struct {
	RecentTransactions []RecentTransaction `json:"recentTransactions"`
	LowStock           LowStockSummary     `json:"lowStock"`
}

type TopProduct struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	SKU          *string `json:"sku,omitempty"`
	TotalQtySold int     `json:"totalQtySold"`
	TotalRevenue string  `json:"totalRevenue"`
}
