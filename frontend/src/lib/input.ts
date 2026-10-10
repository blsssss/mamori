// Chromium draws the focus ring on a focus() that follows a click on a button which had the ring, so
// a screen opened by the mouse must not move the focus itself; a keyboard user needs it moved
let keyboard = false

export function trackInput(): () => void {
  keyboard = false
  const key = () => {
    keyboard = true
  }
  const pointer = () => {
    keyboard = false
  }
  window.addEventListener('keydown', key, true)
  window.addEventListener('pointerdown', pointer, true)
  return () => {
    window.removeEventListener('keydown', key, true)
    window.removeEventListener('pointerdown', pointer, true)
  }
}

export function byKeyboard(): boolean {
  return keyboard
}
