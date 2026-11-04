import type { GenerateParams, GenerateResult } from './types'

interface Envelope<T> {
  code: number
  message?: string
  data?: T
}

async function post<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const json = (await res.json()) as Envelope<T>
  if (json.code !== 0) {
    throw new Error(json.message || '请求失败')
  }
  return json.data as T
}

export function generateCaptcha(params: GenerateParams): Promise<GenerateResult> {
  return post<GenerateResult>('/api/generate', params)
}

export function verifyCaptcha(id: string, answer: string): Promise<{ ok: boolean }> {
  return post<{ ok: boolean }>('/api/verify', { id, answer })
}

export async function fetchFonts(): Promise<string[]> {
  const res = await fetch('/api/fonts')
  const json = (await res.json()) as Envelope<string[]>
  if (json.code !== 0 || !json.data) {
    throw new Error(json.message || '获取字体列表失败')
  }
  return json.data
}
