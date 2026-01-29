import { httpClient } from '../libs'
import type { UrlModel } from '../models'

export const urlRepository = {
  create(params: { originalUrl: string }) {
    return httpClient.post<UrlModel>('/api/urls', params)
  },
}
