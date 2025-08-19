import { useEffect, useState } from 'react'

/**
 * Hook to ensure content is only rendered on the client side
 * This helps prevent hydration mismatches when using dynamic content
 * like dates, random numbers, or browser-specific APIs
 */
export function useClientOnly() {
  const [isClient, setIsClient] = useState(false)

  useEffect(() => {
    setIsClient(true)
  }, [])

  return isClient
}

/**
 * Hook to safely use values that might differ between server and client
 * Returns the fallback value during SSR and the actual value on the client
 */
export function useClientValue<T>(getValue: () => T, fallback: T): T {
  const isClient = useClientOnly()
  const [value, setValue] = useState<T>(fallback)

  useEffect(() => {
    if (isClient) {
      setValue(getValue())
    }
  }, [isClient, getValue])

  return isClient ? value : fallback
}
