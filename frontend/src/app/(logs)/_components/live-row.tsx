import { TableCell, TableRow } from "@/components/custom/table";
import { DataTableColumnSeverityIndicator } from "@/components/data-table/data-table-column/data-table-column-severity-indicator";
import { columns } from "../columns";

export function LiveRow() {
  return (
    <TableRow>
      <TableCell className="w-[--header-severity-size] min-w-[--header-severity-size] max-w-[--header-severity-size] border-b border-l border-r border-t border-info border-r-info/50">
        <DataTableColumnSeverityIndicator value="info" />
      </TableCell>
      <TableCell
        colSpan={columns.length - 1}
        className="border-b border-r border-t border-info font-medium text-info"
      >
        Live Mode
      </TableCell>
    </TableRow>
  );
}
