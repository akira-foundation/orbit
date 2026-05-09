import { forwardRef, type ButtonHTMLAttributes } from 'react'
import { cn } from '../lib/cn'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger'
type Size = 'sm' | 'md'

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
}

const variants: Record<Variant, string> = {
  primary: 'bg-[var(--color-accent)] text-white hover:brightness-110',
  secondary: 'bg-[var(--color-surface-2)] text-[var(--color-text)] hover:bg-[var(--color-border)] ring-1 ring-[var(--color-border)]',
  ghost: 'text-[var(--color-text)] hover:bg-[var(--color-surface-2)]',
  danger: 'bg-rose-500/15 text-rose-300 hover:bg-rose-500/25 ring-1 ring-rose-500/30',
}

const sizes: Record<Size, string> = {
  sm: 'h-8 px-3 text-xs',
  md: 'h-9 px-4 text-sm',
}

export const Button = forwardRef<HTMLButtonElement, Props>(
  ({ className, variant = 'secondary', size = 'md', ...rest }, ref) => (
    <button
      ref={ref}
      className={cn(
        'inline-flex items-center justify-center gap-2 rounded-lg font-medium transition disabled:opacity-50 disabled:cursor-not-allowed',
        variants[variant],
        sizes[size],
        className,
      )}
      {...rest}
    />
  ),
)
Button.displayName = 'Button'
