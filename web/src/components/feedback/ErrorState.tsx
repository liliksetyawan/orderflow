import { AlertCircle } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";

export function ErrorState({
  title = "Something went wrong",
  message,
}: {
  title?: string;
  message?: string;
}) {
  return (
    <Card>
      <CardContent className="flex items-start gap-3 p-6">
        <AlertCircle className="mt-0.5 h-5 w-5 text-destructive" />
        <div>
          <p className="font-medium">{title}</p>
          {message && (
            <p className="mt-1 text-sm text-muted-foreground">{message}</p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
