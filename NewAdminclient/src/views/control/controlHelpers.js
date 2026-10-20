import dayjs from "dayjs";

export const formatDateTime = (value) => {
  if (!value) return "";
  const time = String(value).length === 10 ? Number(value) * 1000 : Number(value);
  return dayjs(time).format("YYYY-MM-DD HH:mm:ss");
};

export const toPercentValue = (value) => {
  if (value === null || value === undefined || value === "") return "0%";
  return `${value}%`;
};

export const toAmount = (value, digits = 2) => {
  if (value === null || value === undefined || value === "") return "0.00";
  return Number(value).toFixed(digits);
};

export const triggerTypeLabel = (value) => {
  if (Number(value) === 1) return "增加";
  if (Number(value) === 2) return "消耗";
  return "-";
};
