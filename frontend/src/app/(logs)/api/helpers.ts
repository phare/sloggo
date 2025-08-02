import { LEVELS } from "@/constants/levels";
import { isArrayOfDates, isArrayOfNumbers } from "@/lib/is-array";
import {
  calculatePercentile,
  calculateSpecificPercentile,
} from "@/lib/request/percentile";
import {
  addDays,
  addMilliseconds,
  differenceInMinutes,
  isSameDay,
} from "date-fns";
import type {
  SyslogSchema,
  FacetMetadataSchema,
  TimelineChartSchema,
} from "../schema";
import type { SearchParamsType } from "../search-params";

export const sliderFilterValues = [
  "priority",
] as const satisfies (keyof SyslogSchema)[];

export const filterValues = [
  "level",
  ...sliderFilterValues,
  "facility",
  "severity",
  "hostname",
  "appName",
  "procId",
  "msgId",
] as const satisfies (keyof SyslogSchema)[];

export function filterData(
  data: SyslogSchema[],
  search: Partial<SearchParamsType>,
): SyslogSchema[] {
  const { start, size, sort, ...filters } = search;
  return data.filter((row) => {
    for (const key in filters) {
      const filter = filters[key as keyof typeof filters];
      if (filter === undefined || filter === null) continue;
      if (
        (key === "priority") &&
        isArrayOfNumbers(filter)
      ) {
        if (filter.length === 1 && row[key] !== filter[0]) {
          return false;
        } else if (
          filter.length === 2 &&
          (row[key] < filter[0] || row[key] > filter[1])
        ) {
          return false;
        }
        return true;
      }
      if (key === "facility" && isArrayOfNumbers(filter)) {
        if (!filter.includes(row[key])) {
          return false;
        }
      }
      if (key === "severity" && isArrayOfNumbers(filter)) {
        if (!filter.includes(row[key])) {
          return false;
        }
      }
      if (key === "date" && isArrayOfDates(filter)) {
        if (filter.length === 1 && !isSameDay(row.timestamp, filter[0])) {
          return false;
        } else if (
          filter.length === 2 &&
          (row.timestamp.getTime() < filter[0].getTime() ||
            row.timestamp.getTime() > filter[1].getTime())
        ) {
          return false;
        }
      }
      if (key === "level" && Array.isArray(filter)) {
        const typedFilter = filter as unknown as (typeof LEVELS)[number][];
        if (!typedFilter.includes(row[key])) {
          return false;
        }
      }
      if (key === "hostname" && typeof filter === "string") {
        if (!row[key].toLowerCase().includes(filter.toLowerCase())) {
          return false;
        }
      }
      if (key === "appName" && typeof filter === "string") {
        if (!row[key].toLowerCase().includes(filter.toLowerCase())) {
          return false;
        }
      }
      if (key === "procId" && typeof filter === "string") {
        if (!row[key].toLowerCase().includes(filter.toLowerCase())) {
          return false;
        }
      }
      if (key === "msgId" && typeof filter === "string") {
        if (!row[key].toLowerCase().includes(filter.toLowerCase())) {
          return false;
        }
      }
    }
    return true;
  });
}

export function sortData(data: SyslogSchema[], sort: SearchParamsType["sort"]) {
  if (!sort) return data;
  return data.sort((a, b) => {
    if (sort.desc) {
      // @ts-ignore
      return a?.[sort.id] < b?.[sort.id] ? 1 : -1;
    } else {
      // @ts-ignore
      return a?.[sort.id] > b?.[sort.id] ? 1 : -1;
    }
  });
}

export function percentileData(data: SyslogSchema[]): SyslogSchema[] {
  const priorities = data.map((row) => row.priority);
  return data.map((row) => ({
    ...row,
    percentile: calculatePercentile(priorities, row.priority),
  }));
}

export function splitData(data: SyslogSchema[], search: SearchParamsType) {
  let newData: SyslogSchema[] = [];
  const now = new Date();

  // TODO: write a helper function for this
  data.forEach((item) => {
    if (search.direction === "next") {
      if (
        item.timestamp.getTime() < search.cursor.getTime() &&
        newData.length < search.size
      ) {
        newData.push(item);
        // TODO: check how to deal with the cases that there are some items left with the same date
      } else if (
        item.timestamp.getTime() === newData[newData.length - 1]?.timestamp.getTime()
      ) {
        newData.push(item);
      }
    } else if (search.direction === "prev") {
      if (
        item.timestamp.getTime() > search.cursor.getTime() &&
        // REMINDER: we need to make sure that we don't get items that are in the future which we do with mockLive data
        item.timestamp.getTime() < now.getTime()
      ) {
        newData.push(item);
      }
    }
  });

  return newData;
}

