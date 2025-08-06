"use client";

import { CopyToClipboardContainer } from "@/components/custom/copy-to-clipboard-container";
import { KVTabs } from "@/components/custom/kv-tabs";
import type {
  DataTableFilterField,
  Option,
  SheetField,
} from "@/components/data-table/types";
import { SEVERITY_LABELS, SEVERITY_VALUES } from "@/constants/severity";
import { getSeverityColor } from "@/lib/request/severity";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import type { SyslogMeta } from "./query-options";
import { type ColumnSchema } from "./schema";

// Syslog facility names
const SYSLOG_FACILITIES = [
  { label: "Kernel messages", value: 0 },
  { label: "User-level messages", value: 1 },
  { label: "Mail system", value: 2 },
  { label: "System daemons", value: 3 },
  { label: "Security/authorization messages", value: 4 },
  { label: "Messages generated internally by syslogd", value: 5 },
  { label: "Line printer subsystem", value: 6 },
  { label: "Network news subsystem", value: 7 },
  { label: "UUCP subsystem", value: 8 },
  { label: "Clock daemon", value: 9 },
  { label: "Security/authorization messages", value: 10 },
  { label: "FTP daemon", value: 11 },
  { label: "NTP subsystem", value: 12 },
  { label: "Log audit", value: 13 },
  { label: "Log alert", value: 14 },
  { label: "Clock daemon", value: 15 },
  { label: "Local use 0", value: 16 },
  { label: "Local use 1", value: 17 },
  { label: "Local use 2", value: 18 },
  { label: "Local use 3", value: 19 },
  { label: "Local use 4", value: 20 },
  { label: "Local use 5", value: 21 },
  { label: "Local use 6", value: 22 },
  { label: "Local use 7", value: 23 },
];

export const filterFields = [
  {
    label: "Time Range",
    value: "timestamp",
    type: "timerange",
    defaultOpen: true,
    commandDisabled: true,
  },
  {
    label: "Severity",
    value: "severity",
    type: "checkbox",
    defaultOpen: true,
    options: SEVERITY_LABELS.map((label, index) => ({
      label: label,
      value: index,
    })),
    component: (props: Option) => {
      const value = props.value as number;
      return (
        <div className="flex w-full max-w-28 items-center justify-between gap-2 font-mono">
          <span className="capitalize text-foreground/70 group-hover:text-accent-foreground">
            {props.label}
          </span>
          <div className="flex items-center gap-2">
            <div
              className={cn(
                "h-2.5 w-2.5 rounded-[2px]",
                getSeverityColor(SEVERITY_VALUES[value]).bg,
              )}
            />
          </div>
        </div>
      );
    },
  },
  {
    label: "Facility",
    value: "facility",
    type: "checkbox",
    options: SYSLOG_FACILITIES,
    component: (props: Option) => {
      return <span className="font-mono">{props.label}</span>;
    },
  },
  {
    label: "Hostname",
    value: "hostname",
    type: "input",
  },
  {
    label: "App Name",
    value: "appName",
    type: "input",
  },
  {
    label: "Proc ID",
    value: "procId",
    type: "input",
  },
  {
    label: "Msg ID",
    value: "msgId",
    type: "input",
  },
] satisfies DataTableFilterField<ColumnSchema>[];

export const sheetFields = [
  {
    id: "id",
    label: "ID",
    type: "readonly",
    component: (props) => <span className="font-mono">{props.id}</span>,
    skeletonClassName: "w-16",
  },
  {
    id: "timestamp",
    label: "Timestamp",
    type: "timerange",
    component: (props) =>
      format(new Date(props.timestamp), "LLL dd, y HH:mm:ss"),
    skeletonClassName: "w-36",
  },
  {
    id: "priority",
    label: "Priority",
    type: "readonly",
    component: (props) => {
      const facility = props.facility;
      const severity = props.severity;
      const facilityNames = [
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
      return (
        <div className="flex flex-col">
          <span className="font-mono">
            {props.facility * 8 + props.severity}
          </span>
          <span className="text-sm text-muted-foreground">
            {facilityNames[facility]} ({facility}) • {SEVERITY_LABELS[severity]}{" "}
            ({severity})
          </span>
        </div>
      );
    },
    skeletonClassName: "w-16",
  },
  {
    id: "hostname",
    label: "Hostname",
    type: "input",
    skeletonClassName: "w-24",
  },
  {
    id: "appName",
    label: "App Name",
    type: "input",
    skeletonClassName: "w-20",
  },
  {
    id: "procId",
    label: "Proc ID",
    type: "input",
    skeletonClassName: "w-16",
  },
  {
    id: "msgId",
    label: "Msg ID",
    type: "input",
    skeletonClassName: "w-16",
  },
  {
    id: "structuredData",
    label: "Structured Data",
    type: "readonly",
    condition: (props) =>
      props.structuredData !== undefined &&
      Object.keys(props.structuredData).length > 0,
    component: (props) => {
      // Render a separate section for each sd-id
      return (
        <div className="mt-0.5 flex w-full flex-col gap-4">
          {Object.entries(props.structuredData || {}).map(([sdId, kvPairs]) => (
            <div key={sdId} className="flex w-full flex-col gap-1">
              <div className="text-left text-sm font-medium">{sdId}</div>
              <KVTabs data={kvPairs} className="-mt-[22px]" />
            </div>
          ))}
        </div>
      );
    },
    className: "flex-col items-start w-full gap-1",
  },
  {
    id: "message",
    label: "Message",
    type: "readonly",
    component: (props) => (
      <CopyToClipboardContainer>{props.message}</CopyToClipboardContainer>
    ),
    className: "flex-col items-start w-full gap-1",
  },
] satisfies SheetField<ColumnSchema, SyslogMeta>[];
