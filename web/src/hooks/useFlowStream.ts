import { useEffect, useState, useCallback, useRef } from 'react'
import type { Flow } from '../types/flow'
import { apiConfig } from '../config/api'

interface UseFlowStreamOptions {
  enabled?: boolean
  onFlow?: (flow: Flow) => void
  onError?: (error: Error) => void
}

export function useFlowStream(options: UseFlowStreamOptions = {}) {
  const { enabled = true, onFlow, onError } = options
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const reconnectAttemptsRef = useRef(0)

  const connect = useCallback(() => {
    if (!enabled || wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    try {
      // Get WebSocket URL from centralized config
      const wsUrl = apiConfig.getWebSocketUrl('/v1/flows/stream')

      const ws = new WebSocket(wsUrl)

      ws.onopen = () => {
        console.log('WebSocket connected to flow stream')
        setIsConnected(true)
        setError(null)
        reconnectAttemptsRef.current = 0
      }

      ws.onmessage = event => {
        try {
          const message = JSON.parse(event.data)
          if (message.type === 'flow' && message.data) {
            onFlow?.(message.data)
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }

      ws.onerror = event => {
        console.error('WebSocket error:', event)
        const err = new Error('WebSocket connection error')
        setError(err)
        onError?.(err)
      }

      ws.onclose = () => {
        console.log('WebSocket disconnected')
        setIsConnected(false)
        wsRef.current = null

        // Auto-reconnect with exponential backoff
        if (enabled && reconnectAttemptsRef.current < 10) {
          const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000)
          reconnectAttemptsRef.current++
          console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttemptsRef.current})`)

          reconnectTimeoutRef.current = setTimeout(() => {
            connect()
          }, delay)
        }
      }

      wsRef.current = ws
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to create WebSocket')
      setError(error)
      onError?.(error)
    }
  }, [enabled, onFlow, onError])

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setIsConnected(false)
  }, [])

  useEffect(() => {
    if (enabled) {
      connect()
    } else {
      disconnect()
    }

    return () => {
      disconnect()
    }
  }, [enabled, connect, disconnect])

  return {
    isConnected,
    error,
    connect,
    disconnect,
  }
}
