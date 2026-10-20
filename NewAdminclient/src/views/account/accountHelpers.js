export const emptyAccountForm = () => ({
  uType: "",
  uName: "",
  account: "",
  password: "",
  ipLimit: "",
  agentId: "",
  realmName: "",
});

export const accountTypeOptions = [
  { label: "总控账号", value: 1 },
  { label: "信息账号", value: 2 },
  { label: "代理账号", value: 3 },
];

export const accountTypeText = (uType) => {
  if (uType === 1) return "总控";
  if (uType === 2) return "信息";
  if (uType === 3) return "代理";
  return "";
};
