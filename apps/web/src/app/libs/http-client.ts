import axios from 'axios'

const API_URL = import.meta.env.VITE_API_URL || ''

const instance = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

instance.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.data?.error) {
      err.message = err.response.data.error
    }
    return Promise.reject(err)
  },
)

export const httpClient = {
  async get<T>(url: string, config?: { params?: Record<string, unknown> }): Promise<T> {
    const res = await instance.get<T>(url, config)
    return res.data
  },

  async post<T>(url: string, data: Record<string, unknown>): Promise<T> {
    const res = await instance.post<T>(url, data)
    return res.data
  },

  async patch<T>(url: string, data: Record<string, unknown>): Promise<T> {
    const res = await instance.patch<T>(url, data)
    return res.data
  },

  async put<T>(url: string, data: Record<string, unknown>): Promise<T> {
    const res = await instance.put<T>(url, data)
    return res.data
  },

  async delete<T>(url: string): Promise<T> {
    const res = await instance.delete<T>(url)
    return res.data
  },
}
