export function getPerms() {
  try {
    return JSON.parse(localStorage.getItem('permissions') || '[]')
  } catch {
    return []
  }
}

export function hasPerm(p) {
  return getPerms().includes(p)
}

export function saveSession(data) {
  localStorage.setItem('accessToken', data.accessToken)
  localStorage.setItem('refreshToken', data.refreshToken || '')
  localStorage.setItem('permissions', JSON.stringify(data.permissions || []))
  localStorage.setItem('user', JSON.stringify(data.user || {}))
  if (data.mustChangePassword) localStorage.setItem('mustChangePassword', '1')
  else localStorage.removeItem('mustChangePassword')
}

export function clearSession() {
  localStorage.clear()
}

export function mustChangePassword() {
  return localStorage.getItem('mustChangePassword') === '1'
}
