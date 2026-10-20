import dayjs from "dayjs";

export const formatUnixTime = (value) => {
  if (!value) return "";
  return dayjs(Number(value) * 1000).format("YYYY-MM-DD HH:mm:ss");
};

export const formatDateTimeValue = (value) => {
  if (!value) return "";
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
};

export const parseReceiveNames = (value) => {
  if (!value) return [];
  if (Array.isArray(value)) return value;
  return String(value)
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
};
