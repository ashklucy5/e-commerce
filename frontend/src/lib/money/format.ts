export function formatMoney(
  amountMinor: number,
  currency: string,
): string {
  const amount = amountMinor / 100;

  if (currency === "BDT") {
    return `৳${new Intl.NumberFormat(
      "en-BD",
      {
        minimumFractionDigits: 0,
        maximumFractionDigits: 2,
      },
    ).format(amount)}`;
  }

  return new Intl.NumberFormat(
    "en",
    {
      style: "currency",
      currency,
      minimumFractionDigits: 0,
      maximumFractionDigits: 2,
    },
  ).format(amount);
}