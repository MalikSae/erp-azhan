// Utility to convert hex to HSL and generate brand color shade variables dynamically

function hexToHSL(hex) {
  let r = 0, g = 0, b = 0;
  const cleanHex = hex.replace('#', '');
  if (cleanHex.length === 3) {
    r = parseInt(cleanHex[0] + cleanHex[0], 16);
    g = parseInt(cleanHex[1] + cleanHex[1], 16);
    b = parseInt(cleanHex[2] + cleanHex[2], 16);
  } else if (cleanHex.length === 6) {
    r = parseInt(cleanHex.substring(0, 2), 16);
    g = parseInt(cleanHex.substring(2, 4), 16);
    b = parseInt(cleanHex.substring(4, 6), 16);
  }
  r /= 255;
  g /= 255;
  b /= 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  let h = 0;
  let s = 0;
  let l = (max + min) / 2;

  if (max !== min) {
    const d = max - min;
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
    switch (max) {
      case r:
        h = (g - b) / d + (g < b ? 6 : 0);
        break;
      case g:
        h = (b - r) / d + 2;
        break;
      case b:
        h = (r - g) / d + 4;
        break;
    }
    h /= 6;
  }
  return [Math.round(h * 360), Math.round(s * 100), Math.round(l * 100)];
}

function hslToHex(h, s, l) {
  s /= 100;
  l /= 100;
  const k = (n) => (n + h / 30) % 12;
  const a = s * Math.min(l, 1 - l);
  const f = (n) => l - a * Math.max(-1, Math.min(k(n) - 3, Math.min(9 - k(n), 1)));
  const toHex = (x) => Math.round(x * 255).toString(16).padStart(2, '0');
  return `#${toHex(f(0))}${toHex(f(8))}${toHex(f(4))}`;
}

export function isLightColor(hexColor) {
  if (!hexColor || typeof hexColor !== 'string' || !hexColor.startsWith('#')) return true;
  const hex = hexColor.replace('#', '');
  if (hex.length !== 6 && hex.length !== 3) return true;
  const r = parseInt(hex.length === 3 ? hex[0] + hex[0] : hex.slice(0, 2), 16);
  const g = parseInt(hex.length === 3 ? hex[1] + hex[1] : hex.slice(2, 4), 16);
  const b = parseInt(hex.length === 3 ? hex[2] + hex[2] : hex.slice(4, 6), 16);
  const brightness = (r * 299 + g * 587 + b * 114) / 1000;
  return brightness > 140;
}

export function generateBrandPalette(baseHex) {
  if (!baseHex || !baseHex.startsWith('#')) baseHex = '#FED853';
  const [h, s, l] = hexToHSL(baseHex);

  return {
    50: hslToHex(h, Math.min(s, 60), 96),
    100: hslToHex(h, Math.min(s, 65), 90),
    200: hslToHex(h, Math.min(s, 70), 82),
    300: hslToHex(h, Math.min(s, 75), 72),
    400: hslToHex(h, Math.min(s, 80), 62),
    500: baseHex,
    600: hslToHex(h, Math.min(s, 85), Math.max(10, l - 7)),
    700: hslToHex(h, Math.min(s, 90), Math.max(8, l - 15)),
    800: hslToHex(h, Math.min(s, 95), Math.max(6, l - 22)),
    900: hslToHex(h, Math.min(s, 95), Math.max(5, l - 30)),
    950: hslToHex(h, Math.min(s, 95), Math.max(4, l - 38)),
  };
}

export function applyBrandTheme(baseHex) {
  if (typeof document === 'undefined') return;
  const color = baseHex && baseHex.startsWith('#') ? baseHex : '#FED853';
  const palette = generateBrandPalette(color);
  const isLight = isLightColor(color);
  const root = document.documentElement;

  root.style.setProperty('--brand-primary', color);
  root.style.setProperty('--brand-primary-text', isLight ? '#14171A' : '#FFFFFF');

  Object.entries(palette).forEach(([shade, hex]) => {
    root.style.setProperty(`--brand-primary-${shade}`, hex);
  });
}
