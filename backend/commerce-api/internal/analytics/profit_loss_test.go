package analytics

import (
	"strings"
	"testing"
)

func TestFinalizeProfitLossComplete(
	t *testing.T,
) {
	result :=
		ProfitLoss{
			GrossMerchandiseRevenueAmount: 150000,

			DiscountAmount: 10000,

			ShippingRevenueAmount: 5000,

			GrossCollectedRevenueAmount: 145000,

			SuccessfulRefundAmount: 5000,

			UnitsSold: 10,

			CostedUnits: 10,

			GrossCOGSAmount: 80000,

			RestockedUnits: 2,

			RestockCostedUnits: 2,

			RestockCOGSRecoveryAmount: 16000,

			ReturnReceivedUnits: 3,

			KnownReturnInventoryLossAmount: 8000,

			RecordedExpensesAmount: 10000,
		}

	finalizeProfitLoss(
		&result,
		"BDT",
	)

	if !result.ProfitComplete {
		t.Fatal(
			"expected complete profit",
		)
	}

	if result.NetMerchandiseRevenueAmount !=
		140000 {

		t.Fatalf(
			"unexpected net merchandise revenue: %d",
			result.NetMerchandiseRevenueAmount,
		)
	}

	if result.NetCollectedRevenueAmount !=
		140000 {

		t.Fatalf(
			"unexpected net revenue: %d",
			result.NetCollectedRevenueAmount,
		)
	}

	if result.NetCOGSAmount !=
		64000 {

		t.Fatalf(
			"unexpected net COGS: %d",
			result.NetCOGSAmount,
		)
	}

	if result.SalesCOGSCoverageBPS !=
		10000 {

		t.Fatalf(
			"unexpected sales COGS coverage: %d",
			result.SalesCOGSCoverageBPS,
		)
	}

	if result.RestockCOGSCoverageBPS !=
		10000 {

		t.Fatalf(
			"unexpected restock COGS coverage: %d",
			result.RestockCOGSCoverageBPS,
		)
	}

	if result.COGSCoverageBPS !=
		10000 {

		t.Fatalf(
			"unexpected total COGS coverage: %d",
			result.COGSCoverageBPS,
		)
	}

	if result.KnownGrossProfitAmount !=
		76000 {

		t.Fatalf(
			"unexpected known gross profit: %d",
			result.KnownGrossProfitAmount,
		)
	}

	if result.GrossProfitAmount == nil ||
		*result.GrossProfitAmount !=
			76000 {

		t.Fatalf(
			"unexpected gross profit: %#v",
			result.GrossProfitAmount,
		)
	}

	if result.NetProfitAfterRecordedExpensesAmount ==
		nil ||
		*result.NetProfitAfterRecordedExpensesAmount !=
			66000 {

		t.Fatalf(
			"unexpected net profit after expenses: %#v",
			result.NetProfitAfterRecordedExpensesAmount,
		)
	}

	if result.NonRestockedReturnUnits !=
		1 {

		t.Fatalf(
			"unexpected non-restocked units: %d",
			result.NonRestockedReturnUnits,
		)
	}

	/*
		The non-restocked return loss is informational here.

		Its cost already remains inside net COGS because only
		restocked units are recovered. Subtracting it again
		would double-count the inventory loss.
	*/
	if result.KnownReturnInventoryLossAmount !=
		8000 {

		t.Fatalf(
			"unexpected known return inventory loss: %d",
			result.KnownReturnInventoryLossAmount,
		)
	}

	if result.ExpenseBasis !=
		"recorded_only" {

		t.Fatalf(
			"unexpected expense basis: %q",
			result.ExpenseBasis,
		)
	}

	if result.Currency !=
		"BDT" {

		t.Fatalf(
			"unexpected currency: %q",
			result.Currency,
		)
	}
}

