import { configureStore } from "@reduxjs/toolkit";
import { setupListeners } from "@reduxjs/toolkit/query";

import { ordersApi } from "@/features/orders/ordersApi";
import { healthApi } from "@/features/health/healthApi";

export const store = configureStore({
  reducer: {
    [ordersApi.reducerPath]: ordersApi.reducer,
    [healthApi.reducerPath]: healthApi.reducer,
  },
  middleware: (getDefault) =>
    getDefault().concat(ordersApi.middleware, healthApi.middleware),
});

// Enables RTK Query refetchOnFocus / refetchOnReconnect.
setupListeners(store.dispatch);

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
