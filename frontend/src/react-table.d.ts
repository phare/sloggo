import type {
  CellData,
  Row,
  RowData,
  TableFeatures,
} from "@tanstack/react-table";

declare module "@tanstack/react-table" {
  // https://github.com/TanStack/table/issues/44#issuecomment-1377024296
  interface TableMeta<
    in out TFeatures extends TableFeatures,
    in out TData extends RowData,
  > {
    getRowClassName?: (row: Row<TFeatures, TData>) => string;
  }

  interface ColumnMeta<
    in out TFeatures extends TableFeatures,
    in out TData extends RowData,
    TValue extends CellData = CellData,
  > {
    headerClassName?: string;
    cellClassName?: string;
    label?: string;
  }
}
