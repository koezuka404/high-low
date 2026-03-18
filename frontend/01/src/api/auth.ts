import { apiPost } from './http'

export async function signup(params: { email: string; password: string }): Promise<void> {
  await apiPost<null>('/signup', params)
}

export async function login(params: { email: string; password: string }): Promise<void> {
  await apiPost<null>('/login', params)
}

export async function logout(): Promise<void> {
  await apiPost<null>('/logout', {})
}
