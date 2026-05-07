/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_ORDER_BASE_URL?: string;
  readonly VITE_PAYMENT_BASE_URL?: string;
  readonly VITE_INVENTORY_BASE_URL?: string;
  readonly VITE_NOTIFICATION_BASE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
