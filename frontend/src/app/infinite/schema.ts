import {
  ARRAY_DELIMITER,
  RANGE_DELIMITER,
  SLIDER_DELIMITER,
} from "@/lib/delimiters";
import { LEVELS } from "@/constants/levels";
import { z } from "zod";

// https://github.com/colinhacks/zod/issues/2985#issue-2008642190
const stringToBoolean = z
  .string()
  .toLowerCase()
  .transform((val) => {
    try {
      return JSON.parse(val);
    } catch (e) {
      console.log(e);
      return undefined;
    }
  })
  .pipe(z.boolean().optional());

// RFC 5424 Syslog Schema
export const syslogSchema = z.object({
  uuid: z.string(),
  priority: z.number().min(0).max(191), // PRI = facility * 8 + severity
  version: z.number().min(1).max(2), // Version of syslog protocol
  timestamp: z.date(),
  hostname: z.string(),
  appName: z.string(),
  procId: z.string(),
  msgId: z.string(),
  structuredData: z.record(z.string()).optional(),
  message: z.string(),
  level: z.enum(LEVELS), // Derived from priority severity
  facility: z.number().min(0).max(23), // Derived from priority
  severity: z.number().min(0).max(7), // Derived from priority
  percentile: z.number().optional(), // Added by percentileData function
});

export type SyslogSchema = z.infer<typeof syslogSchema>;

// TODO: can we get rid of this in favor of nuqs search-params?
export const syslogFilterSchema = z.object({
  level: z
    .string()
    .transform((val) => val.split(ARRAY_DELIMITER))
    .pipe(z.enum(LEVELS).array())
    .optional(),
  hostname: z.string().optional(),
  appName: z.string().optional(),
  procId: z.string().optional(),
  msgId: z.string().optional(),
  facility: z
    .string()
    .transform((val) => val.split(ARRAY_DELIMITER))
    .pipe(z.coerce.number().array())
    .optional(),
  severity: z
    .string()
    .transform((val) => val.split(ARRAY_DELIMITER))
    .pipe(z.coerce.number().array())
    .optional(),
  priority: z
    .string()
    .transform((val) => val.split(SLIDER_DELIMITER))
    .pipe(z.coerce.number().array().max(2))
    .optional(),
  date: z
    .string()
    .transform((val) => val.split(RANGE_DELIMITER).map(Number))
    .pipe(z.coerce.date().array())
    .optional(),
});

export type SyslogFilterSchema = z.infer<typeof syslogFilterSchema>;

export const facetMetadataSchema = z.object({
  rows: z.array(z.object({ value: z.any(), total: z.number() })),
  total: z.number(),
  min: z.number().optional(),
  max: z.number().optional(),
});

export type FacetMetadataSchema = z.infer<typeof facetMetadataSchema>;

export type BaseChartSchema = { timestamp: number; [key: string]: number };

export const timelineChartSchema = z.object({
  timestamp: z.number(), // UNIX
  ...LEVELS.reduce(
    (acc, level) => ({
      ...acc,
      [level]: z.number().default(0),
    }),
    {} as Record<(typeof LEVELS)[number], z.ZodNumber>
  ),
  // REMINDER: make sure to have the `timestamp` field in the object
}) satisfies z.ZodType<BaseChartSchema>;

export type TimelineChartSchema = z.infer<typeof timelineChartSchema>;

// Legacy type aliases for backward compatibility
export type ColumnSchema = SyslogSchema;
export type ColumnFilterSchema = SyslogFilterSchema;
