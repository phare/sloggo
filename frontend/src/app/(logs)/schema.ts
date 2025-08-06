import { SEVERITY_VALUES } from "@/constants/severity";
import {
  ARRAY_DELIMITER,
  RANGE_DELIMITER,
  SLIDER_DELIMITER,
} from "@/lib/delimiters";
import { z } from "zod";

// RFC 5424 Syslog Schema
export const columnSchema = z.object({
  id: z.number().int().positive(),
  priority: z.number().min(0).max(191),
  severity: z.number().min(0).max(7),
  facility: z.number().min(0).max(23),
  timestamp: z.date(),
  hostname: z.string(),
  appName: z.string(),
  procId: z.string(),
  msgId: z.string(),
  message: z.string(),
  structuredData: z.record(z.record(z.string())).optional(),
});

export type ColumnSchema = z.infer<typeof columnSchema>;

export const columnFilterSchema = z.object({
  severity: z
    .string()
    .transform((val) => val.split(ARRAY_DELIMITER))
    .pipe(z.coerce.number().array())
    .optional(),
  facility: z
    .string()
    .transform((val) => val.split(ARRAY_DELIMITER))
    .pipe(z.coerce.number().array())
    .optional(),
  timestamp: z
    .string()
    .transform((val) => val.split(RANGE_DELIMITER).map(Number))
    .pipe(z.coerce.date().array())
    .optional(),
  hostname: z.string().optional(),
  appName: z.string().optional(),
  procId: z.string().optional(),
  msgId: z.string().optional(),
  message: z.string().optional(),
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
  ...SEVERITY_VALUES.reduce(
    (acc, severity) => ({
      ...acc,
      [severity]: z.number().default(0),
    }),
    {} as Record<(typeof SEVERITY_VALUES)[number], z.ZodNumber>,
  ),
}) satisfies z.ZodType<BaseChartSchema>;

export type TimelineChartSchema = z.infer<typeof timelineChartSchema>;
