package sync

import (
	"math"
	"testing"
)

func TestGetFloat_Float64(t *testing.T) {
	row := map[string]interface{}{"DishSumInt": float64(45000)}
	got := GetFloat(row, "DishSumInt")
	if got != 45000 {
		t.Errorf("GetFloat(float64) = %v, want 45000", got)
	}
}

func TestGetFloat_Nil(t *testing.T) {
	row := map[string]interface{}{"DishSumInt": nil}
	got := GetFloat(row, "DishSumInt")
	if got != 0 {
		t.Errorf("GetFloat(nil) = %v, want 0", got)
	}
}

func TestGetFloat_MissingKey(t *testing.T) {
	row := map[string]interface{}{}
	got := GetFloat(row, "DishSumInt")
	if got != 0 {
		t.Errorf("GetFloat(missing) = %v, want 0", got)
	}
}

func TestGetString_Normal(t *testing.T) {
	row := map[string]interface{}{"UniqOrderId.Id": "abc-123"}
	got := GetString(row, "UniqOrderId.Id")
	if got != "abc-123" {
		t.Errorf("GetString = %q, want %q", got, "abc-123")
	}
}

func TestGetString_Nil(t *testing.T) {
	row := map[string]interface{}{"key": nil}
	got := GetString(row, "key")
	if got != "" {
		t.Errorf("GetString(nil) = %q, want empty", got)
	}
}

func TestAggregateOrders_MultiDish(t *testing.T) {
	// Simulate 3 dishes in one order: 5000 + 3000 + 2500 = 10500
	rows := []map[string]interface{}{
		{"UniqOrderId.Id": "order-1", "OpenDate.Typed": "2026-04-09", "DishName": "Beshbarmak", "DishSumInt": float64(5000), "DishDiscountSumInt": float64(0), "DishAmountInt": float64(1)},
		{"UniqOrderId.Id": "order-1", "OpenDate.Typed": "2026-04-09", "DishName": "Lagman", "DishSumInt": float64(3000), "DishDiscountSumInt": float64(100), "DishAmountInt": float64(1)},
		{"UniqOrderId.Id": "order-1", "OpenDate.Typed": "2026-04-09", "DishName": "Chai", "DishSumInt": float64(2500), "DishDiscountSumInt": float64(0), "DishAmountInt": float64(2)},
	}

	orders := AggregateOrdersFromOLAP(rows)

	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}

	agg := orders["order-1"]
	if agg == nil {
		t.Fatal("order-1 not found")
	}

	if agg.Revenue != 10500 {
		t.Errorf("Revenue = %v, want 10500 (5000+3000+2500)", agg.Revenue)
	}
	if agg.Discount != 100 {
		t.Errorf("Discount = %v, want 100", agg.Discount)
	}
	if agg.ItemCount != 4 {
		t.Errorf("ItemCount = %v, want 4 (1+1+2)", agg.ItemCount)
	}
	if agg.OrderDate != "2026-04-09" {
		t.Errorf("OrderDate = %q, want 2026-04-09", agg.OrderDate)
	}
}

func TestAggregateOrders_MultipleOrders(t *testing.T) {
	rows := []map[string]interface{}{
		{"UniqOrderId.Id": "order-1", "OpenDate.Typed": "2026-04-09", "DishName": "A", "DishSumInt": float64(5000), "DishDiscountSumInt": float64(0), "DishAmountInt": float64(1)},
		{"UniqOrderId.Id": "order-2", "OpenDate.Typed": "2026-04-09", "DishName": "B", "DishSumInt": float64(8000), "DishDiscountSumInt": float64(0), "DishAmountInt": float64(1)},
		{"UniqOrderId.Id": "order-1", "OpenDate.Typed": "2026-04-09", "DishName": "C", "DishSumInt": float64(3000), "DishDiscountSumInt": float64(0), "DishAmountInt": float64(1)},
	}

	orders := AggregateOrdersFromOLAP(rows)

	if len(orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(orders))
	}
	if orders["order-1"].Revenue != 8000 {
		t.Errorf("order-1 Revenue = %v, want 8000", orders["order-1"].Revenue)
	}
	if orders["order-2"].Revenue != 8000 {
		t.Errorf("order-2 Revenue = %v, want 8000", orders["order-2"].Revenue)
	}
}

func TestAggregateOrders_NoDivision(t *testing.T) {
	// Revenue should NEVER be divided by 100. A dish costing 49000 tenge
	// must remain 49000, not become 490.
	rows := []map[string]interface{}{
		{"UniqOrderId.Id": "order-big", "OpenDate.Typed": "2026-04-09", "DishName": "Steak", "DishSumInt": float64(49000), "DishDiscountSumInt": float64(0), "DishAmountInt": float64(1)},
	}

	orders := AggregateOrdersFromOLAP(rows)
	agg := orders["order-big"]

	if agg.Revenue != 49000 {
		t.Errorf("Revenue = %v, want 49000 (must NOT be divided by 100)", agg.Revenue)
	}
	if agg.Revenue < 1000 {
		t.Errorf("Revenue %v is suspiciously low — likely divided by 100", agg.Revenue)
	}
}

func TestAggregateOrders_EmptyOrderID(t *testing.T) {
	rows := []map[string]interface{}{
		{"UniqOrderId.Id": "", "DishSumInt": float64(5000)},
		{"DishSumInt": float64(3000)},
	}

	orders := AggregateOrdersFromOLAP(rows)
	if len(orders) != 0 {
		t.Errorf("expected 0 orders for empty IDs, got %d", len(orders))
	}
}

func TestAggregateOrders_LargeOrder(t *testing.T) {
	// Simulate a banquet order with 20 dishes totaling ~500,000 KZT
	rows := make([]map[string]interface{}, 20)
	var expectedTotal float64
	for i := 0; i < 20; i++ {
		price := float64(20000 + i*1000)
		expectedTotal += price
		rows[i] = map[string]interface{}{
			"UniqOrderId.Id":    "banquet-1",
			"OpenDate.Typed":    "2026-04-09",
			"DishName":          "Dish",
			"DishSumInt":        price,
			"DishDiscountSumInt": float64(0),
			"DishAmountInt":     float64(1),
		}
	}

	orders := AggregateOrdersFromOLAP(rows)
	agg := orders["banquet-1"]

	if math.Abs(agg.Revenue-expectedTotal) > 0.01 {
		t.Errorf("Revenue = %v, want %v", agg.Revenue, expectedTotal)
	}
	if agg.Revenue < 100000 {
		t.Errorf("Banquet revenue %v is too low — expected > 100,000 KZT", agg.Revenue)
	}
}
