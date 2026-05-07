import { Routes, Route } from "react-router-dom";
import { Header } from "@/components/layout/Header";

import { Dashboard } from "@/pages/Dashboard";
import { CreateOrder } from "@/pages/CreateOrder";
import { OrderDetail } from "@/pages/OrderDetail";
import { Orders } from "@/pages/Orders";
import { NotFound } from "@/pages/NotFound";

export function App() {
  return (
    <div className="relative min-h-full bg-background">
      <div className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-[420px] bg-radial-fade" />
      <div className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-[420px] bg-grid opacity-[0.35]" />
      <Header />
      <main className="py-10">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/orders/new" element={<CreateOrder />} />
          <Route path="/orders/:id" element={<OrderDetail />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </main>
    </div>
  );
}
