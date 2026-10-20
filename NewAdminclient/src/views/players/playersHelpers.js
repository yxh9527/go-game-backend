import dayjs from "dayjs";

export const safeNumber = (value) => {
  if (value === null || value === undefined || value === "") return 0;
  return Number(value) || 0;
};

export const toFixedValue = (value, digits = 2) => safeNumber(value).toFixed(digits);

export const formatDateTime = (value) => {
  if (!value) return "";
  return dayjs(value * 1000).format("YYYY-MM-DD HH:mm:ss");
};

export const formatPickerDayStart = (value) =>
  dayjs(Number(value)).startOf("day").valueOf();

export const formatPickerDayEnd = (value) =>
  dayjs(Number(value)).endOf("day").valueOf();

/**
 * bet=下注(负)，award=返奖(>=0)
 * - 下注：bet
 * - 返奖：award
 */
export const resolveBillDelta = (row) => {
  if (!row) return 0;
  const award = safeNumber(row.award);
  const bet = safeNumber(row.bet);
  const desc = String(row.desc || "").trim();
  if (desc === "下注") return bet;
  if (desc === "返奖") return award;
  if (award > 0) return award;
  return bet;
};

export const resolveBillBeforeScore = (row) => {
  const after = safeNumber(row && row.currentScore);
  return after - resolveBillDelta(row);
};

export const formatBillBet = (row) => {
  if (!row) return "-";
  const desc = String(row.desc || "").trim();
  if (desc === "返奖") return "0.00";
  return toFixedValue(row.bet);
};

export const formatBillAward = (row) => {
  if (!row) return "0.00";
  return toFixedValue(row.award);
};
