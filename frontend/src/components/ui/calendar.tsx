"use client"

import * as React from "react"
import { ChevronLeft, ChevronRight } from "lucide-react"
import { DayButton, DayPicker } from "react-day-picker"

import { cn } from "@/lib/utils"
import { buttonVariants } from "@/components/ui/button"

export type CalendarProps = React.ComponentProps<typeof DayPicker>

// REMINDER: since react-day-picker v9 the selection modifiers (and `aria-selected`)
// are set on the <td>, not on the button - the button styles are applied here instead
function CalendarDayButton({
  className,
  modifiers,
  ...props
}: React.ComponentProps<typeof DayButton>) {
  return (
    <DayButton
      modifiers={modifiers}
      className={cn(
        className,
        modifiers.today && "bg-accent text-accent-foreground",
        modifiers.selected &&
          "bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground",
        modifiers.range_middle &&
          "bg-accent text-accent-foreground hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground",
        modifiers.outside && "text-muted-foreground opacity-50",
        modifiers.outside &&
          modifiers.selected &&
          "bg-accent/50 text-muted-foreground opacity-30 hover:bg-accent/50 hover:text-muted-foreground focus:bg-accent/50 focus:text-muted-foreground",
        modifiers.disabled && "text-muted-foreground opacity-50"
      )}
      {...props}
    />
  )
}

function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  ...props
}: CalendarProps) {
  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      className={cn("p-3", className)}
      classNames={{
        months: "relative flex flex-col gap-4 sm:flex-row",
        month: "space-y-4",
        month_caption: "flex justify-center pt-1 items-center",
        caption_label: "text-sm font-medium",
        // REMINDER: the nav is rendered next to the months, not inside the caption
        nav: "absolute inset-x-1 top-0 flex items-center justify-between",
        button_previous: cn(
          buttonVariants({ variant: "outline" }),
          "h-7 w-7 bg-transparent p-0 opacity-50 hover:opacity-100"
        ),
        button_next: cn(
          buttonVariants({ variant: "outline" }),
          "h-7 w-7 bg-transparent p-0 opacity-50 hover:opacity-100"
        ),
        month_grid: "w-full border-collapse space-y-1",
        weekdays: "flex",
        weekday:
          "text-muted-foreground rounded-md w-9 font-normal text-[0.8rem]",
        week: "flex w-full mt-2",
        day: "h-9 w-9 text-center text-sm p-0 relative aria-selected:bg-accent data-outside:aria-selected:bg-accent/50 first:aria-selected:rounded-l-md last:aria-selected:rounded-r-md focus-within:relative focus-within:z-20",
        day_button: cn(
          buttonVariants({ variant: "ghost" }),
          "h-9 w-9 p-0 font-normal"
        ),
        range_end: "rounded-r-md",
        hidden: "invisible",
        ...classNames,
      }}
      components={{
        Chevron: ({ orientation }) =>
          orientation === "left" ? (
            <ChevronLeft className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          ),
        DayButton: CalendarDayButton,
      }}
      {...props}
    />
  )
}
Calendar.displayName = "Calendar"

export { Calendar }
