// Common API response wrapper
export interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
  error?: string
}

// Pagination parameters
export interface PaginationParams {
  page?: number
  pageSize?: number
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
}

// Paginated response
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

// Time range filter
export interface TimeRange {
  startTime?: string
  endTime?: string
}

// Status enum
export type Status = 'active' | 'inactive' | 'error' | 'unknown'
