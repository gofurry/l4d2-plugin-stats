export async function copyTextToClipboard(value: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return
    } catch {
      // HTTP admin origins commonly reject the modern Clipboard API. Fall
      // through to the user-gesture-based legacy copy path.
    }
  }

  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.readOnly = true
  textarea.style.position = 'fixed'
  textarea.style.inset = '0 auto auto -9999px'
  textarea.style.opacity = '0'
  document.body.append(textarea)
  textarea.focus()
  textarea.select()
  try {
    if (!document.execCommand?.('copy')) throw new Error('clipboard copy was rejected')
  } finally {
    textarea.remove()
  }
}