func TestFinalizeProfitLossIncompleteCost(
	t *testing.T,
) {
	result :=
		ProfitLoss{
			GrossCollectedRevenueAmount: 100000,

			UnitsSold: 10,

			CostedUnits: 8,

			MissingCostUnits: 2,

			GrossCOGSAmount: 50000,

			RecordedExpensesAmount: 12000,
		}

	finalizeProfitLoss(
		&result,
		"BDT",
	)

	if result.ProfitComplete {
		t.Fatal(
			"expected incomplete profit",
		)
	}

	if result.COGSCoverageBPS !=
		8000 {

		t.Fatalf(
			"unexpected incomplete COGS coverage: %d",
			result.COGSCoverageBPS,
		)
	}

	if result.KnownGrossProfitAmount !=
		50000 {

		t.Fatalf(
			"unexpected known gross profit: %d",
			result.KnownGrossProfitAmount,
		)
	}

	if result.GrossProfitAmount != nil {
		t.Fatal(
			"gross profit must be null when cost coverage is incomplete",
		)
	}

	if result.GrossMarginBPS != nil {
		t.Fatal(
			"gross margin must be null when cost coverage is incomplete",
		)
	}

	if result.NetProfitAfterRecordedExpensesAmount !=
		nil {

		t.Fatal(
			"net profit must be null when cost coverage is incomplete",
		)
	}

	if !warningContains(
		result.Warnings,
		"order-item cost snapshot",
	) ||
		!warningContains(
			result.Warnings,
			"historical SKU buying cost",
		) {

		t.Fatalf(
			"expected incomplete-cost warning, got %v",
			result.Warnings,
		)
	}

	if !warningContains(
		result.Warnings,
		"recorded expenses",
	) {
		t.Fatalf(
			"expected recorded-expense warning, got %v",
			result.Warnings,
		)
	}
}

func TestFinalizeProfitLossForeignCurrencyExpensesExcluded(
	t *testing.T,
) {
	result :=
		ProfitLoss{
			GrossCollectedRevenueAmount: 75000,

			UnitsSold: 5,

			CostedUnits: 5,

			GrossCOGSAmount: 30000,

			RecordedExpenseEntries: 2,

			RecordedExpensesAmount: 5000,

			ForeignCurrencyExpenseEntriesExcluded: 3,
		}

	finalizeProfitLoss(
		&result,
		"BDT",
	)

	if !result.ProfitComplete {
		t.Fatal(
			"expected complete profit",
		)
	}

	if result.NetProfitAfterRecordedExpensesAmount ==
		nil ||
		*result.NetProfitAfterRecordedExpensesAmount !=
			40000 {

		t.Fatalf(
			"unexpected net profit after recorded expenses: %#v",
			result.NetProfitAfterRecordedExpensesAmount,
		)
	}

	if !warningContains(
		result.Warnings,
		"other currencies are excluded",
	) {
		t.Fatalf(
			"expected foreign-currency warning, got %v",
			result.Warnings,
		)
	}
}

func TestFinalizeProfitLossPointIncompleteCost(
	t *testing.T,
) {
	point :=
		ProfitLossPoint{
			GrossCollectedRevenueAmount: 50000,

			UnitsSold: 4,

			CostedUnits: 3,

			MissingCostUnits: 1,

			GrossCOGSAmount: 21000,

			RecordedExpensesAmount: 3000,
		}

	finalizeProfitLossPoint(
		&point,
	)

	if point.ProfitComplete {
		t.Fatal(
			"expected incomplete trend point",
		)
	}

	if point.COGSCoverageBPS !=
		7500 {

		t.Fatalf(
			"unexpected trend point COGS coverage: %d",
			point.COGSCoverageBPS,
		)
	}

	if point.KnownGrossProfitAmount !=
		29000 {

		t.Fatalf(
			"unexpected known trend gross profit: %d",
			point.KnownGrossProfitAmount,
		)
	}

	if point.GrossProfitAmount != nil {
		t.Fatal(
			"trend gross profit must be null when cost coverage is incomplete",
		)
	}

	if point.NetProfitAfterRecordedExpensesAmount !=
		nil {

		t.Fatal(
			"trend net profit must be null when cost coverage is incomplete",
		)
	}
}

func warningContains(
	warnings []string,
	fragment string,
) bool {
	for _, warning := range warnings {

		if strings.Contains(
			warning,
			fragment,
		) {
			return true
		}
	}

	return false
}
