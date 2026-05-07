import { NavLink, Link } from "react-router-dom";
import { Box, Github } from "lucide-react";
import { Container } from "./Container";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/", label: "Dashboard", end: true },
  { to: "/orders", label: "Orders" },
  { to: "/orders/new", label: "Create" },
];

export function Header() {
  return (
    <header className="sticky top-0 z-40 border-b bg-background/80 backdrop-blur-md">
      <Container className="flex h-16 items-center justify-between">
        <Link to="/" className="flex items-center gap-2 font-semibold">
          <span className="grid h-8 w-8 place-items-center rounded-lg bg-primary text-primary-foreground shadow-sm">
            <Box className="h-4 w-4" />
          </span>
          <span className="text-base tracking-tight">OrderFlow</span>
          <span className="hidden text-xs font-normal text-muted-foreground sm:inline">
            saga · outbox · idempotency
          </span>
        </Link>

        <nav className="flex items-center gap-1">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  "rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )
              }
            >
              {item.label}
            </NavLink>
          ))}
          <a
            href="https://github.com/liliksetyawan/orderflow"
            target="_blank"
            rel="noreferrer"
            className="ml-2 rounded-md p-2 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            aria-label="GitHub repo"
          >
            <Github className="h-4 w-4" />
          </a>
        </nav>
      </Container>
    </header>
  );
}