export function getFacetsFromData(data: SyslogSchema[]) {
  const valuesMap = data.reduce((prev, curr) => {
    Object.entries(curr).forEach(([key, value]) => {
      if (filterValues.includes(key as any)) {
        const _value = Array.isArray(value) ? value.toString() : value;
        const total = prev.get(key)?.get(_value) || 0;
        if (prev.has(key) && _value) {
          prev.get(key)?.set(_value, total + 1);
        } else if (_value) {
          prev.set(key, new Map([[_value, 1]]));
        }
      }
    });
    return prev;
  }, new Map<string, Map<any, number>>());

  const facets = Object.fromEntries(
    Array.from(valuesMap.entries()).map(([key, valueMap]) => {
      let min: number | undefined;
      let max: number | undefined;
      const rows = Array.from(valueMap.entries()).map(([value, total]) => {
        if (typeof value === "number") {
          if (!min) min = value;
          else min = value < min ? value : min;
          if (!max) max = value;
          else max = value > max ? value : max;
        }
        return {
          value,
          total,
        };
      });
      const total = Array.from(valueMap.values()).reduce((a, b) => a + b, 0);
      return [key, { rows, total, min, max }];
    }),
  );

  return facets satisfies Record<string, FacetMetadataSchema>;
}

export function getPercentileFromData(data: SyslogSchema[]) {
  const priorities = data.map((row) => row.priority);

  const p50 = calculateSpecificPercentile(priorities, 50);
  const p75 = calculateSpecificPercentile(priorities, 75);
  const p90 = calculateSpecificPercentile(priorities, 90);
  const p95 = calculateSpecificPercentile(priorities, 95);
  const p99 = calculateSpecificPercentile(priorities, 99);

  return { p50, p75, p90, p95, p99 };
}

export function groupChartData(
  data: SyslogSchema[],
  dates: Date[] | null,
): TimelineChartSchema[] {
  if (data?.length === 0 && !dates) return [];

  // If we only have one date, we need to add a day to it
  const _dates = dates?.length === 1 ? [dates[0], addDays(dates[0], 1)] : dates;

  const between =
    _dates || (data?.length ? [data[data.length - 1].timestamp, data[0].timestamp] : []);

  if (!between.length) return [];
  const interval = evaluateInterval(between);

  const duration = Math.abs(
    between[0].getTime() - between[between.length - 1].getTime(),
  );
  const steps = Math.floor(duration / interval);

  const timestamps: { date: Date }[] = [];

  for (let i = 0; i < steps; i++) {
    const newTimestamp = addMilliseconds(between[0], i * interval);
    timestamps.push({ date: newTimestamp });
  }

  // TODO: make it dynamic to avoid havin 200, 400, 500 hardcoded
  // TODO: make it more efficient
  // e.g. make the "status" prop we use as T generic
  return timestamps.map((timestamp, i) => {
    const filteredData = data.filter((row) => {
      const diff = row.timestamp.getTime() - timestamp.date.getTime();
      return diff < interval && diff >= 0;
    });

    return {
      timestamp: timestamp.date.getTime(), // TODO: use date-fns and interval to determine the format
      info: filteredData.filter((row) => row.level === "info").length,
      warning: filteredData.filter((row) => row.level === "warning").length,
      error: filteredData.filter((row) => row.level === "error").length,
    };
  });
}

export function evaluateInterval(dates: Date[] | null): number {
  if (!dates) return 0;
  if (dates.length < 1 || dates.length > 3) return 0;

  // Calculate the time difference in minutes
  const timeDiffInMinutes = Math.abs(differenceInMinutes(dates[0], dates[1]));

  // Define thresholds and their respective intervals in milliseconds
  const intervals = [
    { threshold: 1, interval: 1000 }, // 1 second
    { threshold: 5, interval: 5000 }, // 5 seconds
    { threshold: 10, interval: 10000 }, // 10 seconds
    { threshold: 30, interval: 30000 }, // 30 seconds
    { threshold: 60, interval: 60000 }, // 1 minute
    { threshold: 120, interval: 120000 }, // 2 minutes
    { threshold: 240, interval: 240000 }, // 4 minutes
    { threshold: 480, interval: 480000 }, // 8 minutes
    { threshold: 1440, interval: 1440000 }, // 24 minutes
    { threshold: 2880, interval: 2880000 }, // 48 minutes
    { threshold: 5760, interval: 5760000 }, // 96 minutes
    { threshold: 11520, interval: 11520000 }, // 192 minutes
    { threshold: 23040, interval: 23040000 }, // 384 minutes
  ];

  // Iterate over the intervals and return the matching one
  for (const { threshold, interval } of intervals) {
    if (timeDiffInMinutes < threshold) {
      return interval;
    }
  }

  // Default to the largest interval if no match found
  return 46080000; // 768 minutes
}
