<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const API_BASE = import.meta.env.VITE_API_BASE ?? ''

type ContentType = 'video' | 'audio' | 'subtitle' | 'comments'

interface VideoInfo {
  id: string
  title: string
  url: string
  duration: string
}

interface ResolveResponse {
  success: boolean
  kind: 'video' | 'playlist'
  title?: string
  video_id?: string
  watch_url?: string
  playlist_id?: string
  count?: number
  videos?: VideoInfo[]
  urls?: string[]
}

interface CommentResponse {
  success: boolean
  video_id: string
  video_title: string
  comment_count: number
  comments: unknown[]
}

const inputUrl = ref('')
const contentType = ref<ContentType>('video')
const subLang = ref('tr')
const loading = ref(false)
const errorMsg = ref('')
const resolved = ref<ResolveResponse | null>(null)
const selectedIds = ref<Set<string>>(new Set())
const progressText = ref('')

watch(resolved, (r) => {
  selectedIds.value = new Set()
  if (r?.kind === 'playlist' && r.videos) {
    for (const v of r.videos) {
      selectedIds.value.add(v.id)
    }
  }
})

const playlistVideos = computed(() => resolved.value?.videos ?? [])

function toggleId(id: string) {
  const s = new Set(selectedIds.value)
  if (s.has(id)) {
    s.delete(id)
  } else {
    s.add(id)
  }
  selectedIds.value = s
}

function selectAll() {
  const s = new Set<string>()
  for (const v of playlistVideos.value) {
    s.add(v.id)
  }
  selectedIds.value = s
}

function selectNone() {
  selectedIds.value = new Set()
}

async function postJSON<T>(path: string, body: object): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify(body),
  })
  const ct = res.headers.get('content-type') ?? ''
  if (!res.ok) {
    if (ct.includes('application/json')) {
      const err = (await res.json()) as { message?: string }
      throw new Error(err.message || res.statusText)
    }
    throw new Error(res.statusText)
  }
  return (await res.json()) as T
}

async function resolveLink() {
  errorMsg.value = ''
  resolved.value = null
  const url = inputUrl.value.trim()
  if (!url) {
    errorMsg.value = 'Bir YouTube bağlantısı yapıştırın.'
    return
  }
  loading.value = true
  try {
    const data = await postJSON<ResolveResponse>('/url/resolve', { url })
    if (!data.success) {
      throw new Error('Çözümleme başarısız')
    }
    resolved.value = data
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

function selectedUrls(): string[] {
  const r = resolved.value
  if (!r) return []
  if (r.kind === 'video') {
    return [r.watch_url || '']
  }
  const urls: string[] = []
  for (const v of playlistVideos.value) {
    if (selectedIds.value.has(v.id)) {
      urls.push(v.url)
    }
  }
  return urls
}

async function downloadBlob(path: string, body: object, fallbackName: string) {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const ct = res.headers.get('content-type') ?? ''
    if (ct.includes('application/json')) {
      const err = (await res.json()) as { message?: string }
      throw new Error(err.message || res.statusText)
    }
    throw new Error(res.statusText)
  }
  const cd = res.headers.get('Content-Disposition')
  let name = fallbackName
  const m = cd?.match(/filename="?([^";\n]+)"?/i)
  if (m?.[1]) {
    name = m[1]
  }
  const blob = await res.blob()
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = name
  a.click()
  URL.revokeObjectURL(a.href)
}

