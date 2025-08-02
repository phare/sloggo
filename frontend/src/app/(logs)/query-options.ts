import { infiniteQueryOptions, keepPreviousData } from "@tanstack/react-query";
import SuperJSON from "superjson";
import type {
  BaseChartSchema,
  ColumnSchema,
  FacetMetadataSchema,
} from "./schema";
import { searchParamsSerializer, type SearchParamsType } from "./search-params";

export type SyslogMeta = {
  // Add any specific metadata from the Go API if needed
};

export type InfiniteQueryMeta<TMeta = Record<string, unknown>> = {
  totalRowCount: number;
  filterRowCount: number;
  chartData: BaseChartSchema[];
  facets: Record<string, FacetMetadataSchema>;
  metadata?: TMeta;
  // Add any additional fields from the Go API response if needed
};

export type InfiniteQueryResponse<TData, TMeta = unknown> = {
  data: TData;
  meta: InfiniteQueryMeta<TMeta>;
  prevCursor: number | null;
  nextCursor: number | null;
};

export const dataOptions = (search: SearchParamsType) => {
  return infiniteQueryOptions({
    queryKey: [
      "data-table",
      searchParamsSerializer({ ...search, uuid: null, live: null }),
    ], // remove uuid/live as it would otherwise retrigger a fetch
    queryFn: async ({ pageParam }) => {
      const cursor = new Date(pageParam.cursor);
      const direction = pageParam.direction as "next" | "prev" | undefined;
      const serialize = searchParamsSerializer({
        ...search,
        cursor,
        direction,
        uuid: null,
        live: null,
      });
      // Use localhost in development, and window.location.origin in production
      const apiBaseUrl =
        process.env.NODE_ENV === "development"
          ? "http://localhost:8080"
          : window.location.origin;
      const response = await fetch(`${apiBaseUrl}/api/logs${serialize}`);
      const json = await response.json();
      return SuperJSON.parse<InfiniteQueryResponse<ColumnSchema[], SyslogMeta>>(
        json,
      );
    },
    initialPageParam: { cursor: new Date().getTime(), direction: "next" },
    getPreviousPageParam: (firstPage, _pages) => {
      if (!firstPage.prevCursor) return null;
      return { cursor: firstPage.prevCursor, direction: "prev" };
    },
    getNextPageParam: (lastPage, _pages) => {
      if (!lastPage.nextCursor) return null;
      return { cursor: lastPage.nextCursor, direction: "next" };
    },
    refetchOnWindowFocus: true, // Enable refetching on window focus to ensure latest data
    placeholderData: keepPreviousData,
    staleTime: 30000, // 30 seconds before data is considered stale
  });
};
