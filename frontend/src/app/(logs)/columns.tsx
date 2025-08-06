"use client";

import { TextWithTooltip } from "@/components/custom/text-with-tooltip";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableColumnSeverityIndicator } from "@/components/data-table/data-table-column/data-table-column-severity-indicator";
import { SEVERITY_VALUES } from "@/constants/severity";
import type { ColumnDef } from "@tanstack/react-table";
import { HoverCardTimestamp } from "./_components/hover-card-timestamp";
import type { ColumnSchema } from "./schema";

// Facility names for display
const FACILITY_NAMES = [
  "Kernel",
  "User",
  "Mail",
  "Daemon",
  "Auth",
  "Syslog",
  "LPR",
  "News",
  "UUCP",
  "Cron",
  "AuthPriv",
  "FTP",
  "NTP",
  "Audit",
  "Alert",
  "Clock",
  "Local0",
  "Local1",
  "Local2",
  "Local3",
  "Local4",
  "Local5",
  "Local6",
  "Local7",
];

export const columns: ColumnDef<ColumnSchema>[] = [
  {
    id: "severity",
    accessorKey: "severity",
    header: "",
    cell: ({ row }) => {
      const severity = row.getValue<ColumnSchema["severity"]>("severity");

      return (
        <div className="flex items-baseline gap-2">
          <DataTableColumnSeverityIndicator value={SEVERITY_VALUES[severity]} />
          <span className="font-mono text-sm">
            {" "}
            {SEVERITY_VALUES[severity]}
          </span>
          <span className="text-xs text-muted-foreground">{severity}</span>
        </div>
      );
    },
    enableHiding: false,
    enableResizing: false,
    filterFn: "arrSome",
    size: 27,
    minSize: 27,
    maxSize: 27,
    meta: {
      headerClassName:
        "w-[--header-severity-size] max-w-[--header-severity-size] min-w-[--header-severity-size]",
      cellClassName:
        "w-[--col-severity-size] max-w-[--col-severity-size] min-w-[--col-severity-size]",
    },
  },
  {
    accessorKey: "timestamp",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Timestamp" />
    ),
    cell: ({ row }) => {
      const date = new Date(
        row.getValue<ColumnSchema["timestamp"]>("timestamp"),
      );
      return <HoverCardTimestamp date={date} />;
    },
    filterFn: "inDateRange",
    enableResizing: false,
    size: 200,
    minSize: 200,
    meta: {
      headerClassName:
        "w-[--header-timestamp-size] max-w-[--header-timestamp-size] min-w-[--header-timestamp-size]",
      cellClassName:
        "font-mono w-[--col-timestamp-size] max-w-[--col-timestamp-size] min-w-[--col-timestamp-size]",
    },
  },
  {
    accessorKey: "facility",
    header: "Facility",
    cell: ({ row }) => {
      const facility = row.getValue<ColumnSchema["facility"]>("facility");
      return (
        <div className="flex items-baseline gap-2">
          <span className="font-mono text-sm">{FACILITY_NAMES[facility]}</span>
          <span className="text-xs text-muted-foreground">{facility}</span>
        </div>
      );
    },
    filterFn: "arrIncludesSome",
    enableResizing: false,
    size: 100,
    minSize: 100,
    meta: {
      headerClassName:
        "w-[--header-facility-size] max-w-[--header-facility-size] min-w-[--header-facility-size]",
      cellClassName:
        "font-mono w-[--col-facility-size] max-w-[--col-facility-size] min-w-[--col-facility-size]",
    },
  },
  {
    accessorKey: "hostname",
    header: "Hostname",
    cell: ({ row }) => {
      const value = row.getValue<ColumnSchema["hostname"]>("hostname");
      return <TextWithTooltip text={value} />;
    },
    size: 125,
    minSize: 125,
    meta: {
      cellClassName:
        "font-mono w-[--col-hostname-size] max-w-[--col-hostname-size]",
      headerClassName:
        "min-w-[--header-hostname-size] w-[--header-hostname-size]",
    },
  },
  {
    accessorKey: "appName",
    header: "App Name",
    cell: ({ row }) => {
      const value = row.getValue<ColumnSchema["appName"]>("appName");
      return <TextWithTooltip text={value} />;
    },
    size: 100,
    minSize: 100,
    meta: {
      cellClassName:
        "font-mono w-[--col-appname-size] max-w-[--col-appname-size]",
      headerClassName:
        "min-w-[--header-appname-size] w-[--header-appname-size]",
    },
  },
  {
    accessorKey: "procId",
    header: "Proc ID",
    cell: ({ row }) => {
      const value = row.getValue<ColumnSchema["procId"]>("procId");
      return <span className="font-mono">{value}</span>;
    },
    size: 80,
    minSize: 80,
    meta: {
      cellClassName:
        "font-mono w-[--col-procid-size] max-w-[--col-procid-size]",
      headerClassName: "min-w-[--header-procid-size] w-[--header-procid-size]",
    },
  },
  {
    accessorKey: "msgId",
    header: "Msg ID",
    cell: ({ row }) => {
      const value = row.getValue<ColumnSchema["msgId"]>("msgId");
      return <span className="font-mono">{value}</span>;
    },
    size: 80,
    minSize: 80,
    meta: {
      cellClassName: "font-mono w-[--col-msgid-size] max-w-[--col-msgid-size]",
      headerClassName: "min-w-[--header-msgid-size] w-[--header-msgid-size]",
    },
  },
  {
    accessorKey: "message",
    header: "Message",
    cell: ({ row }) => {
      const value = row.getValue<ColumnSchema["message"]>("message");
      return <TextWithTooltip text={value} />;
    },
    size: 300,
    minSize: 200,
    meta: {
      cellClassName:
        "font-mono w-[--col-message-size] max-w-[--col-message-size]",
      headerClassName:
        "min-w-[--header-message-size] w-[--header-message-size]",
    },
  },
  {
    accessorKey: "structuredData",
    header: "Structured Data",
    cell: ({ row }) => {
      const value =
        row.getValue<ColumnSchema["structuredData"]>("structuredData");
      if (!value || Object.keys(value).length === 0) {
        return <span className="text-muted-foreground">-</span>;
      }
      return (
        <TextWithTooltip
          text={Object.entries(value)
            .map(
              ([sdId, kvPairs]) =>
                `${sdId}:{${Object.entries(kvPairs || {})
                  .map(([k, v]) => `${k}=${v}`)
                  .join(", ")}}`,
            )
            .join(", ")}
        />
      );
    },
    size: 150,
    minSize: 150,
    meta: {
      cellClassName:
        "font-mono w-[--col-structureddata-size] max-w-[--col-structureddata-size]",
      headerClassName:
        "min-w-[--header-structureddata-size] w-[--header-structureddata-size]",
    },
  },
];
