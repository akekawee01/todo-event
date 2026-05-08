<template>
  <div class="step-card">
    <div class="card-title">Verify your email</div>
    <div class="card-desc">Use the demo verification token generated for this registration.</div>

    <div v-if="props.user.verification_token" class="token-panel">
      <div class="token-label">Demo token</div>
      <div class="token-row">
        <code>{{ props.user.verification_token }}</code>
        <button class="btn-token" @click="copyToken">{{ copied ? 'Copied' : 'Copy' }}</button>
        <button class="btn-token" @click="token = props.user.verification_token">Use</button>
      </div>
    </div>

    <div class="form-field">
      <label for="token">Verification token</label>
      <input id="token" v-model="token" type="text" placeholder="Paste token here" @keydown.enter="submit" />
    </div>

    <button class="btn-primary" :disabled="!token.trim() || loading" @click="submit">
      {{ loading ? 'Verifying…' : 'Verify email' }}
    </button>
    <div v-if="error" class="form-error">{{ error }}</div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { verifyEmail } from '../api.ts'
import type { User } from '../types.ts'

const props = defineProps<{ user: User }>()
const emit = defineEmits<{ done: [user: User] }>()

const token = ref('')
const loading = ref(false)
const error = ref('')
const copied = ref(false)

async function copyToken(): Promise<void> {
  if (!props.user.verification_token) return
  await navigator.clipboard.writeText(props.user.verification_token)
  copied.value = true
  window.setTimeout(() => {
    copied.value = false
  }, 1200)
}

async function submit(): Promise<void> {
  if (!token.value.trim()) return
  loading.value = true
  error.value = ''
  try {
    const updated = await verifyEmail(props.user.id, token.value.trim())
    emit('done', updated)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}
</script>
