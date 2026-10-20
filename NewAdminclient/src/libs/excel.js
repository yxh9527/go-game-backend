import * as XLSX from "xlsx";

export function exportExcel(columns, data, filename = "导出数据") {
  const headers = columns.map((item) => ({
    header: item.title,
    key: item.key,
    width: item.width ? item.width / 5 : 20,
  }));

  const workbook = XLSX.utils.book_new();
  const rows = [
    headers.map((item) => item.header),
    ...data.map((item) => headers.map((header) => item[header.key])),
  ];
  const sheet = XLSX.utils.aoa_to_sheet(rows);
  sheet["!cols"] = headers.map((item) => ({ wch: item.width }));
  XLSX.utils.book_append_sheet(workbook, sheet, "Sheet1");
  return XLSX.write(workbook, { bookType: "xlsx", type: "array" });
}

export default {
  exportExcel,
};
