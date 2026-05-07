import { motion } from "framer-motion";
import { Check, X, Loader2, Clock } from "lucide-react";
import { cn } from "@/lib/utils";

export type StepState = "done" | "active" | "pending" | "failed" | "skipped";

export interface SagaStepProps {
  index: number;
  title: string;
  detail?: string;
  state: StepState;
  isLast?: boolean;
}

const dotStyles: Record<StepState, string> = {
  done: "bg-success text-success-foreground border-success",
  active:
    "bg-primary text-primary-foreground border-primary shadow-[0_0_0_6px_hsl(var(--primary)/0.18)]",
  pending: "bg-background text-muted-foreground border-border",
  failed: "bg-destructive text-destructive-foreground border-destructive",
  skipped: "bg-muted text-muted-foreground border-border",
};

const lineStyles: Record<StepState, string> = {
  done: "bg-success",
  active: "bg-gradient-to-b from-primary to-border",
  pending: "bg-border",
  failed: "bg-destructive",
  skipped: "bg-border",
};

export function SagaStep({
  index,
  title,
  detail,
  state,
  isLast,
}: SagaStepProps) {
  const Icon =
    state === "done"
      ? Check
      : state === "failed"
        ? X
        : state === "active"
          ? Loader2
          : Clock;

  return (
    <motion.li
      layout
      initial={{ opacity: 0, x: -8 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.25, delay: index * 0.04 }}
      className="relative flex gap-4 pb-6 last:pb-0"
    >
      {!isLast && (
        <span
          aria-hidden
          className={cn(
            "absolute left-[15px] top-9 bottom-0 w-0.5 rounded",
            lineStyles[state],
          )}
        />
      )}

      <div
        className={cn(
          "relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-2 transition-all",
          dotStyles[state],
        )}
      >
        {state === "active" ? (
          <Icon className="h-4 w-4 animate-spin" />
        ) : (
          <Icon className="h-4 w-4" />
        )}
      </div>

      <div className="min-w-0 flex-1 pt-0.5">
        <p
          className={cn(
            "text-sm font-medium leading-tight",
            state === "pending" && "text-muted-foreground",
            state === "skipped" && "line-through text-muted-foreground",
          )}
        >
          {title}
        </p>
        {detail && (
          <p className="mt-1 text-xs text-muted-foreground leading-snug">
            {detail}
          </p>
        )}
      </div>
    </motion.li>
  );
}
