export const toNumber = (value) => {
  if (value === null || value === undefined || value === "") return 0;
  return Number(String(value).replace(/,/g, "")) || 0;
};

export const toFixedNumber = (value, digits = 2) =>
  toNumber(value).toFixed(digits);

export const calcProfit = (effectiveBetsTotal, profitLossTotal) =>
  toNumber(effectiveBetsTotal) - toNumber(profitLossTotal);

export const calcKillRate = (profit, effectiveBetsTotal) => {
  const chips = toNumber(effectiveBetsTotal);
  if (!chips) return "0.000";
  return (profit / chips).toFixed(3);
};
