<script setup lang="ts">
import { computed, ref } from 'vue'

const API_BASE = import.meta.env.VITE_API_BASE ?? ''

type ContentType = 'video' | 'audio' | 'subtitle' | 'comments'
type VideoQuality = 'best' | '1080' | '720' | '480' | '360'

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

interface QueueItem {
  key: string
  id?: string
  url: string
  title: string
  duration?: string
  thumbnail_url?: string
  source: 'video' | 'playlist'
}

const inputUrl = ref('')
const contentType = ref<ContentType>('video')
const videoQuality = ref<VideoQuality>('best')
const subLang = ref('tr')
/** Yorumlar: API'ye compact=true — yalnızca author, text, timestamp (ISO 8601 / RFC3339) */
const commentsCompact = ref(true)
/** yt-dlp max_comments — arayüz daha önce 200 sabitti; istenen adedi buradan gönderin */
const commentsLimit = ref(500)
const loading = ref(false)
const errorMsg = ref('')
const queueItems = ref<QueueItem[]>([])
const selectedIds = ref<Set<string>>(new Set())
const progressText = ref('')

const selectedCount = computed(() => {
  let n = 0
  for (const it of queueItems.value) {
    if (selectedIds.value.has(it.key)) n++
  }
  return n
})

function toggleId(key: string) {
  const s = new Set(selectedIds.value)
  if (s.has(key)) {
    s.delete(key)
  } else {
    s.add(key)
  }
  selectedIds.value = s
}

function selectAll() {
  const s = new Set<string>()
  for (const it of queueItems.value) {
    s.add(it.key)
  }
  selectedIds.value = s
}

function selectNone() {
  selectedIds.value = new Set()
}

function removeItem(key: string) {
  queueItems.value = queueItems.value.filter((x) => x.key !== key)
  if (selectedIds.value.has(key)) {
    const s = new Set(selectedIds.value)
    s.delete(key)
    selectedIds.value = s
  }
}

function removeSelected() {
  const selected = selectedIds.value
  if (selected.size === 0) return
  queueItems.value = queueItems.value.filter((x) => !selected.has(x.key))
  selectedIds.value = new Set()
}

