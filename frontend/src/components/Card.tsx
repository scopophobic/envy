import type { ReactNode } from 'react'

export function Card({ children, className = '', interactive = false }: { children: ReactNode; className?: string; interactive?: boolean }) {
  return (
    <div className={`rounded-2xl border border-slate-200/80 bg-white/95 p-5 shadow-[0_1px_2px_rgba(15,23,42,.035),0_8px_30px_rgba(15,23,42,.035)] backdrop-blur-sm ${interactive ? 'transition-all duration-200 hover:-translate-y-0.5 hover:border-slate-300 hover:shadow-[0_3px_8px_rgba(15,23,42,.06),0_16px_40px_rgba(15,23,42,.07)]' : ''} ${className}`}>
      {children}
    </div>
  )
}
