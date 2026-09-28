import { ElMessage } from 'element-plus'

// 统一 API 封装：开发走 vite 代理，生产同源直连
export async function api(path, opts = {}) {
  const res = await fetch('/api' + path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  })
  let body = null
  const text = await res.text()
  if (text) {
    try { body = JSON.parse(text) } catch { body = null }
  }
  if (!res.ok) {
    const msg = (body && body.error) || `请求失败 (${res.status})`
    ElMessage.error(msg)
    throw new Error(msg)
  }
  return body
}

export const get = (p) => api(p)
export const post = (p, data) => api(p, { method: 'POST', body: JSON.stringify(data ?? {}) })
export const put = (p, data) => api(p, { method: 'PUT', body: JSON.stringify(data ?? {}) })
export const del = (p) => api(p, { method: 'DELETE' })

export function fmtTime(s) {
  if (!s) return '-'
  return s.replace('T', ' ').replace(/([+-]\d{2}:\d{2}|Z)$/, '').slice(0, 19)
}

// 时长格式化委托给 i18n（随界面语言切换）
export { fmtDur } from './i18n'
