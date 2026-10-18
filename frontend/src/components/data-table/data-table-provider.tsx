import { DataTableFilterField } from "@/components/data-table/types";
import type { DataTableFeatures } from "@/lib/table/features";
import type {
  ColumnDef,
  ColumnFiltersState,
  ColumnVisibilityState,
  PaginationState,
  ReactTable,
  RowData,
  RowSelectionState,
  SortingState,
} from "@tanstack/react-table";
import { createContext, useContext, useMemo } from "react";
import { ControlsProvider } from "../../providers/controls";

// REMINDER: read about how to move controlled state out of the useReactTable hook
// https://github.com/TanStack/table/discussions/4005#discussioncomment-7303569

interface DataTableStateContextType {
  columnFilters: ColumnFiltersState;
  sorting: SortingState;
  rowSelection: RowSelectionState;
  columnOrder: string[];
  columnVisibility: ColumnVisibilityState;
  pagination: PaginationState;
  enableColumnOrdering: boolean;
}

interface DataTableBaseContextType<TData extends RowData = RowData> {
  table: ReactTable<DataTableFeatures, TData>;
  filterFields: DataTableFilterField<TData>[];
  columns: ColumnDef<DataTableFeatures, TData>[];
  isLoading?: boolean;
  getFacetedUniqueValues?: (
    table: ReactTable<DataTableFeatures, TData>,
    columnId: string,
  ) => Map<string, number>;
  getFacetedMinMaxValues?: (
    table: ReactTable<DataTableFeatures, TData>,
    columnId: string,
  ) => undefined | [number, number];
}

interface DataTableContextType<TData extends RowData = RowData>
  extends DataTableStateContextType,
    DataTableBaseContextType<TData> {}

export const DataTableContext = createContext<DataTableContextType<any> | null>(
  null,
);

export function DataTableProvider<TData extends RowData>({
  children,
  ...props
}: Partial<DataTableStateContextType> &
  DataTableBaseContextType<TData> & {
    children: React.ReactNode;
  }) {
  const value = useMemo(
    () => ({
      ...props,
      columnFilters: props.columnFilters ?? [],
      sorting: props.sorting ?? [],
      rowSelection: props.rowSelection ?? {},
      columnOrder: props.columnOrder ?? [],
      columnVisibility: props.columnVisibility ?? {},
      pagination: props.pagination ?? { pageIndex: 0, pageSize: 10 },
      enableColumnOrdering: props.enableColumnOrdering ?? false,
    }),
    [
      props.columnFilters,
      props.sorting,
      props.rowSelection,
      props.columnOrder,
      props.columnVisibility,
      props.pagination,
      props.table,
      props.filterFields,
      props.columns,
      props.enableColumnOrdering,
      props.isLoading,
      props.getFacetedUniqueValues,
      props.getFacetedMinMaxValues,
    ],
  );

  return (
    <DataTableContext.Provider value={value as DataTableContextType}>
      <ControlsProvider>{children}</ControlsProvider>
    </DataTableContext.Provider>
  );
}

export function useDataTable<TData extends RowData = RowData>() {
  const context = useContext(DataTableContext);

  if (!context) {
    throw new Error("useDataTable must be used within a DataTableProvider");
  }

  return context as DataTableContextType<TData>;
}
