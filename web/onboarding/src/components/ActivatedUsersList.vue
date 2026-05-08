<template>
  <div class="step-card">
    <div class="card-title">Activated users</div>
    <div class="card-desc">Everyone who has completed onboarding.</div>

    <div v-if="loading" class="credit-waiting">
      <div class="spinner"></div>
      <div style="font-size: 0.85rem; color: var(--text-muted);">Loading…</div>
    </div>

    <div v-else-if="error" class="form-error">{{ error }}</div>

    <div v-else-if="users.length === 0" class="empty">
      No activated users yet. Walk one through the onboarding flow on the other tab.
    </div>

    <ul v-else class="user-list">
      <li v-for="u in users" :key="u.id" class="user-row">
        <template v-if="editingId !== u.id">
          <div class="user-name">{{ u.name }}</div>
          <div class="user-email">{{ u.email }}</div>
          <div class="user-meta">
            <span>credit {{ u.credit_score }}</span>
            <span>·</span>
            <span>{{ formatDate(u.created_at) }}</span>
          </div>
          <div v-if="u.bio" class="user-bio">{{ u.bio }}</div>
          <div class="user-actions-row">
            <button class="btn-link" @click="startEdit(u)">Edit</button>
            <button class="btn-link" @click="resetPwd(u)">Reset Password</button>
            <button class="btn-link" @click="toggleHistory(u)">
              {{ historyFor === u.id ? 'Hide history' : 'History' }}
            </button>
          </div>

          <div v-if="historyFor === u.id" class="history-panel">
            <div v-if="historyLoading" class="empty">Loading history…</div>
            <div v-else-if="historyError" class="form-error">{{ historyError }}</div>
            <div v-else-if="historyEvents.length === 0" class="empty">No events recorded.</div>
            <ol v-else class="history-list">
              <li v-for="ev in historyEvents" :key="ev.id" class="history-item">
                <div class="history-head">
                  <span class="history-type">{{ shortType(ev.type) }}</span>
                  <span class="history-time">{{ formatDate(ev.created_at) }}</span>
                </div>
                <pre class="history-payload">{{ JSON.stringify(ev.payload, null, 2) }}</pre>
              </li>
            </ol>
          </div>
        </template>

        <template v-else>
          <div class="user-edit">
            <input v-model="draft!.name" placeholder="Name" />
            <input v-model="draft!.email" placeholder="Email" type="email" />
            <textarea v-model="draft!.bio" placeholder="Short profile / bio" rows="3"></textarea>
            <div v-if="editError" class="form-error">{{ editError }}</div>
            <div class="user-actions">
              <button class="btn-primary" :disabled="saving" @click="save(u)">
                {{ saving ? 'Saving…' : 'Save' }}
              </button>
              <button class="btn-secondary" :disabled="saving" @click="cancelEdit">Cancel</button>
            </div>
          </div>
        </template>
      </li>
    </ul>

    <button class="btn-secondary" @click="load" :disabled="loading">Refresh</button>
  </div>

  <div v-if="resetPasswordFor && newPassword" class="password-modal-overlay" @click="resetPasswordFor = null; newPassword = null">
    <div class="password-modal" @click.stop>
      <h3>Password Reset</h3>
      <p class="password-label">New password for {{ users.find(u => u.id === resetPasswordFor)?.email }}:</p>
      <div class="password-display">{{ newPassword }}</div>
      <button class="btn-primary" @click="resetPasswordFor = null; newPassword = null">Close</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listActivated, updateContact, getUserHistory, resetPassword } from '../api.ts'
import type { User, UserEvent } from '../types.ts'

const users = ref<User[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const editingId = ref<string | null>(null)
const draft = ref<{ name: string; email: string; bio: string } | null>(null)
const editError = ref<string | null>(null)
const saving = ref(false)

const historyFor = ref<string | null>(null)
const historyEvents = ref<UserEvent[]>([])
const historyLoading = ref(false)
const historyError = ref<string | null>(null)

const resetPasswordFor = ref<string | null>(null)
const newPassword = ref<string | null>(null)
const resettingPassword = ref(false)

async function load() {
  loading.value = true
  error.value = null
  try {
    users.value = await listActivated()
  } catch (e: any) {
    error.value = e?.message ?? 'Failed to load'
  } finally {
    loading.value = false
  }
}

function startEdit(u: User) {
  editingId.value = u.id
  draft.value = { name: u.name, email: u.email, bio: u.bio }
  editError.value = null
}

function cancelEdit() {
  editingId.value = null
  draft.value = null
  editError.value = null
}

async function save(u: User) {
  if (!draft.value) return
  saving.value = true
  editError.value = null
  try {
    const updated = await updateContact(u.id, draft.value.name, draft.value.email, draft.value.bio)
    const i = users.value.findIndex(x => x.id === u.id)
    if (i >= 0) users.value[i] = updated
    cancelEdit()
  } catch (e: any) {
    editError.value = e?.message ?? 'Failed to save'
  } finally {
    saving.value = false
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString()
}

function shortType(t: string): string {
  return t.replace(/^user\./, '').replace(/_/g, ' ')
}

async function toggleHistory(u: User) {
  if (historyFor.value === u.id) {
    historyFor.value = null
    return
  }
  historyFor.value = u.id
  historyLoading.value = true
  historyEvents.value = []
  historyError.value = null
  try {
    historyEvents.value = await getUserHistory(u.id)
  } catch (e: any) {
    historyError.value = e?.message ?? 'Failed to load history'
  } finally {
    historyLoading.value = false
  }
}

async function resetPwd(u: User) {
  if (!confirm(`Reset password for ${u.email}?`)) return
  resettingPassword.value = true
  resetPasswordFor.value = u.id
  newPassword.value = null
  try {
    const result = await resetPassword(u.id)
    newPassword.value = result.new_password
  } catch (e: any) {
    alert(e?.message ?? 'Failed to reset password')
    resetPasswordFor.value = null
  } finally {
    resettingPassword.value = false
  }
}

onMounted(load)
</script>
