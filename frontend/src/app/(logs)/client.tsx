"use client";

import { log } from "node:console";
import { SEVERITY_VALUES } from "@/constants/severity";
import { useHotKey } from "@/hooks/use-hot-key";
import { getSeverityRowClassName } from "@/lib/request/severity";
import { cn } from "@/lib/utils";
import { useInfiniteQuery } from "@tanstack/react-query";
import type { Table as TTable } from "@tanstack/react-table";
import { useQueryState, useQueryStates } from "nuqs";
import * as React from "react";
import { LiveRow } from "./_components/live-row";
import { columns } from "./columns";
import { filterFields as defaultFilterFields, sheetFields } from "./constants";
import { DataTableInfinite } from "./data-table-infinite";
import { dataOptions } from "./query-options";
import type { FacetMetadataSchema } from "./schema";
import { searchParamsParser } from "./search-params";

export function Client() {
  const [search, setSearch] = useQueryStates(searchParamsParser);
  const {
    data,
    isFetching,
    isLoading,
    fetchNextPage,
    hasNextPage,
    fetchPreviousPage,
    refetch,
  } = useInfiniteQuery(dataOptions(search));
  useResetFocus();

  const flatData = React.useMemo(
    () => data?.pages?.flatMap((page) => page.data ?? []) ?? [],
    [data?.pages],
  );

  const liveMode = useLiveMode(flatData);

  // REMINDER: meta data is always the same for all pages as filters do not change(!)
  const lastPage = data?.pages?.[data?.pages.length - 1];
  const totalDBRowCount = lastPage?.meta?.totalRowCount;
  const filterDBRowCount = lastPage?.meta?.filterRowCount;
  const metadata = lastPage?.meta?.metadata;
  const chartData = lastPage?.meta?.chartData;
  const facets = lastPage?.meta?.facets;
  const totalFetched = flatData?.length;

  const { sort, start, size, id, cursor, direction, live, ...filter } = search;

  const filterFields = React.useMemo(() => {
    return defaultFilterFields.map((field) => {
      const facetsField = facets?.[field.value];
      if (!facetsField) return field;
      if (field.options && field.options.length > 0) return field;

      const options = facetsField.rows.map(({ value }) => {
        return {
          label: `${value}`,
          value,
        };
      });

      return { ...field, options };
    });
  }, [facets]);

  return (
    <DataTableInfinite
      columns={columns}
      data={flatData}
      totalRows={totalDBRowCount}
      filterRows={filterDBRowCount}
      totalRowsFetched={totalFetched}
      defaultColumnFilters={Object.entries(filter)
        .map(([key, value]) => ({
          id: key,
          value,
        }))
        .filter(({ value }) => value ?? undefined)}
      defaultColumnSorting={sort ? [sort] : undefined}
      defaultRowSelection={id ? { [id]: true } : undefined}
      defaultColumnVisibility={{}}
      meta={metadata}
      filterFields={filterFields}
      sheetFields={sheetFields}
      isFetching={isFetching}
      isLoading={isLoading}
      fetchNextPage={fetchNextPage}
      hasNextPage={hasNextPage}
      fetchPreviousPage={fetchPreviousPage}
      refetch={refetch}
      chartData={chartData}
      chartDataColumnId="timestamp"
      getRowClassName={(row) => {
        const rowTimestamp = row.original.timestamp.getTime();
        const isPast = rowTimestamp <= (liveMode.timestamp || -1);
        const severityClassName = getSeverityRowClassName(
          SEVERITY_VALUES[row.original.severity],
        );
        return cn(severityClassName, isPast ? "opacity-50" : "opacity-100");
      }}
      getRowId={(row) => String(row.id)}
      getFacetedUniqueValues={getFacetedUniqueValues(facets)}
      getFacetedMinMaxValues={getFacetedMinMaxValues(facets)}
      renderLiveRow={(props) => {
        if (!liveMode.timestamp) return null;
        if (!liveMode?.row || props?.row.original.id !== liveMode?.row.id)
          return null;
        return <LiveRow />;
      }}
      renderSheetTitle={(props) => props.row?.original.message}
      searchParamsParser={searchParamsParser}
    />
  );
}

function useResetFocus() {
  useHotKey(() => {
    // FIXME: some dedicated div[tabindex="0"] do not auto-unblur (e.g. the DataTableFilterResetButton)
    // REMINDER: we cannot just document.activeElement?.blur(); as the next tab will focus the next element in line,
    // which is not what we want. We want to reset entirely.
    document.body.setAttribute("tabindex", "0");
    document.body.focus();
    document.body.removeAttribute("tabindex");
  }, ".");
}

export function useLiveMode<TData extends { timestamp: Date; id: number }>(
  data: TData[],
) {
  const [live] = useQueryState("live", searchParamsParser.live);
  // REMINDER: used to capture the live mode on timestamp
  const liveTimestamp = React.useRef<number | undefined>(
    live ? Date.now() : undefined,
  );

  React.useEffect(() => {
    if (live) liveTimestamp.current = Date.now();
    else liveTimestamp.current = undefined;
  }, [live]);

  const anchorRow = React.useMemo(() => {
    if (!live) return undefined;

    const item = data.find((item) => {
      // return first item that is there if not liveTimestamp
      if (!liveTimestamp.current) return true;
      // return first item that is after the liveTimestamp
      if (item.timestamp.getTime() > liveTimestamp.current) return false;
      return true;
      // return first item if no liveTimestamp
    });

    return item;
  }, [live, data]);

  return { row: anchorRow, timestamp: liveTimestamp.current };
}

export function getFacetedUniqueValues<TData>(
  facets?: Record<string, FacetMetadataSchema>,
) {
  return (_: TTable<TData>, columnId: string): Map<string, number> => {
    return new Map(
      facets?.[columnId]?.rows?.map(({ value, total }) => [value, total]) || [],
    );
  };
}

export function getFacetedMinMaxValues<TData>(
  facets?: Record<string, FacetMetadataSchema>,
) {
  return (_: TTable<TData>, columnId: string): [number, number] | undefined => {
    const min = facets?.[columnId]?.min;
    const max = facets?.[columnId]?.max;
    if (typeof min === "number" && typeof max === "number") return [min, max];
    if (typeof min === "number") return [min, min];
    if (typeof max === "number") return [max, max];
    return undefined;
  };
}