function extractUrls(text: string): string[] {
  const matches = text.match(/https?:\/\/[^\s<>"']+/gi) ?? []
  const urls = matches.map((s) => s.trim()).filter(Boolean)
  const seen = new Set<string>()
  const out: string[] = []
  for (const u of urls) {
    if (seen.has(u)) continue
    seen.add(u)
    out.push(u)
  }
  return out
}

function guessVideoIdFromUrl(url: string): string | undefined {
  try {
    const u = new URL(url)
    const v = u.searchParams.get('v')
    if (v) return v
    if (u.hostname === 'youtu.be') {
      const p = u.pathname.replace(/^\/+/, '').trim()
      if (p) return p.split(/[/?#&]/)[0]
    }
  } catch {
    // ignore
  }
  return undefined
}

function thumbnailUrlForId(id: string): string {
  return `https://i.ytimg.com/vi/${id}/hqdefault.jpg`
}

function clampCommentLimit(n: number): number {
  if (!Number.isFinite(n) || n < 1) return 100
  return Math.min(10000, Math.floor(n))
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

function addQueueItem(item: Omit<QueueItem, 'key'> & { key?: string }) {
  const key = item.key ?? item.id ?? item.url
  if (!key) return
  if (queueItems.value.some((x) => x.key === key)) return
  queueItems.value = [...queueItems.value, { ...item, key }]
  selectedIds.value = new Set([...selectedIds.value, key])
}

async function enqueueFromText(text: string) {
  errorMsg.value = ''
  const urls = extractUrls(text)
  if (urls.length === 0) return

  loading.value = true
  progressText.value = ''
  try {
    let i = 0
    for (const url of urls) {
      i++
      progressText.value = `Bağlantılar ekleniyor ${i} / ${urls.length}…`
      const data = await postJSON<ResolveResponse>('/url/resolve', { url })
      if (!data.success) continue

      if (data.kind === 'playlist' && data.videos?.length) {
        for (const v of data.videos) {
          const id = v.id || guessVideoIdFromUrl(v.url)
          addQueueItem({
            id,
            url: v.url,
            title: v.title || '(başlıksız)',
            duration: v.duration,
            thumbnail_url: id ? thumbnailUrlForId(id) : undefined,
            source: 'playlist',
          })
        }
        continue
      }

      const resolvedUrl = data.watch_url || url
      const id =
        data.video_id || guessVideoIdFromUrl(resolvedUrl) || guessVideoIdFromUrl(url) || guessVideoIdFromUrl(text)
      addQueueItem({
        id,
        url: resolvedUrl,
        title: data.title || '(başlıksız)',
        thumbnail_url: id ? thumbnailUrlForId(id) : undefined,
        source: 'video',
      })
    }
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
    progressText.value = ''
  }
}

async function addFromInput() {
  const text = inputUrl.value.trim()
  if (!text) {
    errorMsg.value = 'Bir veya birden fazla YouTube bağlantısı yapıştırın.'
    return
  }
  await enqueueFromText(text)
  inputUrl.value = ''
}

async function onPaste(e: ClipboardEvent) {
  const text = e.clipboardData?.getData('text') ?? ''
  const urls = extractUrls(text)
  if (urls.length === 0) return
  e.preventDefault()
  inputUrl.value = urls.join('\n')
}

function selectedUrls(): string[] {
  const urls: string[] = []
  for (const it of queueItems.value) {
    if (!selectedIds.value.has(it.key)) continue
    if (it.url) urls.push(it.url)
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
      await downloadBlob('/download/video', { urls, quality: videoQuality.value }, 'video.zip')
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
        const data = await postJSON<CommentResponse>('/video/comments', {
          url: u,
          limit: clampCommentLimit(commentsLimit.value),
          compact: commentsCompact.value,
        })
        const suffix = commentsCompact.value ? '-ozet' : ''
        downloadJsonFile(`yorumlar-${data.video_id}${suffix}.json`, data)
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
          : contentType.value === 'video'
            ? { urls: [u], quality: videoQuality.value }
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
</script>

<template>
  <div>
    <h1>YouTube indirici</h1>
    <p class="lead">
      Bağlantıları yapıştırın; listeye eklensin. Aksiyon seçin ve seçilenleri tek tek veya ZIP olarak indirin.
    </p>

    <div class="toolbar">
      <input
        v-model="inputUrl"
        type="url"
        autocomplete="off"
        placeholder="https://www.youtube.com/watch?v=… veya oynatma listesi"
        @paste="onPaste"
        @keydown.enter.prevent="addFromInput"
      />
      <button type="button" class="btn" :disabled="loading" @click="addFromInput">Listeye ekle</button>
    </div>

    <div class="actions-bar">
      <select v-model="contentType">
        <option value="video">Video (MP4)</option>
        <option value="audio">Ses (MP3)</option>
        <option value="subtitle">Altyazı (SRT)</option>
        <option value="comments">Yorumlar (JSON)</option>
      </select>

      <select v-if="contentType === 'video'" v-model="videoQuality">
        <option value="best">Kalite: En iyi</option>
        <option value="1080">Kalite: 1080p</option>
        <option value="720">Kalite: 720p</option>
        <option value="480">Kalite: 480p</option>
        <option value="360">Kalite: 360p</option>
      </select>

      <div v-if="contentType === 'subtitle'" class="sub-lang">
        <label for="lang">Altyazı dili (kod)</label>
        <input id="lang" v-model="subLang" type="text" maxlength="8" placeholder="tr" />
      </div>

      <div v-if="contentType === 'comments'" class="sub-lang comments-opt">
        <label class="check">
          <input v-model="commentsCompact" type="checkbox" />
          Sadeleştirilmiş yorum JSON’u (yazar, metin, tarih-saat ISO)
        </label>
        <label class="comments-limit-label" for="comments-limit">
          En fazla yorum
          <input
            id="comments-limit"
            v-model.number="commentsLimit"
            type="number"
            min="1"
            max="10000"
            step="1"
          />
        </label>
      </div>
    </div>

    <p v-if="errorMsg" class="err">{{ errorMsg }}</p>
    <p v-if="progressText" class="progress">{{ progressText }}</p>

    <div v-if="queueItems.length" class="panel">
      <h2>Liste ({{ selectedCount }} / {{ queueItems.length }} seçili)</h2>
      <p class="meta">Toplu (tek istekte ZIP) veya tek tek dosya indirme.</p>

      <div class="list-actions">
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="loading || queueItems.length === 0"
          @click="selectAll"
        >
          Tümünü seç
        </button>
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="loading || queueItems.length === 0"
          @click="selectNone"
        >
          Hiçbirini seçme
        </button>
        <button type="button" class="btn btn-danger" :disabled="loading || selectedCount === 0" @click="removeSelected">
          Seçilenleri sil
        </button>
      </div>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th style="width: 2.5rem"></th>
              <th style="width: 4.5rem"></th>
              <th>Video</th>
              <th style="width: 5rem">Süre</th>
              <th style="width: 6rem"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in queueItems" :key="it.key">
              <td>
                <input type="checkbox" :checked="selectedIds.has(it.key)" @change="toggleId(it.key)" />
              </td>
              <td>
                <img v-if="it.thumbnail_url" class="thumb" :src="it.thumbnail_url" alt="" loading="lazy" />
                <div v-else class="thumb thumb-empty" aria-hidden="true"></div>
              </td>
              <td>
                <div class="title-row">
                  <span class="item-title">{{ it.title }}</span>
                  <span class="item-url mono">{{ it.url }}</span>
                </div>
              </td>
              <td>{{ it.duration || '—' }}</td>
              <td>
                <button type="button" class="btn btn-danger btn-mini" :disabled="loading" @click="removeItem(it.key)">
                  Sil
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="btn-row">
        <button type="button" class="btn" :disabled="loading || contentType === 'comments'" @click="runDownloadZip">
          Seçilenleri ZIP ile indir
        </button>
        <button type="button" class="btn btn-secondary" :disabled="loading" @click="runDownloadOneByOne">
          {{ contentType === 'comments' ? 'Seçilenleri tek tek JSON' : 'Seçilenleri tek tek indir' }}
        </button>
      </div>
    </div>

    <p class="meta" style="margin-top: 2rem; text-align: center">
      API:
      <a :href="`${API_BASE}/swagger/index.html`" target="_blank" rel="noopener">Swagger</a>
    </p>
  </div>
</template>
