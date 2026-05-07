import { motion } from "framer-motion";
import { SagaStep, type StepState } from "./SagaStep";
import type { OrderStatus } from "@/lib/types";

interface Step {
  key: string;
  title: string;
  detail?: string;
  state: StepState;
}

type StepTemplate = Omit<Step, "state">;

const HAPPY_STEPS: StepTemplate[] = [
  { key: "created", title: "Order Created", detail: "Inserted with status PENDING + 2 outbox rows" },
  { key: "pay-cmd", title: "Payment Authorize", detail: "Command published to RabbitMQ" },
  { key: "pay-ok", title: "Payment Authorized", detail: "Mock gateway charged the customer" },
  { key: "inv-cmd", title: "Inventory Reserve", detail: "Command published" },
  { key: "inv-ok", title: "Inventory Reserved", detail: "Stock decremented atomically" },
  { key: "confirmed", title: "Order Confirmed", detail: "Saga complete" },
];

/**
 * Derive the visual saga steps from current order status. The API only
 * returns one terminal status, so for CANCELED we present a generic
 * compensation view instead of trying to guess where it failed.
 */
export function buildSteps(status: OrderStatus): Step[] {
  const happy = HAPPY_STEPS;

  if (status === "CONFIRMED") {
    return happy.map((s) => ({ ...s, state: "done" as const }));
  }

  if (status === "PENDING") {
    return happy.map((s, i) =>
      i === 0
        ? { ...s, state: "done" as const }
        : i === 1
          ? { ...s, state: "active" as const }
          : { ...s, state: "pending" as const },
    );
  }

  if (status === "AUTHORIZED") {
    return happy.map((s, i) =>
      i <= 2
        ? { ...s, state: "done" as const }
        : i === 3
          ? { ...s, state: "active" as const }
          : { ...s, state: "pending" as const },
    );
  }

  // CANCELED — we don't know which step failed from the API, so present a
  // generic compensation view. The timeline ends at the cancellation.
  return [
    ...happy.slice(0, 1).map((s) => ({ ...s, state: "done" as const })),
    ...happy.slice(1, 2).map((s) => ({ ...s, state: "done" as const })),
    {
      key: "failed",
      title: "Saga Compensated",
      detail: "Either payment.failed or inventory.failed triggered rollback",
      state: "failed" as const,
    },
    {
      key: "canceled",
      title: "Order Canceled",
      detail: "Compensating events emitted, terminal state reached",
      state: "failed" as const,
    },
  ];
}

export function SagaTimeline({ status }: { status: OrderStatus }) {
  const steps = buildSteps(status);
  return (
    <motion.ol
      layout
      className="relative"
      initial="hidden"
      animate="visible"
    >
      {steps.map((s, i) => (
        <SagaStep
          key={s.key}
          index={i}
          title={s.title}
          detail={s.detail}
          state={s.state}
          isLast={i === steps.length - 1}
        />
      ))}
    </motion.ol>
  );
}
