import axios, { AxiosError } from 'axios'
import type { AxiosInstance, InternalAxiosRequestConfig } from 'axios'

// Create axios instance with default config
const apiClient: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor - add common headers
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // Add timestamp to prevent caching
    config.headers['X-Request-Time'] = new Date().toISOString()
    return config
  },
  (error: AxiosError) => {
    console.error('Request interceptor error:', error)
    return Promise.reject(error)
  }
)

// Response interceptor - unified error handling
apiClient.interceptors.response.use(
  response => {
    return response
  },
  (error: AxiosError) => {
    // Handle different error scenarios
    if (error.response) {
      // Server responded with error status
      const status = error.response.status
      const message = (error.response.data as { message?: string })?.message || error.message

      switch (status) {
        case 400:
          console.error('Bad Request:', message)
          break
        case 401:
          console.error('Unauthorized:', message)
          break
        case 403:
          console.error('Forbidden:', message)
          break
        case 404:
          console.error('Not Found:', message)
          break
        case 500:
          console.error('Internal Server Error:', message)
          break
        default:
          console.error(`HTTP Error ${status}:`, message)
      }
    } else if (error.request) {
      // Request was made but no response received
      console.error('Network Error: No response from server')
    } else {
      // Error in request configuration
      console.error('Request Configuration Error:', error.message)
    }

    return Promise.reject(error)
  }
)

export default apiClient
