import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { clsx } from 'clsx'

// EmptyState is the shared component for the three distinct kinds of
// "nothing here" UI:
//
//   tone='healthy'  — emerald, reassuring. "All checks passing".
//                     Communicates "things are good" not "no data".
//   tone='filtered' — neutral. "No findings match the current filters."
//                     Communicates "your filter is too narrow", suggests
//                     widening.
//   tone='neutral'  — pre-data / setup. "No clusters connected yet" with
//                     an action CTA.
//
// Variants:
//   variant='card'   — centered icon + headline + body in a bordered
//                      box. Use when an entire panel/page section is
//                      empty.
//   variant='inline' — single-line callout. Use when one tile in a row
//                      is "empty" and you want to collapse zeros into
//                      one positive line instead.

export type EmptyStateTone = 'healthy' | 'filtered' | 'neutral'
export type EmptyStateVariant = 'card' | 'inline'

interface Props {
  tone?: EmptyStateTone
  variant?: EmptyStateVariant
  /** Optional Lucide icon. When omitted: the inline variant falls back
   *  to a small colored dot (emerald for healthy, tertiary text for the
   *  rest) so the line keeps a visual anchor; the card variant renders
   *  no leading visual. Callers pass icons that match their context
   *  (e.g. CheckCircle2 for a healthy state). */
  icon?: LucideIcon
  headline: string
  body?: ReactNode
  /** Optional CTA — usually an <a> or <button>. Card variant only. */
  action?: ReactNode
  className?: string
}

const TONE_CLASSES: Record<EmptyStateTone, { card: string; inline: string; icon: string }> = {
  healthy: {
    card: 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200',
    inline: 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200',
    icon: 'text-emerald-600 dark:text-emerald-400',
  },
  filtered: {
    card: 'border-theme-border bg-theme-surface text-theme-text-secondary',
    inline: 'border-theme-border bg-theme-surface text-theme-text-secondary',
    icon: 'text-theme-text-tertiary',
  },
  neutral: {
    card: 'border-theme-border bg-theme-surface text-theme-text-secondary',
    inline: 'border-theme-border bg-theme-surface text-theme-text-secondary',
    icon: 'text-theme-text-tertiary',
  },
}

export function EmptyState({
  tone = 'neutral',
  variant = 'card',
  icon: Icon,
  headline,
  body,
  action,
  className,
}: Props) {
  const t = TONE_CLASSES[tone]

  if (variant === 'inline') {
    return (
      <div
        className={clsx(
          'flex flex-1 min-w-[200px] items-center gap-2 rounded-md border px-3 py-2 text-sm',
          t.inline,
          className,
        )}
      >
        {Icon ? (
          <Icon className={clsx('h-4 w-4 flex-shrink-0', t.icon)} aria-hidden />
        ) : (
          // Default to a small dot for the inline variant — keeps a visual
          // anchor without forcing every caller to pick an icon.
          <span
            className={clsx(
              'inline-flex h-2 w-2 flex-shrink-0 rounded-full',
              tone === 'healthy'
                ? 'bg-emerald-500 dark:bg-emerald-400'
                : 'bg-theme-text-tertiary',
            )}
            aria-hidden
          />
        )}
        <span>
          {headline}
          {body && <span className="opacity-80"> · {body}</span>}
        </span>
      </div>
    )
  }

  return (
    <div
      className={clsx(
        'flex flex-col items-center gap-2 rounded-md border px-4 py-8 text-center',
        t.card,
        className,
      )}
    >
      {Icon && <Icon className={clsx('h-6 w-6', t.icon)} aria-hidden />}
      <div className="text-sm font-medium">{headline}</div>
      {body && <div className="max-w-md text-xs opacity-80">{body}</div>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  )
}
