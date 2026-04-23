import { useCallback, useEffect, useRef, useState } from 'react'

interface UseWakeLockOptions {
  onError?: (error: Error) => void
}

export function useWakeLock(options: UseWakeLockOptions = {}) {
  const wakeLockRef = useRef<WakeLockSentinel | null>(null)
  const [isSupported, setIsSupported] = useState(false)
  const [isActive, setIsActive] = useState(false)

  useEffect(() => {
    setIsSupported('wakeLock' in navigator)
  }, [])

  const request = useCallback(async () => {
    if (!isSupported) {
      console.log('Wake Lock not supported')
      return false
    }

    try {
      wakeLockRef.current = await navigator.wakeLock.request('screen')
      setIsActive(true)

      wakeLockRef.current.addEventListener('release', () => {
        setIsActive(false)
      })

      return true
    } catch (err) {
      console.error('Failed to request wake lock:', err)
      options.onError?.(err as Error)
      return false
    }
  }, [isSupported, options])

  const release = useCallback(async () => {
    if (wakeLockRef.current) {
      await wakeLockRef.current.release()
      wakeLockRef.current = null
      setIsActive(false)
    }
  }, [])

  useEffect(() => {
    const handleVisibilityChange = async () => {
      if (document.visibilityState === 'visible' && isActive) {
        await request()
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      release()
    }
  }, [isActive, request, release])

  return {
    isSupported,
    isActive,
    request,
    release,
  }
}
