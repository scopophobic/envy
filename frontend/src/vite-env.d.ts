/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL?: string
  /** e.g. ₹299/mo — display only; charged in Razorpay */
  readonly VITE_PRICE_STARTER_LABEL?: string
  readonly VITE_PRICE_TEAM_LABEL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
