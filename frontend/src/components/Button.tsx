import type { ButtonHTMLAttributes, ReactNode } from 'react'

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
  loading?: boolean
  icon?: ReactNode
}

const variants = {
  primary: 'border-slate-950 bg-slate-950 text-white shadow-[0_1px_2px_rgba(15,23,42,.25),0_6px_16px_rgba(15,23,42,.12)] hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-[0_2px_4px_rgba(15,23,42,.22),0_10px_24px_rgba(15,23,42,.16)] active:translate-y-0 active:scale-[.98]',
  secondary: 'border-slate-200 bg-white text-slate-700 shadow-sm hover:-translate-y-0.5 hover:border-slate-300 hover:bg-slate-50 hover:shadow-md active:translate-y-0 active:scale-[.98]',
  danger: 'border-red-200 bg-white text-red-600 shadow-sm hover:-translate-y-0.5 hover:bg-red-50 hover:shadow-md active:translate-y-0 active:scale-[.98]',
  ghost: 'border-transparent bg-transparent text-slate-600 hover:bg-slate-100 hover:text-slate-900 active:scale-[.98]',
}

export function Button(props: ButtonProps) {
  const { className = '', variant = 'primary', loading = false, icon, children, disabled, ...rest } = props
  return (
    <button
      className={`inline-flex min-h-9 items-center justify-center gap-2 rounded-lg border px-4 py-2 text-sm font-semibold transition-all duration-200 focus:outline-none disabled:pointer-events-none disabled:translate-y-0 disabled:opacity-55 ${variants[variant]} ${className}`}
      disabled={disabled || loading}
      {...rest}
    >
      {loading ? (
        <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true" />
      ) : icon}
      {children}
    </button>
  )
}
