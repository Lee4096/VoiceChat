const API_BASE = '/api/v1'

interface ApiResponse<T> {
  data?: T
  error?: {
    code: string
    message: string
  }
}

async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const token = localStorage.getItem('token')

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  if (options.headers) {
    const existingHeaders = options.headers as Record<string, string>
    Object.assign(headers, existingHeaders)
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  try {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 5000)

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers,
      signal: controller.signal,
    })

    clearTimeout(timeoutId)

    const data = await response.json()

    if (!response.ok) {
      return { error: data.error || { code: 'UNKNOWN', message: 'Unknown error' } }
    }

    return { data }
  } catch (error) {
    if (error instanceof Error && error.name === 'AbortError') {
      return { error: { code: 'TIMEOUT', message: 'Request timeout' } }
    }
    return { error: { code: 'NETWORK', message: 'Network error' } }
  }
}

export interface LoginResponse {
  token: string
  user: {
    id: string
    email: string
    name: string
    avatar: string
    provider: string
  }
}

export interface RoomListResponse {
  rooms: Array<{
    id: string
    name: string
    owner_id: string
    created_at: string
  }>
}

export const api = {
  auth: {
    getLoginURL: async (provider: string) => {
      const res = await request<{ url: string }>(`/auth/login/${provider}`)
      return res.data?.url
    },
    callback: async (provider: string, code: string) => {
      const res = await request<LoginResponse>(`/auth/callback/${provider}?code=${code}`)
      return res.data
    },
    register: async (email: string, password: string, name: string) => {
      const res = await request<LoginResponse>('/auth/register', {
        method: 'POST',
        body: JSON.stringify({ email, password, name }),
      })
      return res
    },
    loginWithPassword: async (email: string, password: string) => {
      const res = await request<LoginResponse>('/auth/login/password', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      })
      return res
    },
  },

  rooms: {
    list: async (limit = 20, offset = 0) => {
      const res = await request<RoomListResponse>(`/rooms?limit=${limit}&offset=${offset}`)
      return res.data?.rooms || []
    },
    get: async (id: string) => {
      const res = await request<{ room: any; members: any[] }>(`/rooms/${id}`)
      return res.data
    },
    create: async (name: string) => {
      const res = await request<{ id: string; name: string; owner_id: string }>('/rooms', {
        method: 'POST',
        body: JSON.stringify({ name }),
      })
      return res.data
    },
    join: async (id: string) => {
      const res = await request<{ member: any }>(`/rooms/${id}/join`, { method: 'POST' })
      return res.data
    },
    leave: async (id: string) => {
      await request(`/rooms/${id}/leave`, { method: 'POST' })
    },
  },

  users: {
    me: async () => {
      const res = await request<{
        id: string
        email: string
        name: string
        avatar: string
        provider: string
      }>('/users/me')
      return res.data
    },
  },

  health: async () => {
    const res = await request<{ status: string }>('/health')
    return res.data
  },
}
