import dayjs from "dayjs";

export const safeNumber = (value) => {
  if (value === null || value === undefined || value === "") return 0;
  if (typeof value === "object" && value.value !== undefined) return Number(value.value) || 0;
  return Number(value) || 0;
};

export const toFixedValue = (value, digits = 2) => safeNumber(value).toFixed(digits);

export const calcProfit = (effective, payout) => safeNumber(effective) - safeNumber(payout);

export const calcKill = (profit, chips) => {
  const chipsValue = safeNumber(chips);
  if (!chipsValue) return "0.000";
  return (safeNumber(profit) / chipsValue).toFixed(3);
};

export const formatDayRange = (date) => ({
  startTime: dayjs(date).startOf("day").unix(),
  endTime: dayjs(date).endOf("day").unix(),
});

export const formatDateParam = (date) => dayjs(date).format("YYYY-MM-DD");
