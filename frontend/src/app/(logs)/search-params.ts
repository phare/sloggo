import { SEVERITY_VALUES } from "@/constants/severity";
// Note: import from 'nuqs/server' to avoid the "use client" directive
import {
  ARRAY_DELIMITER,
  RANGE_DELIMITER,
  SLIDER_DELIMITER,
  SORT_DELIMITER,
} from "@/lib/delimiters";
import {
  createParser,
  createSearchParamsCache,
  createSerializer,
  parseAsArrayOf,
  parseAsBoolean,
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
  parseAsTimestamp,
  type inferParserType,
} from "nuqs/server";

// https://logs.run/i?sort=priority.desc

export const parseAsSort = createParser({
  parse(queryValue) {
    const [id, desc] = queryValue.split(SORT_DELIMITER);
    if (!id && !desc) return null;
    return { id, desc: desc === "desc" };
  },
  serialize(value) {
    return `${value.id}.${value.desc ? "desc" : "asc"}`;
  },
});

export const searchParamsParser = {
  // CUSTOM FILTERS
  facility: parseAsArrayOf(parseAsInteger, ARRAY_DELIMITER),
  severity: parseAsArrayOf(parseAsInteger, ARRAY_DELIMITER),
  hostname: parseAsArrayOf(parseAsString, ARRAY_DELIMITER),
  appName: parseAsArrayOf(parseAsString, ARRAY_DELIMITER),
  procId: parseAsArrayOf(parseAsString, ARRAY_DELIMITER),
  msgId: parseAsArrayOf(parseAsString, ARRAY_DELIMITER),
  timestamp: parseAsArrayOf(parseAsTimestamp, RANGE_DELIMITER),
  // REQUIRED FOR SORTING & PAGINATION
  // Opaque keyset cursor returned by the API (must be passed back verbatim)
  cursor: parseAsString,
  sort: parseAsSort,
  size: parseAsInteger.withDefault(40),
  start: parseAsInteger.withDefault(0),
  // REQUIRED FOR INFINITE SCROLLING (Live Mode and Load More)
  direction: parseAsStringLiteral(["prev", "next"]).withDefault("next"),
  live: parseAsBoolean.withDefault(false),
  id: parseAsInteger,
};

export const searchParamsCache = createSearchParamsCache(searchParamsParser);

export const searchParamsSerializer = createSerializer(searchParamsParser);

export type SearchParamsType = inferParserType<typeof searchParamsParser>;
