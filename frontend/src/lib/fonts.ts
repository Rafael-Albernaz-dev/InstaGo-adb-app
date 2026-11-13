export interface FontLoadResult {
  loaded: string[];
  failed: string[];
}

const LOCAL_FONT_DEFS = [
  { family: 'Sora', url: '/creative-assets/fonts/sora.woff2', weight: '100 900' },
  { family: 'Inter', url: '/creative-assets/fonts/inter.woff2', weight: '100 900' }
];

export async function loadBrandFonts(): Promise<FontLoadResult> {
  if (typeof FontFace === 'undefined' || typeof document === 'undefined' || !document.fonts) {
    return { loaded: [], failed: LOCAL_FONT_DEFS.map((f) => f.family) };
  }

  const loaded: string[] = [];
  const failed: string[] = [];

  for (const def of LOCAL_FONT_DEFS) {
    try {
      const face = new FontFace(def.family, `url("${def.url}") format("woff2")`, {
        weight: def.weight,
        style: 'normal'
      });
      const ready = await face.load();
      document.fonts.add(ready);
      loaded.push(def.family);
    } catch {
      failed.push(def.family);
    }
  }

  try {
    await document.fonts.ready;
  } catch {
    // ignore in synthetic DOM environments
  }

  return { loaded, failed };
}
