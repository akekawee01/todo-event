export type UserStatus =
  | 'registered'
  | 'email_verified'
  | 'credit_approved'
  | 'credit_denied'
  | 'onboarding_complete'

export interface User {
  id: string
  name: string
  email: string
  bio: string
  status: UserStatus
  verification_token: string
  credit_score: number
  credit_approved: boolean
  created_at: string
}

export interface Captcha {
  id: string
  question: string
  expires_at: string
}

export interface UserEvent {
  id: string
  aggregate_id: string
  type: string
  payload: any
  created_at: string
}
