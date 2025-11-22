/**
 * API Configuration
 *
 * Centralized configuration for API endpoints and WebSocket connections.
 * Uses Vite environment variables for flexibility across environments.
 */

// API base URL from environment variable or fallback to relative path
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

// Parse the base URL to extract components
function parseApiUrl(baseUrl: string) {
  if (!baseUrl) {
    // When no base URL is set, use current origin
    return {
      protocol: window.location.protocol,
      host: window.location.hostname,
      port: window.location.port || '8080',
      basePath: '/api',
    }
  }

  try {
    const url = new URL(baseUrl)
    return {
      protocol: url.protocol,
      host: url.hostname,
      port: url.port || (url.protocol === 'https:' ? '443' : '80'),
      basePath: url.pathname === '/' ? '/api' : url.pathname,
    }
  } catch {
    // Fallback for invalid URLs
    console.warn('Invalid VITE_API_BASE_URL, using defaults')
    return {
      protocol: window.location.protocol,
      host: window.location.hostname,
      port: '8080',
      basePath: '/api',
    }
  }
}

// Lazily parse the URL (only when needed)
let cachedConfig: ReturnType<typeof parseApiUrl> | null = null

function getConfig() {
  if (!cachedConfig) {
    cachedConfig = parseApiUrl(API_BASE_URL)
  }
  return cachedConfig
}

/**
 * Get the full HTTP API base URL
 * @returns The base URL for HTTP API requests (e.g., "http://localhost:8080/api")
 */
export function getApiBaseUrl(): string {
  const config = getConfig()
  const portPart = config.port ? `:${config.port}` : ''
  return `${config.protocol}//${config.host}${portPart}${config.basePath}`
}

/**
 * Get the WebSocket URL for a specific endpoint
 * @param path - The WebSocket endpoint path (e.g., "/v1/flows/stream")
 * @returns The full WebSocket URL
 */
export function getWebSocketUrl(path: string): string {
  const config = getConfig()
  const wsProtocol = config.protocol === 'https:' ? 'wss:' : 'ws:'
  const portPart = config.port ? `:${config.port}` : ''
  const fullPath = path.startsWith('/') ? `${config.basePath}${path}` : `${config.basePath}/${path}`
  return `${wsProtocol}//${config.host}${portPart}${fullPath}`
}

/**
 * API configuration object for direct access
 */
export const apiConfig = {
  get baseUrl() {
    return getApiBaseUrl()
  },
  get timeout() {
    return Number(import.meta.env.VITE_API_TIMEOUT) || 10000
  },
  getWebSocketUrl,
}

export default apiConfig
