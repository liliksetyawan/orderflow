import { Link } from "react-router-dom";
import { Home } from "lucide-react";
import { Container } from "@/components/layout/Container";
import { Button } from "@/components/ui/button";

export function NotFound() {
  return (
    <Container className="flex flex-col items-center justify-center gap-6 py-24 text-center">
      <p className="font-mono text-6xl font-semibold tracking-tight text-primary">
        404
      </p>
      <div className="space-y-1">
        <h1 className="text-xl font-semibold">Page not found</h1>
        <p className="text-sm text-muted-foreground">
          The path you're looking for isn't part of OrderFlow.
        </p>
      </div>
      <Button asChild>
        <Link to="/">
          <Home className="h-4 w-4" />
          Back to dashboard
        </Link>
      </Button>
    </Container>
  );
}
