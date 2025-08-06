import { SEVERITY_VALUES } from "@/constants/severity";
import { getSeverityColor } from "@/lib/request/severity";
import { cn } from "@/lib/utils";

export function DataTableColumnSeverityIndicator({
  value,
  className,
}: {
  value: (typeof SEVERITY_VALUES)[number];
  className?: string;
}) {
  return (
    <div className={cn("flex items-center justify-center", className)}>
      <div
        className={cn("h-2.5 w-2.5 rounded-[2px]", getSeverityColor(value).bg)}
      />
    </div>
  );
}
