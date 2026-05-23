import { forwardRef, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { ChevronDown, Check, Globe, Search, AlertTriangle } from 'lucide-react'
import { useNamespaceScope, useSetActiveNamespace } from '../api/client'
import { useToast } from './ui/Toast'
import { Tooltip } from './ui/Tooltip'

export interface NamespaceSwitcherHandle {
  open: () => void
}

interface NamespaceSwitcherProps {
  className?: string
  disabled?: boolean
  disabledTooltip?: string
}

const ALL_NAMESPACES = '__all__'

/**
 * NamespaceSwitcher is a per-user view filter for the cluster view. It does
 * NOT reshape the shared informer cache — the pick is saved server-side per
 * user and intersected with the user's RBAC-allowed namespaces on each read.
 *
 * Three states reflect what the backend reports:
 *   - cluster-wide: empty trigger label "All namespaces", picker lets the
 *     user narrow the view; otherwise informational.
 *   - namespace:    label shows the active namespace; picker offers others
 *     they have access to plus the "All namespaces" reset.
 *   - restricted:   user can't list namespaces and isn't pinned; picker
 *     surfaces only the kubeconfig context's namespace + any saved pick.
 */
export const NamespaceSwitcher = forwardRef<NamespaceSwitcherHandle, NamespaceSwitcherProps>(function NamespaceSwitcher(
  { className = '', disabled = false, disabledTooltip },
  ref,
) {
  const { data: scope, isLoading } = useNamespaceScope()
  const setActive = useSetActiveNamespace()
  const { showError } = useToast()

  const [isOpen, setIsOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [pos, setPos] = useState({ top: 0, left: 0, width: 0 })

  const triggerRef = useRef<HTMLButtonElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useImperativeHandle(ref, () => ({
    open: () => {
      if (disabled || isLoading || setActive.isPending) return
      setIsOpen(true)
    },
  }), [disabled, isLoading, setActive.isPending])

  const items = useMemo(() => {
    if (!scope) return [] as string[]
    return [...scope.accessibleNamespaces].sort((a, b) => a.localeCompare(b))
  }, [scope])

  const filteredItems = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return items
    return items.filter(n => n.toLowerCase().includes(q))
  }, [items, search])

  useEffect(() => {
    if (!isOpen) return
    const trigger = triggerRef.current
    if (!trigger) return
    const r = trigger.getBoundingClientRect()
    setPos({ top: r.bottom + 4, left: r.left, width: Math.max(r.width, 220) })
  }, [isOpen])

  useEffect(() => {
    if (!isOpen) return
    function onClick(e: MouseEvent) {
      if (
        !dropdownRef.current?.contains(e.target as Node) &&
        !triggerRef.current?.contains(e.target as Node)
      ) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [isOpen])

  if (!scope) return null

  const handleSelect = (ns: string) => {
    setIsOpen(false)
    setSearch('')
    const target = ns === ALL_NAMESPACES ? '' : ns
    if (target === scope.active) return
    setActive.mutate(
      { namespace: target },
      {
        onError: err => {
          showError('Namespace switch failed', err.message)
        },
      },
    )
  }

  const triggerLabel = scope.active === '' ? 'All namespaces' : scope.active
  const isClusterWide = scope.active === ''
  const restrictedHint = scope.mode === 'restricted'
  const isDisabled = disabled || isLoading || setActive.isPending
  const canClearAll = scope.canClearNamespace || scope.active === ''
  const tooltipContent = disabled && disabledTooltip
    ? disabledTooltip
    : restrictedHint
      ? 'Limited namespace visibility — only namespaces granted by your RBAC are shown.'
      : isClusterWide
        ? 'Currently viewing all namespaces. Click to narrow the view.'
        : `View is filtered to namespace ${scope.active}. Click to switch or reset.`

  return (
    <>
      <Tooltip
        content={tooltipContent}
        delay={300}
        position="bottom"
      >
        <button
          ref={triggerRef}
          onClick={() => !isDisabled && setIsOpen(o => !o)}
          disabled={isDisabled}
          className={`flex items-center gap-1.5 px-2 py-1 rounded text-sm bg-theme-elevated hover:bg-theme-hover text-theme-text-primary disabled:opacity-60 transition-colors ${className}`}
          aria-label="Switch active namespace"
        >
          {isClusterWide ? (
            <Globe className="w-3.5 h-3.5 text-theme-text-tertiary" />
          ) : restrictedHint ? (
            <AlertTriangle className="w-3.5 h-3.5 text-theme-text-tertiary" />
          ) : null}
          <span className="font-medium max-w-[180px] truncate">
            {setActive.isPending ? 'Switching…' : triggerLabel}
          </span>
          <ChevronDown className="w-3 h-3 opacity-60" />
        </button>
      </Tooltip>

      {isOpen &&
        createPortal(
          <div
            ref={dropdownRef}
            style={{ position: 'fixed', top: pos.top, left: pos.left, minWidth: pos.width, zIndex: 100 }}
            className="bg-theme-surface border border-theme-border rounded-md shadow-theme-lg overflow-hidden"
          >
            {items.length > 6 && (
              <div className="flex items-center gap-2 px-2 py-1.5 border-b border-theme-border">
                <Search className="w-3.5 h-3.5 text-theme-text-tertiary" />
                <input
                  autoFocus
                  value={search}
                  onChange={e => setSearch(e.target.value)}
                  placeholder="Filter namespaces"
                  className="flex-1 bg-transparent text-sm outline-none text-theme-text-primary placeholder:text-theme-text-tertiary"
                />
              </div>
            )}

            <ul className="max-h-80 overflow-y-auto py-1">
              <li>
                <button
                  onClick={() => canClearAll && handleSelect(ALL_NAMESPACES)}
                  disabled={!canClearAll}
                  className="w-full flex items-center justify-between px-3 py-1.5 text-sm hover:bg-theme-hover text-left text-theme-text-primary disabled:opacity-50 disabled:hover:bg-transparent"
                >
                  <span className="flex items-center gap-2">
                    <Globe className="w-3.5 h-3.5 text-theme-text-tertiary" />
                    All namespaces
                  </span>
                  {scope.active === '' && <Check className="w-3.5 h-3.5 text-theme-text-secondary" />}
                </button>
              </li>

              {filteredItems.length === 0 && search && (
                <li className="px-3 py-2 text-xs text-theme-text-tertiary">
                  No matches.
                </li>
              )}

              {filteredItems.map(ns => {
                const isActive = ns === scope.active
                const isContextDefault = ns === scope.kubeconfigNamespace && ns !== ''
                return (
                  <li key={ns}>
                    <button
                      onClick={() => handleSelect(ns)}
                      className="w-full flex items-center justify-between px-3 py-1.5 text-sm hover:bg-theme-hover text-left text-theme-text-primary"
                    >
                      <span className="flex items-center gap-2 min-w-0">
                        <span className="truncate">{ns}</span>
                        {isContextDefault && (
                          <span className="text-[10px] uppercase tracking-wide text-theme-text-tertiary shrink-0">
                            kubeconfig
                          </span>
                        )}
                      </span>
                      {isActive && <Check className="w-3.5 h-3.5 text-theme-text-secondary" />}
                    </button>
                  </li>
                )
              })}
            </ul>

            {!scope.authoritative && (
              <div className="px-3 py-2 border-t border-theme-border text-[11px] status-degraded">
                Limited list — your RBAC doesn&rsquo;t allow listing all
                namespaces. Other namespaces may be accessible but won&rsquo;t
                appear here until you switch context.
              </div>
            )}
          </div>,
          document.body,
        )}
    </>
  )
})
