import { useEffect, useState, useCallback, useRef } from 'react'
import type { Flow } from '../types/flow'

interface UseFlowStreamOptions {
  enabled?: boolean
  onFlow?: (flow: Flow) => void
  onError?: (error: Error) => void
}

export function useFlowStream({ enabled = true, onFlow, onError }: UseFlowStreamOptions = {}) {
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>()
  const reconnectAttemptsRef = useRef(0)

  const connect = useCallback(() => {
    if (!enabled || wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    try {
      // Construct WebSocket URL from current location
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const host = window.location.hostname
      const port = import.meta.env.VITE_API_PORT || '8080'
      const wsUrl = `${protocol}//${host}:${port}/api/v1/flows/stream`

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
