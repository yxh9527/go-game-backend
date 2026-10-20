export const forEach = (arr, fn) => {
  if (!Array.isArray(arr) || !fn) return;
  for (let i = 0; i < arr.length; i += 1) {
    fn(arr[i], i, arr);
  }
};

export const hasOneOf = (targetArr, arr) => {
  return targetArr.some((item) => arr.indexOf(item) > -1);
};

const getHandledValue = (num) => (num < 10 ? `0${num}` : `${num}`);

export const getDate = (timeStamp, startType = "year") => {
  if (!timeStamp) return "";
  const d = new Date(timeStamp);
  const year = d.getFullYear();
  const month = getHandledValue(d.getMonth() + 1);
  const date = getHandledValue(d.getDate());
  const hours = getHandledValue(d.getHours());
  const minutes = getHandledValue(d.getMinutes());
  const second = getHandledValue(d.getSeconds());
  if (startType === "year") {
    return `${year}-${month}-${date} ${hours}:${minutes}:${second}`;
  }
  return `${month}-${date} ${hours}:${minutes}`;
};

export const objEqual = (obj1, obj2) => {
  const keysArr1 = Object.keys(obj1 || {});
  const keysArr2 = Object.keys(obj2 || {});
  if (keysArr1.length !== keysArr2.length) return false;
  return !keysArr1.some((key) => obj1[key] !== obj2[key]);
};
