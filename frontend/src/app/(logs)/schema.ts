import {
  ARRAY_DELIMITER,
  RANGE_DELIMITER,
  SLIDER_DELIMITER,
} from "@/lib/delimiters";
import { LEVELS } from "@/constants/levels";
import { z } from "zod";

// RFC 5424 Syslog Schema
export const columnSchema = z.object({
  uuid: z.string(),
  facility: z.number().min(0).max(23),
  severity: z.number().min(0).max(7),
  priority: z.number().min(0).max(191), // Calculated: facility * 8 + severity
  timestamp: z.date(),
  hostname: z.string(),
  appName: z.string(),
  procId: z.string(),
  msgId: z.string(),
  structuredData: z.record(z.string()).optional(),
  message: z.string(),
  level: z.enum(LEVELS), // Derived from severity
});

export type ColumnSchema = z.infer<typeof columnSchema>;

// TODO: can we get rid of this in favor of nuqs search-params?
export const columnFilterSchema = z.object({
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

export type ColumnFilterSchema = z.infer<typeof columnFilterSchema>;

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
