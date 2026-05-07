import { useForm, useFieldArray, type SubmitHandler } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  createOrderSchema,
  type CreateOrderForm,
} from "@/lib/schemas";
import { formatAmount } from "@/lib/utils";

const SUGGESTED_SKUS = ["A", "B", "C", "D"];

export interface OrderFormProps {
  isSubmitting?: boolean;
  onSubmit: SubmitHandler<CreateOrderForm>;
}

export function OrderForm({ isSubmitting, onSubmit }: OrderFormProps) {
  const form = useForm<CreateOrderForm>({
    resolver: zodResolver(createOrderSchema),
    defaultValues: {
      customer_id: "c1",
      items: [{ sku: "A", quantity: 3, price: 1000 }],
    },
  });

  const { register, control, handleSubmit, watch, formState } = form;
  const { fields, append, remove } = useFieldArray({ control, name: "items" });
  const items = watch("items");
  const total = (items ?? []).reduce(
    (sum, it) => sum + (Number(it.quantity) || 0) * (Number(it.price) || 0),
    0,
  );

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      <div className="space-y-2">
        <Label htmlFor="customer_id">Customer ID</Label>
        <Input
          id="customer_id"
          placeholder="c1"
          {...register("customer_id")}
        />
        {formState.errors.customer_id && (
          <p className="text-xs text-destructive">
            {formState.errors.customer_id.message}
          </p>
        )}
      </div>

      <div>
        <div className="mb-3 flex items-center justify-between">
          <Label>Items</Label>
          <p className="text-xs text-muted-foreground">
            Suggested SKUs (seeded): {SUGGESTED_SKUS.join(", ")}. Use{" "}
            <code className="font-mono">Z</code> to force inventory.failed.
          </p>
        </div>

        <div className="space-y-3">
          {fields.map((field, index) => (
            <div
              key={field.id}
              className="grid grid-cols-1 gap-2 rounded-lg border bg-card p-3 sm:grid-cols-[1fr_120px_140px_auto]"
            >
              <div>
                <Input
                  placeholder="SKU"
                  list="sku-suggestions"
                  {...register(`items.${index}.sku`)}
                />
                {formState.errors.items?.[index]?.sku && (
                  <p className="mt-1 text-xs text-destructive">
                    {formState.errors.items[index]?.sku?.message}
                  </p>
                )}
              </div>
              <div>
                <Input
                  type="number"
                  inputMode="numeric"
                  placeholder="Qty"
                  min={1}
                  {...register(`items.${index}.quantity`)}
                />
                {formState.errors.items?.[index]?.quantity && (
                  <p className="mt-1 text-xs text-destructive">
                    {formState.errors.items[index]?.quantity?.message}
                  </p>
                )}
              </div>
              <div>
                <Input
                  type="number"
                  inputMode="numeric"
                  placeholder="Price"
                  min={0}
                  {...register(`items.${index}.price`)}
                />
                {formState.errors.items?.[index]?.price && (
                  <p className="mt-1 text-xs text-destructive">
                    {formState.errors.items[index]?.price?.message}
                  </p>
                )}
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => remove(index)}
                disabled={fields.length === 1}
                aria-label="Remove item"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))}
        </div>
        <datalist id="sku-suggestions">
          {SUGGESTED_SKUS.map((s) => (
            <option key={s} value={s} />
          ))}
        </datalist>

        <Button
          type="button"
          variant="outline"
          size="sm"
          className="mt-3"
          onClick={() => append({ sku: "", quantity: 1, price: 0 })}
        >
          <Plus className="h-4 w-4" /> Add item
        </Button>

        {formState.errors.items && !Array.isArray(formState.errors.items) && (
          <p className="mt-2 text-xs text-destructive">
            {formState.errors.items.message}
          </p>
        )}
      </div>

      <div className="flex items-center justify-between rounded-lg border bg-muted/40 px-4 py-3">
        <span className="text-sm text-muted-foreground">Total</span>
        <span className="font-mono text-base font-semibold">
          {formatAmount(total)}
        </span>
      </div>

      <Button
        type="submit"
        size="lg"
        className="w-full"
        disabled={isSubmitting || !formState.isValid}
      >
        {isSubmitting ? "Submitting…" : "Place order"}
      </Button>
    </form>
  );
}
