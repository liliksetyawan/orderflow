import { Link } from "react-router-dom";
import { ArrowRight, Sparkles, Workflow, ShieldCheck } from "lucide-react";

import { Container } from "@/components/layout/Container";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { HealthGrid } from "@/components/health/HealthGrid";
import { RecentOrdersList } from "@/components/orders/RecentOrdersList";

const FEATURES = [
  {
    icon: Workflow,
    title: "Saga orchestration",
    body: "Order service drives a 4-service saga with built-in compensation when payment declines or stock is short.",
  },
  {
    icon: ShieldCheck,
    title: "3-layer idempotency",
    body: "Redis fast-skip → DB UNIQUE constraint → domain state guards. Safe under at-least-once delivery.",
  },
  {
    icon: Sparkles,
    title: "Live timeline",
    body: "RTK Query polls until terminal state — every saga step animates in as the backend moves.",
  },
] as const;

export function Dashboard() {
  return (
    <Container className="space-y-10">
      {/* Hero */}
      <section className="grid gap-8 lg:grid-cols-[1.2fr_1fr] lg:items-center">
        <div className="space-y-5">
          <span className="inline-flex items-center gap-1.5 rounded-full border bg-card px-3 py-1 text-xs font-medium text-muted-foreground">
            <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-success" />
            Live demo · saga + outbox + idempotency
          </span>

          <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
            Watch a distributed transaction{" "}
            <span className="bg-gradient-to-br from-primary to-fuchsia-500 bg-clip-text text-transparent">
              actually work
            </span>
            .
          </h1>

          <p className="max-w-xl text-base text-muted-foreground">
            OrderFlow is a four-service Go backend that demonstrates production
            saga patterns. Place an order, then watch the timeline update as
            payment authorizes and inventory reserves — or compensates.
          </p>

          <div className="flex flex-wrap gap-3 pt-2">
            <Button size="lg" asChild>
              <Link to="/orders/new">
                Place an order
                <ArrowRight className="h-4 w-4" />
              </Link>
            </Button>
            <Button size="lg" variant="outline" asChild>
              <Link to="/orders">Browse all orders</Link>
            </Button>
          </div>
        </div>

        <Card className="overflow-hidden">
          <CardHeader>
            <CardTitle>Service health</CardTitle>
          </CardHeader>
          <CardContent>
            <HealthGrid />
            <p className="mt-4 text-xs text-muted-foreground">
              Polled every 5 seconds via{" "}
              <code className="font-mono">GET /health</code> on each service.
            </p>
          </CardContent>
        </Card>
      </section>

      {/* Features strip */}
      <section className="grid gap-4 sm:grid-cols-3">
        {FEATURES.map((f) => (
          <Card key={f.title}>
            <CardContent className="space-y-3 p-6">
              <span className="inline-flex h-9 w-9 items-center justify-center rounded-lg bg-accent text-accent-foreground">
                <f.icon className="h-4 w-4" />
              </span>
              <p className="font-medium">{f.title}</p>
              <p className="text-sm text-muted-foreground">{f.body}</p>
            </CardContent>
          </Card>
        ))}
      </section>

      {/* Recent orders */}
      <section>
        <RecentOrdersList />
      </section>
    </Container>
  );
}