async function runDownloadZip() {
  const urls = selectedUrls()
  if (urls.length === 0) {
    errorMsg.value = 'En az bir video seçin.'
    return
  }
  errorMsg.value = ''
  loading.value = true
  progressText.value = ''
  try {
    if (contentType.value === 'comments') {
      errorMsg.value = 'Yorumlar için ZIP yerine JSON kullanın veya tek tek seçin.'
      return
    }
    if (contentType.value === 'video') {
      await downloadBlob('/download/video', { urls }, 'video.zip')
    } else if (contentType.value === 'audio') {
      await downloadBlob('/download/audio', { urls }, 'audio.zip')
    } else {
      await downloadBlob('/download/subtitle', { urls, lang: subLang.value }, 'subs.zip')
    }
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

async function runDownloadOneByOne() {
  const urls = selectedUrls()
  if (urls.length === 0) {
    errorMsg.value = 'En az bir video seçin.'
    return
  }
  errorMsg.value = ''
  loading.value = true
  let i = 0
  try {
    if (contentType.value === 'comments') {
      for (const u of urls) {
        i++
        progressText.value = `Yorumlar ${i} / ${urls.length}…`
        const data = await postJSON<CommentResponse>('/video/comments', { url: u, limit: 200 })
        downloadJsonFile(`yorumlar-${data.video_id}.json`, data)
      }
      progressText.value = ''
      return
    }
    const path =
      contentType.value === 'video'
        ? '/download/video'
        : contentType.value === 'audio'
          ? '/download/audio'
          : '/download/subtitle'

    for (const u of urls) {
      i++
      progressText.value = `İndiriliyor ${i} / ${urls.length}…`
      const body =
        contentType.value === 'subtitle'
          ? { urls: [u], lang: subLang.value }
          : { urls: [u] }
      const ext =
        contentType.value === 'video' ? 'mp4' : contentType.value === 'audio' ? 'mp3' : 'srt'
      await downloadBlob(path, body, `item-${i}.${ext}`)
    }
    progressText.value = ''
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

function downloadJsonFile(filename: string, data: unknown) {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = filename
  a.click()
  URL.revokeObjectURL(a.href)
}

async function runSingleVideoFlow() {
  const r = resolved.value
  if (!r || r.kind !== 'video') return
  const url = r.watch_url
  if (!url) {
    errorMsg.value = 'Geçerli bir izleme URL’si yok.'
    return
  }
  errorMsg.value = ''
  loading.value = true
  progressText.value = ''
  try {
    if (contentType.value === 'comments') {
      const data = await postJSON<CommentResponse>('/video/comments', { url, limit: 200 })
      downloadJsonFile(`yorumlar-${data.video_id}.json`, data)
      return
    }
    if (contentType.value === 'video') {
      await downloadBlob('/download/video', { urls: [url] }, 'video.mp4')
    } else if (contentType.value === 'audio') {
      await downloadBlob('/download/audio', { urls: [url] }, 'audio.mp3')
    } else {
      await downloadBlob('/download/subtitle', { urls: [url], lang: subLang.value }, 'subtitle.srt')
    }
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

const sourceUrlForResolve = computed(() => inputUrl.value.trim())
</script>

<template>
  <div>
    <h1>YouTube indirici</h1>
    <p class="lead">Bağlantıyı yapıştırın, ne indirmek istediğinizi seçin, önce bağlantıyı çözün.</p>

    <div class="toolbar">
      <input
        v-model="inputUrl"
        type="url"
        autocomplete="off"
        placeholder="https://www.youtube.com/watch?v=… veya oynatma listesi"
        @keydown.enter.prevent="resolveLink"
      />
      <select v-model="contentType">
        <option value="video">Video (MP4)</option>
        <option value="audio">Ses (MP3)</option>
        <option value="subtitle">Altyazı (SRT)</option>
        <option value="comments">Yorumlar (JSON)</option>
      </select>
      <button type="button" class="btn" :disabled="loading" @click="resolveLink">
        {{ loading && !resolved ? 'Çözülüyor…' : 'Bağlantıyı çöz' }}
      </button>
    </div>

    <div v-if="contentType === 'subtitle'" class="sub-lang">
      <label for="lang">Altyazı dili (kod)</label>
      <input id="lang" v-model="subLang" type="text" maxlength="8" placeholder="tr" />
    </div>

    <p v-if="errorMsg" class="err">{{ errorMsg }}</p>
    <p v-if="progressText" class="progress">{{ progressText }}</p>

    <div v-if="resolved" class="panel">
      <h2>
        <span :class="resolved.kind === 'playlist' ? 'badge badge-pl' : 'badge badge-video'">
          {{ resolved.kind === 'playlist' ? 'Oynatma listesi' : 'Video' }}
        </span>
        {{ resolved.title || '(başlıksız)' }}
      </h2>
      <p v-if="resolved.kind === 'video' && resolved.video_id" class="meta mono">ID: {{ resolved.video_id }}</p>
      <p v-if="resolved.kind === 'playlist'" class="meta">
        {{ resolved.count }} video · Kaynak:
        <span class="mono">{{ sourceUrlForResolve }}</span>
      </p>

      <template v-if="resolved.kind === 'video'">
        <div class="btn-row">
          <button type="button" class="btn" :disabled="loading" @click="runSingleVideoFlow">
            İndir
          </button>
        </div>
      </template>

      <template v-else>
        <p class="meta">Toplu (tek istekte ZIP) veya tek tek dosya indirme.</p>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th style="width: 2.5rem"></th>
                <th>Video</th>
                <th style="width: 5rem">Süre</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="v in playlistVideos" :key="v.id">
                <td>
                  <input type="checkbox" :checked="selectedIds.has(v.id)" @change="toggleId(v.id)" />
                </td>
                <td>{{ v.title }}</td>
                <td>{{ v.duration || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="small-actions">
          <button type="button" class="btn-secondary" @click="selectAll">Tümünü seç</button>
          <button type="button" class="btn-secondary" @click="selectNone">Hiçbirini seçme</button>
        </div>
        <div class="btn-row">
          <button
            type="button"
            class="btn"
            :disabled="loading || contentType === 'comments'"
            @click="runDownloadZip"
          >
            Seçilenleri ZIP ile indir
          </button>
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="runDownloadOneByOne">
            {{ contentType === 'comments' ? 'Seçilenleri tek tek JSON' : 'Seçilenleri tek tek indir' }}
          </button>
        </div>
      </template>
    </div>

    <p class="meta" style="margin-top: 2rem; text-align: center">
      API:
      <a :href="`${API_BASE}/swagger/index.html`" target="_blank" rel="noopener">Swagger</a>
    </p>
  </div>
</template>
