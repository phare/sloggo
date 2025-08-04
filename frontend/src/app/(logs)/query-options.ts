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

// Helper to get a cursor that ensures data will be returned
// For initial load, we use a time in the future to load the most recent logs
const getInitialCursor = () => {
  // Set cursor to 3 minutes in the future to ensure we get the most recent logs on initial load
  return new Date(Date.now() + 3 * 60 * 1000).getTime();
};

export const dataOptions = (search: SearchParamsType) => {
  // Create a stable query key that doesn't include the cursor
  // This prevents duplicate requests when only the cursor changes slightly
  const { id, live, cursor, ...stableKey } = { ...search };
  // Using object destructuring instead of delete to avoid TypeScript errors

  return infiniteQueryOptions({
    queryKey: ["data-table", searchParamsSerializer(stableKey)], // remove id/live/cursor as they would otherwise retrigger a fetch
    refetchOnMount: false, // Prevent refetch on component mount
    queryFn: async ({ pageParam }) => {
      // Ensure cursor is a valid date
      const cursor = pageParam.cursor ? new Date(pageParam.cursor) : new Date();
      const direction = pageParam.direction as "next" | "prev" | undefined;
      const serialize = searchParamsSerializer({
        ...search,
        cursor,
        direction,
        id: null,
        live: null,
      });

      // Use localhost in development, and window.location.origin in production
      const apiBaseUrl =
        process.env.NODE_ENV === "development"
          ? "http://localhost:8080"
          : window.location.origin;
      const response = await fetch(`${apiBaseUrl}/api/logs${serialize}`);
      const json = await response.json();

      // Process the JSON data to ensure dates are properly parsed
      if (json.data && Array.isArray(json.data)) {
        json.data = json.data.map(
          (item: { timestamp: string | number | Date }) => ({
            ...item,
            // Convert timestamp string to Date object
            timestamp: item.timestamp ? new Date(item.timestamp) : new Date(),
          }),
        );
      }

      return json as InfiniteQueryResponse<ColumnSchema[], SyslogMeta>;
    },
    initialPageParam: { cursor: new Date().getTime(), direction: "next" },
    getPreviousPageParam: (firstPage, _pages) => {
      // For previous page, use the previous cursor or null if it doesn't exist
      if (!firstPage.prevCursor) return null;
      return { cursor: firstPage.prevCursor, direction: "prev" };
    },
    getNextPageParam: (lastPage, _pages) => {
      // For next page, use the next cursor or null if it doesn't exist
      if (!lastPage.nextCursor) return null;
      return { cursor: lastPage.nextCursor, direction: "next" };
    },
    refetchOnWindowFocus: true, // Enable refetching on window focus to ensure latest data
    placeholderData: keepPreviousData,
    staleTime: 30000, // 30 seconds before data is considered stale
  });
};
