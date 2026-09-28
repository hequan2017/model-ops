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

export function fmtDur(sec) {
  if (sec == null) return '-'
  if (sec < 60) return `${sec}秒`
  if (sec < 3600) return `${Math.floor(sec / 60)}分${sec % 60}秒`
  return `${Math.floor(sec / 3600)}时${Math.floor((sec % 3600) / 60)}分`
}
