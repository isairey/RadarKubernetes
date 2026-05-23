import { useState, useCallback, useRef } from 'react'

const MIN_SPIN_DURATION = 400 // ms
const SUCCESS_DISPLAY_DURATION = 1200 // ms

type RefreshPhase = 'idle' | 'spinning' | 'success'

/**
 * Hook that provides a three-phase refresh animation:
 *   idle → spinning → success (checkmark) → idle
 *
 * @param refetchFn - The actual refetch function to call
 * @returns [wrappedRefetch, phase] - A wrapped function and current animation phase
 */
export function useRefreshAnimation(refetchFn: () => void | Promise<unknown>): [() => void, boolean, RefreshPhase] {
  const [phase, setPhase] = useState<RefreshPhase>('idle')
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const wrappedRefetch = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }

    setPhase('spinning')

    const result = refetchFn()
    const startTime = Date.now()

    const showSuccess = () => {
      const elapsed = Date.now() - startTime
      const remaining = MIN_SPIN_DURATION - elapsed

      const transitionToSuccess = () => {
        setPhase('success')
        timeoutRef.current = setTimeout(() => {
          setPhase('idle')
        }, SUCCESS_DISPLAY_DURATION)
      }

      if (remaining > 0) {
        timeoutRef.current = setTimeout(transitionToSuccess, remaining)
      } else {
        transitionToSuccess()
      }
    }

    if (result instanceof Promise) {
      result.finally(showSuccess)
    } else {
      showSuccess()
    }
  }, [refetchFn])

  // isAnimating = backward compat (true when not idle)
  const isAnimating = phase !== 'idle'

  return [wrappedRefetch, isAnimating, phase]
}
