import type { CartItem, Product, SellerProfile } from '@/types/domain'

type ManagedMediaKind = 'product' | 'seller-logo' | 'seller-banner'

interface ManagedMediaRef {
  kind: ManagedMediaKind
  seed: string
  variant: string
}

interface MediaOptions {
  kind?: ManagedMediaKind
  seed?: string
  title?: string
  subtitle?: string
  description?: string
  badges?: string[]
  price?: number
  currency?: string
  stock?: number
  categoryName?: string
  sku?: string
  renderMode?: 'photo' | 'illustration'
}

interface Palette {
  backgroundStart: string
  backgroundEnd: string
  panel: string
  accent: string
  accentSoft: string
  glow: string
  text: string
  textMuted: string
}

const MANAGED_MEDIA_PROTOCOL = 'marketplace-media://'
const EXTERNAL_PLACEHOLDER_PATTERN = /^https?:\/\/(?:placehold\.co|loremflickr\.com)\//i
const PRODUCT_GALLERY_VARIANTS = ['hero', 'detail', 'lifestyle'] as const

const palettes: Palette[] = [
  {
    backgroundStart: '#12090b',
    backgroundEnd: '#d81f27',
    panel: '#0d0708',
    accent: '#ff9ca0',
    accentSoft: '#ffe3e5',
    glow: '#ff6169',
    text: '#fff7f7',
    textMuted: '#f7c9cc',
  },
  {
    backgroundStart: '#1a1012',
    backgroundEnd: '#9f151c',
    panel: '#0c0809',
    accent: '#ffc2c5',
    accentSoft: '#fff0f1',
    glow: '#f87171',
    text: '#fff7f7',
    textMuted: '#f4c4c7',
  },
  {
    backgroundStart: '#101012',
    backgroundEnd: '#7e1318',
    panel: '#09090a',
    accent: '#ffd1c2',
    accentSoft: '#fff1ea',
    glow: '#ff7d57',
    text: '#fff7f4',
    textMuted: '#ffd0c2',
  },
  {
    backgroundStart: '#180d12',
    backgroundEnd: '#5d0f18',
    panel: '#10070c',
    accent: '#ffb0be',
    accentSoft: '#ffe8ee',
    glow: '#f14668',
    text: '#fff5f8',
    textMuted: '#f7c7d2',
  },
  {
    backgroundStart: '#141414',
    backgroundEnd: '#8e1c23',
    panel: '#09090a',
    accent: '#ffb8b3',
    accentSoft: '#fff0ee',
    glow: '#ff7467',
    text: '#fff7f6',
    textMuted: '#f3cbc8',
  },
  {
    backgroundStart: '#1b0c0d',
    backgroundEnd: '#2a2a2a',
    panel: '#0c0909',
    accent: '#ff9ea1',
    accentSoft: '#ffe7e8',
    glow: '#ff5058',
    text: '#fff8f8',
    textMuted: '#edc3c5',
  },
]

const hashString = (value: string) => {
  let hash = 2166136261
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return hash >>> 0
}

const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max)

const getPalette = (seed: string) => palettes[hashString(seed) % palettes.length]

const escapeXml = (value: string) =>
  value
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

const trimText = (value: string, maxLength: number) => {
  if (value.length <= maxLength) {
    return value
  }
  return `${value.slice(0, Math.max(0, maxLength - 3)).trimEnd()}...`
}

const splitLines = (value: string, maxLength: number, maxLines: number) => {
  const clean = value.trim()
  if (!clean) {
    return []
  }

  const words = clean.split(/\s+/)
  const lines: string[] = []
  let current = ''

  for (const word of words) {
    const candidate = current ? `${current} ${word}` : word
    if (candidate.length <= maxLength) {
      current = candidate
      continue
    }

    if (current) {
      lines.push(current)
      current = word
    } else {
      lines.push(trimText(word, maxLength))
      current = ''
    }

    if (lines.length === maxLines - 1) {
      const remainder = [current, ...words.slice(words.indexOf(word) + 1)].filter(Boolean).join(' ')
      if (remainder) {
        lines.push(trimText(remainder, maxLength))
      }
      return lines.slice(0, maxLines)
    }
  }

  if (current) {
    lines.push(current)
  }

  return lines.slice(0, maxLines)
}

const encodeSvg = (svg: string) => `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`

const getInitials = (value: string) =>
  value
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('')

const normalizeBadges = (badges: Array<string | undefined>) =>
  Array.from(
    new Set(
      badges
        .map((badge) => badge?.trim())
        .filter((badge): badge is string => Boolean(badge)),
    ),
  ).slice(0, 3)

const parseManagedMediaRef = (value: string): ManagedMediaRef | null => {
  if (!value.startsWith(MANAGED_MEDIA_PROTOCOL)) {
    return null
  }

  const rawPath = value.slice(MANAGED_MEDIA_PROTOCOL.length)
  const parts = rawPath.split('/').filter(Boolean)

  if (parts[0] === 'product') {
    return {
      kind: 'product',
      seed: parts[1] ?? 'product',
      variant: parts[2] ?? 'hero',
    }
  }

  if (parts[0] === 'seller') {
    const variant = parts[2] ?? 'banner'
    return {
      kind: variant === 'logo' ? 'seller-logo' : 'seller-banner',
      seed: parts[1] ?? 'seller',
      variant,
    }
  }

  return null
}

const shouldUseManagedMedia = (value?: string) => {
  if (!value) {
    return true
  }
  return value.startsWith(MANAGED_MEDIA_PROTOCOL) || EXTERNAL_PLACEHOLDER_PATTERN.test(value)
}

export const isGeneratedMediaSource = (value?: string) => {
  if (!value) {
    return false
  }
  return value.startsWith(MANAGED_MEDIA_PROTOCOL) || EXTERNAL_PLACEHOLDER_PATTERN.test(value)
}

type ProductVisualKind =
  | 'phone'
  | 'laptop'
  | 'audio'
  | 'camera'
  | 'gaming'
  | 'router'
  | 'appliance'
  | 'furniture'
  | 'fashion'
  | 'shoe'
  | 'beauty'
  | 'bag'
  | 'book'
  | 'food'
  | 'tool'
  | 'pet'
  | 'sports'
  | 'jewelry'
  | 'baby'
  | 'auto'
  | 'generic'

const PRODUCT_PHOTO_TAGS: Record<ProductVisualKind, string[]> = {
  phone: ['smartphone', 'electronics', 'product', 'studio'],
  laptop: ['laptop', 'computer', 'desk', 'technology'],
  audio: ['headphones', 'audio', 'speaker', 'product'],
  camera: ['camera', 'photography', 'device', 'product'],
  gaming: ['gaming', 'controller', 'console', 'product'],
  router: ['router', 'electronics', 'network', 'device'],
  appliance: ['appliance', 'kitchen', 'home', 'product'],
  furniture: ['furniture', 'interior', 'home', 'design'],
  fashion: ['fashion', 'clothing', 'apparel', 'studio'],
  shoe: ['shoes', 'sneakers', 'footwear', 'fashion'],
  beauty: ['cosmetics', 'beauty', 'skincare', 'product'],
  bag: ['bag', 'accessories', 'fashion', 'product'],
  book: ['book', 'reading', 'stationery', 'product'],
  food: ['food', 'grocery', 'packaging', 'product'],
  tool: ['tools', 'hardware', 'workshop', 'product'],
  pet: ['pet', 'accessories', 'animal', 'product'],
  sports: ['fitness', 'sports', 'equipment', 'product'],
  jewelry: ['jewelry', 'accessories', 'fashion', 'luxury'],
  baby: ['baby', 'kids', 'nursery', 'product'],
  auto: ['car', 'automotive', 'accessories', 'product'],
  generic: ['product', 'shopping', 'retail', 'studio'],
}

const PRODUCT_VARIANT_TAGS: Record<(typeof PRODUCT_GALLERY_VARIANTS)[number], string[]> = {
  hero: ['studio', 'isolated'],
  detail: ['closeup', 'studio'],
  lifestyle: ['lifestyle', 'interior'],
}

const PRODUCT_QUERY_HINTS: Array<{ match: string; tags: string[] }> = [
  { match: 'смартфон', tags: ['smartphone'] },
  { match: 'телефон', tags: ['smartphone'] },
  { match: 'iphone', tags: ['smartphone'] },
  { match: 'android', tags: ['smartphone'] },
  { match: 'науш', tags: ['headphones'] },
  { match: 'гарнитур', tags: ['headphones'] },
  { match: 'колонк', tags: ['speaker'] },
  { match: 'ноутбук', tags: ['laptop'] },
  { match: 'монитор', tags: ['monitor'] },
  { match: 'камера', tags: ['camera'] },
  { match: 'фотоаппарат', tags: ['camera'] },
  { match: 'роутер', tags: ['router'] },
  { match: 'router', tags: ['router'] },
  { match: 'миксер', tags: ['mixer'] },
  { match: 'блендер', tags: ['blender'] },
  { match: 'кофемаш', tags: ['coffee-maker'] },
  { match: 'чайник', tags: ['kettle'] },
  { match: 'пылесос', tags: ['vacuum'] },
  { match: 'лампа', tags: ['lamp'] },
  { match: 'диван', tags: ['sofa'] },
  { match: 'кресл', tags: ['armchair'] },
  { match: 'стол', tags: ['table'] },
  { match: 'стул', tags: ['chair'] },
  { match: 'сумк', tags: ['handbag'] },
  { match: 'рюкзак', tags: ['backpack'] },
  { match: 'чемодан', tags: ['luggage'] },
  { match: 'кошелек', tags: ['wallet'] },
  { match: 'кроссов', tags: ['sneakers'] },
  { match: 'кеды', tags: ['sneakers'] },
  { match: 'ботин', tags: ['boots'] },
  { match: 'туфл', tags: ['heels'] },
  { match: 'шампун', tags: ['shampoo'] },
  { match: 'крем', tags: ['cream'] },
  { match: 'сыворот', tags: ['serum'] },
  { match: 'помада', tags: ['lipstick'] },
  { match: 'парф', tags: ['perfume'] },
  { match: 'книга', tags: ['book'] },
  { match: 'блокнот', tags: ['notebook'] },
  { match: 'тетрад', tags: ['notebook'] },
  { match: 'йога', tags: ['yoga'] },
  { match: 'гантел', tags: ['dumbbell'] },
  { match: 'мяч', tags: ['ball'] },
  { match: 'велосипед', tags: ['bicycle'] },
  { match: 'самокат', tags: ['scooter'] },
  { match: 'корм', tags: ['pet-food'] },
  { match: 'ошейник', tags: ['collar'] },
  { match: 'коляск', tags: ['stroller'] },
  { match: 'подгуз', tags: ['diaper'] },
  { match: 'очки', tags: ['sunglasses'] },
  { match: 'браслет', tags: ['bracelet'] },
  { match: 'кольц', tags: ['ring'] },
  { match: 'серьг', tags: ['earrings'] },
  { match: 'smartfon', tags: ['smartphone'] },
  { match: 'naush', tags: ['headphones'] },
  { match: 'noutbuk', tags: ['laptop'] },
  { match: 'kolonk', tags: ['speaker'] },
  { match: 'mikser', tags: ['mixer'] },
  { match: 'blender', tags: ['blender'] },
  { match: 'kofemash', tags: ['coffee-maker'] },
  { match: 'pylesos', tags: ['vacuum'] },
  { match: 'lamp', tags: ['lamp'] },
  { match: 'divan', tags: ['sofa'] },
  { match: 'kreslo', tags: ['armchair'] },
  { match: 'ryukzak', tags: ['backpack'] },
  { match: 'sumka', tags: ['handbag'] },
  { match: 'chemodan', tags: ['luggage'] },
  { match: 'krossov', tags: ['sneakers'] },
  { match: 'botin', tags: ['boots'] },
  { match: 'shampun', tags: ['shampoo'] },
  { match: 'kniga', tags: ['book'] },
  { match: 'bloknot', tags: ['notebook'] },
  { match: 'yoga', tags: ['yoga'] },
  { match: 'gantel', tags: ['dumbbell'] },
  { match: 'myach', tags: ['ball'] },
  { match: 'ochki', tags: ['sunglasses'] },
  { match: 'braslet', tags: ['bracelet'] },
]

interface ProductPhotoRule {
  patterns: string[]
  tags: string[]
}

const PRODUCT_PHRASE_RULES: ProductPhotoRule[] = [
  { patterns: ['chehol-dlya-smartfona', 'chehol-na-telefon'], tags: ['phone-case', 'smartphone'] },
  { patterns: ['chehol-knizhka'], tags: ['wallet-case', 'smartphone'] },
  { patterns: ['vodonepronitsaemyy-chehol'], tags: ['waterproof-phone-case', 'smartphone'] },
  { patterns: ['zaschitnoe-steklo', 'zaschitnaya-plenka'], tags: ['screen-protector', 'smartphone'] },
  { patterns: ['derzhatel-v-avto', 'magnitnyy-derzhatel', 'derzhatel-dlya-telefona-v-avto'], tags: ['car-phone-holder'] },
  { patterns: ['koltsevaya-lampa'], tags: ['ring-light'] },
  { patterns: ['shtativ-dlya-telefona'], tags: ['phone-tripod'] },
  { patterns: ['monopod-dlya-selfi'], tags: ['selfie-stick'] },
  { patterns: ['setevoe-zaryadnoe-ustroystvo'], tags: ['wall-charger'] },
  { patterns: ['avtomobilnoe-zaryadnoe-ustroystvo', 'zaryadka-v-prikurivatel'], tags: ['car-charger'] },
  { patterns: ['besprovodnaya-zaryadka'], tags: ['wireless-charger'] },
  { patterns: ['kabel-usb-c', 'zaryadnyy-kabel-lightning', 'zaryadnyy-kabel-micro-usb'], tags: ['charging-cable'] },
  { patterns: ['perekhodnik-3-5', 'usb-c-hdmi', 'adapter-hdmi', 'bluetooth-adapter'], tags: ['adapter'] },
  { patterns: ['pauerbank-s-bystroy-zaryadkoy', 'vneshniy-akkumulyator-s-solnechnoy-panelyu'], tags: ['power-bank'] },
  { patterns: ['vneshniy-zhestkiy-disk', 'vneshniy-hdd'], tags: ['external-hard-drive'] },
  { patterns: ['ssd-nakopitel', 'vneshniy-ssd'], tags: ['external-ssd'] },
  { patterns: ['usb-fleshka', 'fleshka-dlya-smartfona'], tags: ['usb-drive'] },
  { patterns: ['karta-pamyati'], tags: ['memory-card'] },
  { patterns: ['wi-fi-usilitel'], tags: ['wifi-repeater'] },
  { patterns: ['smart-lampa'], tags: ['smart-bulb'] },
  { patterns: ['smart-rozetka'], tags: ['smart-plug'] },
  { patterns: ['umnaya-kolonka'], tags: ['smart-speaker'] },
  { patterns: ['veb-kamera', 'webcam'], tags: ['webcam'] },
  { patterns: ['ekshn-kamera'], tags: ['action-camera'] },
  { patterns: ['fotoapparat'], tags: ['camera'] },
  { patterns: ['videokamera'], tags: ['camcorder'] },
  { patterns: ['ip-kamera'], tags: ['security-camera'] },
  { patterns: ['videoregistrator'], tags: ['dashcam'] },
  { patterns: ['gps-navigator'], tags: ['gps-navigator'] },
  { patterns: ['diktofon'], tags: ['voice-recorder'] },
  { patterns: ['lazernaya-ukazka'], tags: ['laser-pointer'] },
  { patterns: ['mini-proektor'], tags: ['projector'] },
  { patterns: ['elektronnaya-kniga'], tags: ['e-reader'] },
  { patterns: ['graficheskiy-planshet'], tags: ['drawing-tablet'] },
  { patterns: ['vr-garnitura'], tags: ['vr-headset'] },
  { patterns: ['igrovaya-konsol'], tags: ['game-console'] },
  { patterns: ['dzhoystik', 'geympad'], tags: ['game-controller'] },
  { patterns: ['besprovodnye-naushniki', 'tws-naushniki'], tags: ['wireless-earbuds'] },
  { patterns: ['provodnye-naushniki'], tags: ['wired-headphones'] },
  { patterns: ['portativnaya-kolonka'], tags: ['portable-speaker'] },
  { patterns: ['domashnyaya-akustika'], tags: ['home-speaker'] },
  { patterns: ['smart-chasy', 'chasy-naruchnye'], tags: ['watch'] },
  { patterns: ['fitnes-braslet'], tags: ['fitness-tracker'] },
  { patterns: ['robot-pylesos'], tags: ['robot-vacuum'] },
  { patterns: ['vertikalnyy-pylesos', 'pylesos-moyuschiy'], tags: ['vacuum-cleaner'] },
  { patterns: ['elektrochaynik', 'chaynik-dlya-plity'], tags: ['kettle'] },
  { patterns: ['kofevarka', 'kofemashina'], tags: ['coffee-maker'] },
  { patterns: ['pogruzhnoy-blender', 'blender'], tags: ['blender'] },
  { patterns: ['mikser'], tags: ['stand-mixer'] },
  { patterns: ['kukhonnyy-kombayn'], tags: ['food-processor'] },
  { patterns: ['myasorubka'], tags: ['meat-grinder'] },
  { patterns: ['multivarka', 'skorovarka', 'parovarka'], tags: ['multicooker'] },
  { patterns: ['aerogril', 'elektrogril'], tags: ['air-fryer'] },
  { patterns: ['kholodilnik', 'morozilnaya-kamera'], tags: ['refrigerator'] },
  { patterns: ['stiralnaya-mashina', 'sushilnaya-mashina'], tags: ['washing-machine'] },
  { patterns: ['posudomoechnaya-mashina'], tags: ['dishwasher'] },
  { patterns: ['mikrovolnovaya-pech'], tags: ['microwave'] },
  { patterns: ['duhovoy-shkaf'], tags: ['oven'] },
  { patterns: ['varochnaya-panel', 'elektricheskaya-plita', 'gazovaya-plita'], tags: ['cooktop'] },
  { patterns: ['vytyazhka'], tags: ['range-hood'] },
  { patterns: ['utyug', 'otparivatel'], tags: ['clothes-steamer'] },
  { patterns: ['shveynaya-mashina'], tags: ['sewing-machine'] },
  { patterns: ['obogrevatel', 'konvektor', 'teploventilyator'], tags: ['space-heater'] },
  { patterns: ['konditsioner'], tags: ['air-conditioner'] },
  { patterns: ['uvlazhnitel-vozdukha', 'ochistitel-vozdukha', 'osushitel-vozdukha'], tags: ['air-purifier'] },
  { patterns: ['ventilyator-napolnyy'], tags: ['tower-fan'] },
  { patterns: ['vodonagrevatel'], tags: ['water-heater'] },
  { patterns: ['elektrokamin'], tags: ['electric-fireplace'] },
  { patterns: ['sokovyzhimalka'], tags: ['juicer'] },
  { patterns: ['khlebopechka'], tags: ['bread-maker'] },
  { patterns: ['yogurtnitsa'], tags: ['yogurt-maker'] },
  { patterns: ['vafelnitsa'], tags: ['waffle-maker'] },
  { patterns: ['fen-schetka', 'fen'], tags: ['hair-dryer'] },
  { patterns: ['elektrobritva'], tags: ['electric-shaver'] },
  { patterns: ['epilyator'], tags: ['epilator'] },
  { patterns: ['skovoroda'], tags: ['frying-pan'] },
  { patterns: ['kastrulya', 'kovsh'], tags: ['cooking-pot'] },
  { patterns: ['nozh-kukhonnyy', 'nabor-nozhey'], tags: ['kitchen-knife'] },
  { patterns: ['doska-razdelochnaya'], tags: ['cutting-board'] },
  { patterns: ['terka'], tags: ['grater'] },
  { patterns: ['durshlag'], tags: ['colander'] },
  { patterns: ['konteyner-dlya-edy', 'lanch-boks'], tags: ['food-container'] },
  { patterns: ['termos-turisticheskiy', 'termos', 'termokruzhka'], tags: ['thermos'] },
  { patterns: ['butylka-dlya-vody', 'butylka-sportivnaya'], tags: ['water-bottle'] },
  { patterns: ['tarelka', 'salatnik'], tags: ['plate'] },
  { patterns: ['kruzhka', 'chashka'], tags: ['mug'] },
  { patterns: ['stakan', 'bokal'], tags: ['glassware'] },
  { patterns: ['stolovye-pribory'], tags: ['cutlery'] },
  { patterns: ['otkryvalka', 'shtopor'], tags: ['bottle-opener'] },
  { patterns: ['banka-dlya-sypuchikh-produktov'], tags: ['storage-jar'] },
  { patterns: ['sushilka-dlya-posudy'], tags: ['dish-rack'] },
  { patterns: ['polotentse-kukhonnoe', 'polotentse-bannoe'], tags: ['towel'] },
  { patterns: ['podushka'], tags: ['pillow'] },
  { patterns: ['odeyalo', 'pled'], tags: ['blanket'] },
  { patterns: ['postelnoe-bele', 'prostynya', 'navolochka'], tags: ['bedding'] },
  { patterns: ['shtora', 'karniz'], tags: ['curtain'] },
  { patterns: ['kovrik-dlya-vannoy'], tags: ['bath-mat'] },
  { patterns: ['korzina-dlya-belya'], tags: ['laundry-basket'] },
  { patterns: ['divan'], tags: ['sofa'] },
  { patterns: ['kreslo-ofisnoe', 'ofisnoe-kreslo'], tags: ['office-chair'] },
  { patterns: ['kreslo-meshok'], tags: ['bean-bag-chair'] },
  { patterns: ['kreslo'], tags: ['armchair'] },
  { patterns: ['puf'], tags: ['ottoman'] },
  { patterns: ['zhurnalnyy-stolik'], tags: ['coffee-table'] },
  { patterns: ['obedennyy-stol', 'rabochiy-stol', 'stol-kompyuternyy'], tags: ['desk'] },
  { patterns: ['barnyy-stul'], tags: ['bar-stool'] },
  { patterns: ['stul'], tags: ['chair'] },
  { patterns: ['shkaf-kupe', 'knizhnyy-shkaf', 'shkaf'], tags: ['wardrobe'] },
  { patterns: ['komod'], tags: ['dresser'] },
  { patterns: ['tumba-prikrovatnaya', 'televizionnaya-tumba'], tags: ['nightstand'] },
  { patterns: ['krovat-dvuspalnaya', 'krovat-odnospalnaya', 'detskaya-krovatka'], tags: ['bed'] },
  { patterns: ['matras'], tags: ['mattress'] },
  { patterns: ['stellazh', 'polka-nastennaya'], tags: ['bookshelf'] },
  { patterns: ['tualetnyy-stolik'], tags: ['vanity-table'] },
  { patterns: ['zerkalo-napolnoe', 'zerkalo-nastennoe'], tags: ['mirror'] },
  { patterns: ['obuvnitsa'], tags: ['shoe-rack'] },
  { patterns: ['veshalka-napolnaya', 'veshalka-nastennaya'], tags: ['coat-rack'] },
  { patterns: ['lyustra'], tags: ['chandelier'] },
  { patterns: ['bras'], tags: ['wall-sconce'] },
  { patterns: ['torsher'], tags: ['floor-lamp'] },
  { patterns: ['nastolnaya-lampa'], tags: ['table-lamp'] },
  { patterns: ['kovrovaya-dorozhka', 'kover'], tags: ['rug'] },
  { patterns: ['chasy-nastennye'], tags: ['wall-clock'] },
  { patterns: ['vaza'], tags: ['vase'] },
  { patterns: ['fotoramka'], tags: ['photo-frame'] },
  { patterns: ['podsvechnik'], tags: ['candle-holder'] },
  { patterns: ['kashpo'], tags: ['plant-pot'] },
  { patterns: ['aromaticheskiy-diffuzor'], tags: ['reed-diffuser'] },
  { patterns: ['futbolka'], tags: ['t-shirt'] },
  { patterns: ['mayka'], tags: ['tank-top'] },
  { patterns: ['rubashka'], tags: ['shirt'] },
  { patterns: ['bluzka'], tags: ['blouse'] },
  { patterns: ['svitshot'], tags: ['sweatshirt'] },
  { patterns: ['khudi'], tags: ['hoodie'] },
  { patterns: ['sviter'], tags: ['sweater'] },
  { patterns: ['kardigan'], tags: ['cardigan'] },
  { patterns: ['zhaket', 'pidzhak'], tags: ['blazer'] },
  { patterns: ['plate'], tags: ['dress'] },
  { patterns: ['sarafan'], tags: ['sundress'] },
  { patterns: ['yubka'], tags: ['skirt'] },
  { patterns: ['dzhinsy'], tags: ['jeans'] },
  { patterns: ['bryuki', 'dzhoggery'], tags: ['pants'] },
  { patterns: ['leginsy'], tags: ['leggings'] },
  { patterns: ['shorty'], tags: ['shorts'] },
  { patterns: ['kombinezon'], tags: ['jumpsuit'] },
  { patterns: ['trench'], tags: ['trench-coat'] },
  { patterns: ['palto'], tags: ['coat'] },
  { patterns: ['kurtka', 'bomber', 'anorak'], tags: ['jacket'] },
  { patterns: ['pukhovik'], tags: ['puffer-jacket'] },
  { patterns: ['zhilet', 'zhiletka'], tags: ['vest'] },
  { patterns: ['pizhama'], tags: ['pajamas'] },
  { patterns: ['khalat'], tags: ['robe'] },
  { patterns: ['byustgalter'], tags: ['bra'] },
  { patterns: ['kupalnik'], tags: ['swimsuit'] },
  { patterns: ['noski'], tags: ['socks'] },
  { patterns: ['kolgotki'], tags: ['tights'] },
  { patterns: ['shapka', 'beret', 'kepka', 'panama', 'shlyapa'], tags: ['hat'] },
  { patterns: ['sharf', 'palantin'], tags: ['scarf'] },
  { patterns: ['perchatki', 'varezhki'], tags: ['gloves'] },
  { patterns: ['remen'], tags: ['belt'] },
  { patterns: ['tapochki', 'tapochki-dlya-dusha'], tags: ['slippers'] },
  { patterns: ['sportivnyy-kostyum'], tags: ['tracksuit'] },
  { patterns: ['vetrovka', 'dozhdevik'], tags: ['windbreaker'] },
  { patterns: ['fartuk'], tags: ['apron'] },
  { patterns: ['krossovki-begovye', 'begovye-krossovki'], tags: ['running-shoes'] },
  { patterns: ['basketbolnye-krossovki'], tags: ['basketball-shoes'] },
  { patterns: ['krossovki'], tags: ['sneakers'] },
  { patterns: ['kedy'], tags: ['canvas-shoes'] },
  { patterns: ['botinki-trekkingovye', 'trekkingovye-botinki'], tags: ['hiking-boots'] },
  { patterns: ['botinki-rabochie', 'rabochie-botinki'], tags: ['work-boots'] },
  { patterns: ['botinki', 'botilony', 'chelsi', 'bertsy'], tags: ['boots'] },
  { patterns: ['lofery', 'mokasiny', 'oksfordy', 'derbi'], tags: ['loafers'] },
  { patterns: ['tufli-na-kabluke', 'tufli-lodochki'], tags: ['heels'] },
  { patterns: ['baletki'], tags: ['flats'] },
  { patterns: ['sandalii', 'bosonozhki'], tags: ['sandals'] },
  { patterns: ['slantsy', 'vetnamki'], tags: ['flip-flops'] },
  { patterns: ['sapogi', 'uggi', 'valenki'], tags: ['winter-boots'] },
  { patterns: ['espadrili'], tags: ['espadrilles'] },
  { patterns: ['sabo', 'klogi'], tags: ['clogs'] },
  { patterns: ['galoshi'], tags: ['galoshes'] },
  { patterns: ['cheshki', 'puanty'], tags: ['ballet-shoes'] },
  { patterns: ['podguzniki'], tags: ['diapers'] },
  { patterns: ['vlazhnye-salfetki-detskie'], tags: ['baby-wipes'] },
  { patterns: ['detskiy-krem'], tags: ['baby-cream'] },
  { patterns: ['detskiy-shampun'], tags: ['baby-shampoo'] },
  { patterns: ['butylochka-dlya-kormleniya'], tags: ['baby-bottle'] },
  { patterns: ['soska-pustyshka'], tags: ['pacifier'] },
  { patterns: ['sterilizator-butylochek'], tags: ['bottle-sterilizer'] },
  { patterns: ['molokootsos'], tags: ['breast-pump'] },
  { patterns: ['stulchik-dlya-kormleniya'], tags: ['high-chair'] },
  { patterns: ['kolyaska'], tags: ['stroller'] },
  { patterns: ['avtokreslo'], tags: ['car-seat'] },
  { patterns: ['kenguru-perenoska', 'sling'], tags: ['baby-carrier'] },
  { patterns: ['bodi', 'polzunki', 'raspashonka'], tags: ['baby-clothes'] },
  { patterns: ['gorshok'], tags: ['potty-chair'] },
  { patterns: ['vannochka-dlya-kupaniya'], tags: ['baby-bathtub'] },
  { patterns: ['prorezyvatel'], tags: ['teether'] },
  { patterns: ['razvivayuschiy-kovrik'], tags: ['baby-play-mat'] },
  { patterns: ['mobil-na-krovatku'], tags: ['crib-mobile'] },
  { patterns: ['radionyanya', 'videonyanya'], tags: ['baby-monitor'] },
  { patterns: ['nochnik-detskiy'], tags: ['night-light'] },
  { patterns: ['shampun'], tags: ['shampoo'] },
  { patterns: ['konditsioner-dlya-volos'], tags: ['conditioner'] },
  { patterns: ['maska-dlya-volos', 'maska-dlya-litsa'], tags: ['cosmetic-mask'] },
  { patterns: ['maslo-dlya-volos'], tags: ['hair-oil'] },
  { patterns: ['rascheska'], tags: ['hair-brush'] },
  { patterns: ['ployka'], tags: ['curling-iron'] },
  { patterns: ['utyuzhok-dlya-volos'], tags: ['hair-straightener'] },
  { patterns: ['trimmer', 'mashinka-dlya-strizhki'], tags: ['trimmer'] },
  { patterns: ['krem-dlya-litsa', 'krem-dlya-ruk', 'krem-dlya-tela'], tags: ['cream'] },
  { patterns: ['syvorotka-dlya-litsa'], tags: ['serum'] },
  { patterns: ['tonik'], tags: ['facial-toner'] },
  { patterns: ['penka-dlya-umyvaniya'], tags: ['face-cleanser'] },
  { patterns: ['patchi-dlya-glaz'], tags: ['eye-patches'] },
  { patterns: ['tonalnyy-krem'], tags: ['foundation'] },
  { patterns: ['pudra'], tags: ['face-powder'] },
  { patterns: ['rumyana'], tags: ['blush'] },
  { patterns: ['khaylayter'], tags: ['highlighter-makeup'] },
  { patterns: ['tush'], tags: ['mascara'] },
  { patterns: ['podvodka'], tags: ['eyeliner'] },
  { patterns: ['teni-dlya-vek'], tags: ['eyeshadow'] },
  { patterns: ['pomada', 'blesk-dlya-gub', 'balzam-dlya-gub'], tags: ['lipstick'] },
  { patterns: ['lak-dlya-nogtey', 'gel-lak'], tags: ['nail-polish'] },
  { patterns: ['gel-dlya-dusha'], tags: ['body-wash'] },
  { patterns: ['mylo-zhidkoe'], tags: ['liquid-soap'] },
  { patterns: ['dezodorant'], tags: ['deodorant'] },
  { patterns: ['dukhi', 'tualetnaya-voda'], tags: ['perfume'] },
  { patterns: ['britvennyy-stanok'], tags: ['razor'] },
  { patterns: ['zubnaya-schetka', 'elektricheskaya-zubnaya-schetka'], tags: ['toothbrush'] },
  { patterns: ['zubnaya-pasta'], tags: ['toothpaste'] },
  { patterns: ['irrigator'], tags: ['oral-irrigator'] },
  { patterns: ['manikyurnyy-nabor'], tags: ['manicure-kit'] },
  { patterns: ['yoga-kovrik'], tags: ['yoga-mat'] },
  { patterns: ['ganteli'], tags: ['dumbbells'] },
  { patterns: ['giri'], tags: ['kettlebell'] },
  { patterns: ['rezinka-fitnes', 'espander'], tags: ['resistance-bands'] },
  { patterns: ['skakalka'], tags: ['jump-rope'] },
  { patterns: ['obruch'], tags: ['fitness-hoop'] },
  { patterns: ['fitbol'], tags: ['exercise-ball'] },
  { patterns: ['turnik', 'brusya-domashnie'], tags: ['pull-up-bar'] },
  { patterns: ['velotrenazher'], tags: ['exercise-bike'] },
  { patterns: ['begovaya-dorozhka'], tags: ['treadmill'] },
  { patterns: ['ellipticheskiy-trenazher'], tags: ['elliptical-machine'] },
  { patterns: ['massazhnyy-rolik'], tags: ['foam-roller'] },
  { patterns: ['sportivnaya-sumka'], tags: ['gym-bag'] },
  { patterns: ['ryukzak-turisticheskiy'], tags: ['hiking-backpack'] },
  { patterns: ['palatka'], tags: ['camping-tent'] },
  { patterns: ['spalnyy-meshok'], tags: ['sleeping-bag'] },
  { patterns: ['fonar-nalobnyy'], tags: ['headlamp'] },
  { patterns: ['multitul'], tags: ['multitool'] },
  { patterns: ['kompas'], tags: ['compass'] },
  { patterns: ['binokl'], tags: ['binoculars'] },
  { patterns: ['velosiped'], tags: ['bicycle'] },
  { patterns: ['samokat-tryukovoy', 'samokat'], tags: ['scooter'] },
  { patterns: ['roliki'], tags: ['roller-skates'] },
  { patterns: ['skeytbord'], tags: ['skateboard'] },
  { patterns: ['myach-futbolnyy', 'myach-basketbolnyy', 'myach-voleybolnyy'], tags: ['sports-ball'] },
  { patterns: ['raketka-dlya-tennisa', 'raketka-dlya-badmintona'], tags: ['tennis-racket'] },
  { patterns: ['konki'], tags: ['ice-skates'] },
  { patterns: ['lyzhi'], tags: ['skis'] },
  { patterns: ['snoubord'], tags: ['snowboard'] },
  { patterns: ['maska-gornolyzhnaya'], tags: ['ski-goggles'] },
  { patterns: ['shlem-velosipednyy'], tags: ['bike-helmet'] },
  { patterns: ['ochki-dlya-plavaniya'], tags: ['swim-goggles'] },
  { patterns: ['doska-dlya-sap-serfinga'], tags: ['paddle-board'] },
  { patterns: ['avtomobilnyy-kompressor'], tags: ['air-compressor'] },
  { patterns: ['puskovoe-ustroystvo'], tags: ['jump-starter'] },
  { patterns: ['zaryadnoe-ustroystvo-dlya-akkumulyatora'], tags: ['battery-charger'] },
  { patterns: ['akkumulyator'], tags: ['car-battery'] },
  { patterns: ['motornoe-maslo'], tags: ['motor-oil'] },
  { patterns: ['antifriz', 'tormoznaya-zhidkost', 'omyvatel-stekla'], tags: ['car-fluid'] },
  { patterns: ['schetka-stekloochistitelya'], tags: ['wiper-blade'] },
  { patterns: ['chekhly-na-sidenya', 'nakidka-na-sidene'], tags: ['car-seat-cover'] },
  { patterns: ['kovriki-avtomobilnye'], tags: ['car-floor-mats'] },
  { patterns: ['organayzer-v-bagazhnik'], tags: ['trunk-organizer'] },
  { patterns: ['avtomagnitola'], tags: ['car-stereo'] },
  { patterns: ['kolonki-avtomobilnye'], tags: ['car-speakers'] },
  { patterns: ['kamera-zadnego-vida'], tags: ['backup-camera'] },
  { patterns: ['parktronik'], tags: ['parking-sensor'] },
  { patterns: ['avtopylesos'], tags: ['car-vacuum'] },
  { patterns: ['domkrat'], tags: ['car-jack'] },
  { patterns: ['buksirovochnyy-tros'], tags: ['tow-rope'] },
  { patterns: ['aptechka-avtomobilnaya'], tags: ['first-aid-kit'] },
  { patterns: ['ognetushitel'], tags: ['fire-extinguisher'] },
  { patterns: ['kanistra'], tags: ['fuel-can'] },
  { patterns: ['tsepi-protivoskolzheniya'], tags: ['snow-chains'] },
  { patterns: ['polirol-dlya-kuzova'], tags: ['car-polish'] },
  { patterns: ['aromatizator-v-mashinu'], tags: ['car-air-freshener'] },
  { patterns: ['opletka-na-rul', 'chehol-na-rul'], tags: ['steering-wheel-cover'] },
  { patterns: ['deflektory-okon'], tags: ['window-deflectors'] },
  { patterns: ['solntsezaschitnaya-shtorka'], tags: ['car-sunshade'] },
  { patterns: ['perekhodnik-obd2'], tags: ['obd2-scanner'] },
  { patterns: ['provoda-prikurivaniya'], tags: ['jumper-cables'] },
  { patterns: ['termokruzhka-v-podstakannik'], tags: ['travel-mug'] },
  { patterns: ['drely', 'shurupovert'], tags: ['power-drill'] },
  { patterns: ['perforator'], tags: ['rotary-hammer'] },
  { patterns: ['bolgarka'], tags: ['angle-grinder'] },
  { patterns: ['lobzik'], tags: ['jigsaw'] },
  { patterns: ['tsirkulyarnaya-pila', 'tortsovochnaya-pila'], tags: ['circular-saw'] },
  { patterns: ['stroitelnyy-fen'], tags: ['heat-gun'] },
  { patterns: ['kraskopult'], tags: ['paint-sprayer'] },
  { patterns: ['svarochnyy-apparat'], tags: ['welding-machine'] },
  { patterns: ['payalnik'], tags: ['soldering-iron'] },
  { patterns: ['nabor-otvertok', 'nabor-bit'], tags: ['screwdriver-set'] },
  { patterns: ['molotok', 'kiyanka'], tags: ['hammer'] },
  { patterns: ['passatizhi', 'bokorezy'], tags: ['pliers'] },
  { patterns: ['razvodnoy-klyuch', 'gaechnyy-klyuch', 'shestigranniki'], tags: ['wrench-set'] },
  { patterns: ['ruletka'], tags: ['tape-measure'] },
  { patterns: ['lazernyy-uroven', 'uroven'], tags: ['spirit-level'] },
  { patterns: ['nozh-stroitelnyy'], tags: ['utility-knife'] },
  { patterns: ['lestnitsa-stremyanka'], tags: ['step-ladder'] },
  { patterns: ['tachka-stroitelnaya'], tags: ['wheelbarrow'] },
  { patterns: ['shpatel', 'kelma'], tags: ['putty-knife'] },
  { patterns: ['malyarnyy-valik'], tags: ['paint-roller'] },
  { patterns: ['kist-malyarnaya'], tags: ['paint-brush'] },
  { patterns: ['montazhnaya-pena'], tags: ['expanding-foam'] },
  { patterns: ['germetik'], tags: ['sealant'] },
  { patterns: ['cement'], tags: ['cement-bag'] },
  { patterns: ['shpatlevka'], tags: ['wall-putty'] },
  { patterns: ['gruntovka'], tags: ['primer-paint'] },
  { patterns: ['kraska-interernaya', 'lak'], tags: ['paint-can'] },
  { patterns: ['oboi'], tags: ['wallpaper-roll'] },
  { patterns: ['plitka-keramicheskaya'], tags: ['ceramic-tile'] },
  { patterns: ['laminat'], tags: ['laminate-flooring'] },
  { patterns: ['linoleum'], tags: ['linoleum-roll'] },
  { patterns: ['rozetka', 'vyklyuchatel'], tags: ['electrical-outlet'] },
  { patterns: ['udlinitel'], tags: ['extension-cord'] },
  { patterns: ['sukhoy-korm-dlya-sobak', 'vlazhnyy-korm-dlya-sobak'], tags: ['dog-food'] },
  { patterns: ['sukhoy-korm-dlya-koshek', 'vlazhnyy-korm-dlya-koshek'], tags: ['cat-food'] },
  { patterns: ['miska-dlya-korma', 'miska-dlya-vody'], tags: ['pet-bowl'] },
  { patterns: ['poilka-avtomaticheskaya'], tags: ['pet-water-fountain'] },
  { patterns: ['kormushka-avtomaticheskaya'], tags: ['automatic-pet-feeder'] },
  { patterns: ['lezhanka-dlya-sobaki'], tags: ['dog-bed'] },
  { patterns: ['lezhanka-dlya-koshki'], tags: ['cat-bed'] },
  { patterns: ['kogtetochka'], tags: ['scratching-post'] },
  { patterns: ['domik-dlya-koshki'], tags: ['cat-house'] },
  { patterns: ['napolnitel-dlya-lotka', 'lotok-dlya-koshki'], tags: ['cat-litter-box'] },
  { patterns: ['osheynik-ot-kleschey', 'osheynik'], tags: ['pet-collar'] },
  { patterns: ['povodok', 'shleyka'], tags: ['dog-leash'] },
  { patterns: ['namordnik'], tags: ['dog-muzzle'] },
  { patterns: ['perenoska-dlya-zhivotnykh', 'ryukzak-perenoska'], tags: ['pet-carrier'] },
  { patterns: ['akvarium'], tags: ['aquarium'] },
  { patterns: ['filtr-dlya-akvariuma'], tags: ['aquarium-filter'] },
  { patterns: ['terrarium'], tags: ['terrarium'] },
  { patterns: ['schetka-dlya-shersti'], tags: ['pet-grooming-brush'] },
  { patterns: ['kogterez'], tags: ['pet-nail-clipper'] },
  { patterns: ['igrushka-dlya-sobak', 'igrushka-dlya-koshek'], tags: ['pet-toy'] },
  { patterns: ['autogamak-dlya-sobak'], tags: ['dog-car-seat-cover'] },
  { patterns: ['tetrad', 'bloknot'], tags: ['notebook'] },
  { patterns: ['ezhednevnik', 'planer'], tags: ['planner'] },
  { patterns: ['albom-dlya-risovaniya'], tags: ['sketchbook'] },
  { patterns: ['tsvetnaya-bumaga', 'karton'], tags: ['colored-paper'] },
  { patterns: ['ruchka-sharikovaya', 'ruchka-gelevaya'], tags: ['pen'] },
  { patterns: ['karandash', 'karandashi-tsvetnye'], tags: ['pencil'] },
  { patterns: ['flomastery', 'markery-tekstovye'], tags: ['markers'] },
  { patterns: ['lastik'], tags: ['eraser'] },
  { patterns: ['tochilka'], tags: ['pencil-sharpener'] },
  { patterns: ['lineyka', 'ugolnik'], tags: ['ruler'] },
  { patterns: ['tsirkul'], tags: ['drawing-compass'] },
  { patterns: ['kley-karandash', 'kley-pva'], tags: ['glue-stick'] },
  { patterns: ['nozhnitsy'], tags: ['scissors'] },
  { patterns: ['stepler', 'skoby-dlya-steplera'], tags: ['stapler'] },
  { patterns: ['dyrokol'], tags: ['hole-punch'] },
  { patterns: ['skrepki', 'zazhimy-dlya-bumag'], tags: ['paper-clips'] },
  { patterns: ['papka-skorosshivatel', 'fayl-vkladysh'], tags: ['folder'] },
  { patterns: ['konvert'], tags: ['envelope'] },
  { patterns: ['stikery', 'bumaga-dlya-zametok'], tags: ['sticky-notes'] },
  { patterns: ['penal'], tags: ['pencil-case'] },
  { patterns: ['ryukzak-dlya-ucheby', 'ryukzak-shkolnyy'], tags: ['school-backpack'] },
  { patterns: ['uchebnik', 'kniga-khudozhestvennaya', 'kniga-detskaya', 'slovar', 'atlas', 'entsiklopediya'], tags: ['book'] },
  { patterns: ['komiks'], tags: ['comic-book'] },
  { patterns: ['organayzer-nastolnyy', 'podstavka-dlya-ruchek'], tags: ['desk-organizer'] },
  { patterns: ['zakladki-dlya-knig'], tags: ['bookmark'] },
  { patterns: ['konstruktor'], tags: ['building-toy'] },
  { patterns: ['kukla'], tags: ['doll'] },
  { patterns: ['myagkaya-igrushka'], tags: ['plush-toy'] },
  { patterns: ['mashinka-igrushechnaya', 'radioupravlyaemaya-mashina'], tags: ['toy-car'] },
  { patterns: ['zheleznaya-doroga'], tags: ['train-set'] },
  { patterns: ['robot-igrushechnyy'], tags: ['toy-robot'] },
  { patterns: ['kvadrokopter-igrushechnyy'], tags: ['toy-drone'] },
  { patterns: ['pazl'], tags: ['jigsaw-puzzle'] },
  { patterns: ['nastolnaya-igra'], tags: ['board-game'] },
  { patterns: ['shakhmaty', 'shashki', 'domino'], tags: ['board-game'] },
  { patterns: ['molbert-detskiy'], tags: ['easel'] },
  { patterns: ['plastilin'], tags: ['modeling-clay'] },
  { patterns: ['kineticheskiy-pesok'], tags: ['kinetic-sand'] },
  { patterns: ['nabor-dlya-eksperimentov'], tags: ['science-kit'] },
  { patterns: ['teleskop-detskiy'], tags: ['toy-telescope'] },
  { patterns: ['mikroskop-detskiy'], tags: ['toy-microscope'] },
  { patterns: ['muzykalnaya-igrushka', 'ksilofon', 'detskiy-sintezator'], tags: ['kids-musical-toy'] },
  { patterns: ['kukolnyy-domik'], tags: ['dollhouse'] },
  { patterns: ['kukhnya-igrushechnaya'], tags: ['play-kitchen'] },
  { patterns: ['parkovka-igrushechnaya', 'trek-dlya-mashinok'], tags: ['toy-garage'] },
  { patterns: ['almaznaya-mozaika'], tags: ['diamond-painting-kit'] },
  { patterns: ['kartina-po-nomeram'], tags: ['paint-by-numbers'] },
  { patterns: ['fotoalbom'], tags: ['photo-album'] },
  { patterns: ['nabor-dlya-svechevareniya'], tags: ['candle-making-kit'] },
  { patterns: ['nabor-dlya-mylovareniya'], tags: ['soap-making-kit'] },
  { patterns: ['golovolomka'], tags: ['brain-teaser'] },
  { patterns: ['voda-pitevaya'], tags: ['water-bottle'] },
  { patterns: ['sok'], tags: ['juice-carton'] },
  { patterns: ['gazirovannyy-napitok'], tags: ['soft-drink'] },
  { patterns: ['chay-chernyy', 'chay-zelenyy'], tags: ['tea-box'] },
  { patterns: ['kofe-zernovoy', 'kofe-molotyy'], tags: ['coffee-bag'] },
  { patterns: ['moloko', 'kefir', 'yogurt'], tags: ['dairy-product'] },
  { patterns: ['syr-tverdyy', 'tvorog'], tags: ['cheese'] },
  { patterns: ['khleb', 'lavash'], tags: ['bread'] },
  { patterns: ['makarony', 'ris', 'grechka', 'ovsyanka'], tags: ['dry-groceries'] },
  { patterns: ['muka', 'sakhar', 'sol'], tags: ['baking-ingredients'] },
  { patterns: ['maslo-olivkovoe', 'podsolnechnoe-maslo'], tags: ['cooking-oil'] },
  { patterns: ['konservy-rybnye', 'konservy-myasnye', 'tushenka'], tags: ['canned-food'] },
  { patterns: ['kolbasa', 'sosiski', 'kuritsa', 'govyadina', 'svinina'], tags: ['meat-product'] },
  { patterns: ['ryba-zamorozhennaya', 'krevetki'], tags: ['seafood'] },
  { patterns: ['pelmeni', 'vareniki'], tags: ['frozen-food'] },
  { patterns: ['kartofel', 'luk', 'morkov'], tags: ['vegetables'] },
  { patterns: ['yabloki', 'banany', 'apelsiny'], tags: ['fruit'] },
  { patterns: ['orekhi', 'sukhofrukty'], tags: ['nuts'] },
  { patterns: ['pechene', 'shokolad', 'med', 'dzhem'], tags: ['sweet-snacks'] },
  { patterns: ['semena'], tags: ['seed-packet'] },
  { patterns: ['rassada-tsvetov'], tags: ['flower-seedlings'] },
  { patterns: ['sazhenets-yabloni', 'sazhenets-rozy'], tags: ['sapling'] },
  { patterns: ['grunt-universalnyy', 'torf'], tags: ['potting-soil'] },
  { patterns: ['udobrenie'], tags: ['fertilizer-bag'] },
  { patterns: ['leyka'], tags: ['watering-can'] },
  { patterns: ['shlang-polivochnyy'], tags: ['garden-hose'] },
  { patterns: ['sekator', 'sadovye-nozhnitsy'], tags: ['pruning-shears'] },
  { patterns: ['lopata', 'grabli', 'tyapka', 'motyga'], tags: ['garden-tools'] },
  { patterns: ['sadovaya-tachka'], tags: ['garden-wheelbarrow'] },
  { patterns: ['parnik', 'teplitsa'], tags: ['greenhouse'] },
  { patterns: ['gorshok-dlya-rasteniy', 'kashpo-sadovoe'], tags: ['plant-pot'] },
  { patterns: ['gazonnokosilka'], tags: ['lawn-mower'] },
  { patterns: ['trimmer-sadovyy'], tags: ['grass-trimmer'] },
  { patterns: ['kustorez'], tags: ['hedge-trimmer'] },
  { patterns: ['pila-sadovaya', 'topor', 'drovokol'], tags: ['garden-saw'] },
  { patterns: ['mangal'], tags: ['barbecue-grill'] },
  { patterns: ['reshotka-dlya-grilya', 'shampury'], tags: ['grill-tools'] },
  { patterns: ['sadovaya-mebel'], tags: ['patio-furniture'] },
  { patterns: ['zont-sadovyy'], tags: ['patio-umbrella'] },
  { patterns: ['gamak'], tags: ['hammock'] },
  { patterns: ['fonar-sadovyy'], tags: ['garden-lantern'] },
  { patterns: ['skvorechnik', 'kormushka-dlya-ptits'], tags: ['birdhouse'] },
  { patterns: ['koltso-serebryanoe', 'koltso-zolotoe'], tags: ['ring'] },
  { patterns: ['sergi-gvozdiki', 'sergi-koltsa'], tags: ['earrings'] },
  { patterns: ['podveska'], tags: ['pendant'] },
  { patterns: ['tsepochka-na-nogu', 'tsepochka'], tags: ['necklace'] },
  { patterns: ['braslet'], tags: ['bracelet'] },
  { patterns: ['brosh', 'brosh-bulavka'], tags: ['brooch'] },
  { patterns: ['kole'], tags: ['necklace'] },
  { patterns: ['choker'], tags: ['choker-necklace'] },
  { patterns: ['zapanki'], tags: ['cufflinks'] },
  { patterns: ['zazhim-dlya-galstuka'], tags: ['tie-clip'] },
  { patterns: ['solntsezaschitnye-ochki'], tags: ['sunglasses'] },
  { patterns: ['oprava-dlya-ochkov'], tags: ['eyeglasses-frame'] },
  { patterns: ['koshelek', 'portmone'], tags: ['wallet'] },
  { patterns: ['vizitnitsa', 'kartholder'], tags: ['card-holder'] },
  { patterns: ['sumka-shopper'], tags: ['shopper-bag'] },
  { patterns: ['sumka-cherez-plecho'], tags: ['crossbody-bag'] },
  { patterns: ['klatch'], tags: ['clutch-bag'] },
  { patterns: ['ryukzak-gorodskoy'], tags: ['city-backpack'] },
  { patterns: ['chemodan'], tags: ['suitcase'] },
  { patterns: ['dorozhnaya-sumka'], tags: ['duffel-bag'] },
  { patterns: ['zont'], tags: ['umbrella'] },
  { patterns: ['rezinka-dlya-volos'], tags: ['hair-scrunchie'] },
  { patterns: ['zakolka-dlya-volos', 'obodok'], tags: ['hair-accessories'] },
  { patterns: ['kosmetichka'], tags: ['makeup-bag'] },
  { patterns: ['chekhol-dlya-pasporta'], tags: ['passport-cover'] },
  { patterns: ['birka-na-bagazh'], tags: ['luggage-tag'] },
  { patterns: ['klyuchnitsa'], tags: ['key-holder'] },
  { patterns: ['yuvelirnaya-shkatulka'], tags: ['jewelry-box'] },
]

const PRODUCT_TOKEN_RULES: ProductPhotoRule[] = [
  { patterns: ['womens-clothing', 'zhensk'], tags: ['women', 'fashion'] },
  { patterns: ['mens-clothing', 'muzhsk'], tags: ['men', 'fashion'] },
  { patterns: ['footwear'], tags: ['footwear'] },
  { patterns: ['kids', 'detsk'], tags: ['kids'] },
  { patterns: ['beauty', 'ukhod'], tags: ['beauty'] },
  { patterns: ['sports', 'sportiv', 'fitnes'], tags: ['sports'] },
  { patterns: ['auto', 'avto', 'avtomobil'], tags: ['automotive'] },
  { patterns: ['pets', 'zootovary'], tags: ['pet'] },
  { patterns: ['stationery', 'books', 'kants'], tags: ['stationery'] },
  { patterns: ['toys', 'hobbies', 'igrush'], tags: ['toy'] },
  { patterns: ['garden', 'sadov'], tags: ['garden'] },
  { patterns: ['jewelry', 'accessories'], tags: ['accessories'] },
  { patterns: ['home-kitchen', 'kukhon', 'kitchen'], tags: ['kitchen'] },
  { patterns: ['furniture-interior', 'interior'], tags: ['interior'] },
  { patterns: ['electronics-gadgets', 'smartphones-accessories', 'computers-office'], tags: ['electronics'] },
  { patterns: ['food', 'grocery', 'groceries'], tags: ['grocery'] },
  { patterns: ['product'], tags: ['product'] },
]

const collectPhotoRuleTags = (source: string, rules: ProductPhotoRule[]) =>
  rules.flatMap((rule) => (rule.patterns.some((pattern) => source.includes(pattern)) ? rule.tags : []))

const QUERY_CONTEXT_TAGS = new Set([
  'women',
  'men',
  'kids',
  'fashion',
  'kitchen',
  'interior',
  'electronics',
  'automotive',
  'pet',
  'stationery',
  'toy',
  'garden',
  'grocery',
  'sports',
  'accessories',
  'home',
])

const containsKeyword = (value: string, keywords: string[]) => keywords.some((keyword) => value.includes(keyword))

const detectProductVisualKind = (title: string, categoryName?: string): ProductVisualKind => {
  const source = `${title} ${categoryName ?? ''}`.toLowerCase()

  if (containsKeyword(source, ['смартфон', 'телефон', 'iphone', 'android', 'планшет', 'электронная книга'])) {
    return 'phone'
  }
  if (containsKeyword(source, ['ноутбук', 'ультрабук', 'моноблок', 'системный блок', 'монитор', 'телевизор'])) {
    return 'laptop'
  }
  if (containsKeyword(source, ['наушники', 'гарнитура', 'колонка', 'акустика', 'микрофон'])) {
    return 'audio'
  }
  if (containsKeyword(source, ['камера', 'фотоаппарат', 'видеокамера', 'видеорегистратор', 'веб-камера'])) {
    return 'camera'
  }
  if (containsKeyword(source, ['консоль', 'джойстик', 'геймпад', 'vr'])) {
    return 'gaming'
  }
  if (containsKeyword(source, ['роутер', 'wi-fi', 'усилитель', 'адаптер', 'хаб', 'usb', 'ssd', 'флешка'])) {
    return 'router'
  }
  if (containsKeyword(source, ['холодильник', 'стираль', 'микроволнов', 'чайник', 'кофемаш', 'блендер', 'пылесос', 'вентилятор', 'кондиционер'])) {
    return 'appliance'
  }
  if (containsKeyword(source, ['диван', 'кресло', 'пуф', 'стол', 'стул', 'шкаф', 'комод', 'кровать', 'лампа', 'люстра', 'торшер'])) {
    return 'furniture'
  }
  if (containsKeyword(source, ['футболка', 'майка', 'рубашка', 'худи', 'свитер', 'кардиган', 'пиджак', 'брюки', 'джинсы', 'куртка', 'пуховик', 'платье', 'юбка'])) {
    return 'fashion'
  }
  if (containsKeyword(source, ['кроссовки', 'кеды', 'ботинки', 'туфли', 'сапоги', 'лоферы', 'сланцы', 'сандалии', 'босоножки', 'тапочки'])) {
    return 'shoe'
  }
  if (containsKeyword(source, ['шампунь', 'крем', 'сыворотка', 'маска', 'бальзам', 'духи', 'помада', 'лак', 'гель', 'дезодорант', 'паста'])) {
    return 'beauty'
  }
  if (containsKeyword(source, ['сумка', 'рюкзак', 'клатч', 'чемодан', 'кошелек', 'портмоне', 'косметичка'])) {
    return 'bag'
  }
  if (containsKeyword(source, ['книга', 'тетрадь', 'блокнот', 'ежедневник', 'учебник', 'комикс', 'энциклопедия', 'словарь', 'альбом'])) {
    return 'book'
  }
  if (containsKeyword(source, ['вода', 'сок', 'чай', 'кофе', 'молоко', 'йогурт', 'сыр', 'хлеб', 'рис', 'макароны', 'печенье', 'шоколад', 'джем', 'мед'])) {
    return 'food'
  }
  if (containsKeyword(source, ['дрель', 'шуруповерт', 'перфоратор', 'лобзик', 'пила', 'молоток', 'отверт', 'рулетка', 'уровень', 'ключ', 'болгарка'])) {
    return 'tool'
  }
  if (containsKeyword(source, ['корм', 'лежанка', 'когтеточка', 'лоток', 'ошейник', 'поводок', 'переноска', 'аквариум', 'террариум'])) {
    return 'pet'
  }
  if (containsKeyword(source, ['гантели', 'гири', 'йога', 'скакалка', 'палатка', 'велосипед', 'самокат', 'ролики', 'скейт', 'мяч', 'ракетка', 'лыжи', 'сноуборд'])) {
    return 'sports'
  }
  if (containsKeyword(source, ['кольцо', 'серьги', 'подвеска', 'цепочка', 'браслет', 'брошь', 'колье', 'чокер', 'запонки'])) {
    return 'jewelry'
  }
  if (containsKeyword(source, ['подгузники', 'бутылочка', 'соска', 'кроватка', 'коляска', 'автокресло', 'боди', 'ползунки', 'манеж', 'радионяня', 'видеоняня'])) {
    return 'baby'
  }
  if (containsKeyword(source, ['аккумулятор', 'масло', 'компрессор', 'магнитола', 'парктроник', 'домкрат', 'коврики автомобильные', 'видеорегистратор', 'на сиденье'])) {
    return 'auto'
  }

  return 'generic'
}

const normalizePhotoTags = (tags: string[]) =>
  Array.from(
    new Set(
      tags
        .map((tag) => tag.trim().toLowerCase())
        .filter(Boolean),
    ),
  ).slice(0, 6)

const buildProductPhotoSignals = (
  seed: string,
  variant: (typeof PRODUCT_GALLERY_VARIANTS)[number],
  title: string,
  subtitle: string,
  description: string,
  categoryName?: string,
) => {
  const source = `${seed} ${title} ${subtitle} ${description} ${categoryName ?? ''}`.toLowerCase()
  const visualKind = detectProductVisualKind(title, categoryName)
  const phraseTags = collectPhotoRuleTags(source, PRODUCT_PHRASE_RULES)
  const tokenTags = collectPhotoRuleTags(source, PRODUCT_TOKEN_RULES)
  const matchedHints = PRODUCT_QUERY_HINTS.flatMap((hint) => (source.includes(hint.match) ? hint.tags : []))
  const hasSpecificMatch = phraseTags.length > 0

  return {
    visualKind,
    phraseTags: normalizePhotoTags(phraseTags),
    tokenTags: normalizePhotoTags(tokenTags),
    matchedHints: normalizePhotoTags(matchedHints),
    fallbackTags: normalizePhotoTags([
      ...(hasSpecificMatch ? [] : PRODUCT_PHOTO_TAGS[visualKind]),
      ...PRODUCT_VARIANT_TAGS[variant],
    ]),
    allTags: normalizePhotoTags([
    ...phraseTags,
    ...tokenTags,
    ...matchedHints,
    ...(hasSpecificMatch ? [] : PRODUCT_PHOTO_TAGS[visualKind]),
    ...PRODUCT_VARIANT_TAGS[variant],
    ]),
  }
}

const tagToQuery = (value: string) => value.replace(/-/g, ' ').trim()

const normalizeQueryTerms = (terms: string[]) =>
  Array.from(
    new Set(
      terms
        .map((term) => term.trim().toLowerCase())
        .filter(Boolean),
    ),
  )

const createSearchQuery = (...terms: string[]) => normalizeQueryTerms(terms).join(' ')

const buildProductPhotoQueries = (
  seed: string,
  variant: (typeof PRODUCT_GALLERY_VARIANTS)[number],
  title: string,
  subtitle: string,
  description: string,
  categoryName?: string,
) => {
  const signals = buildProductPhotoSignals(seed, variant, title, subtitle, description, categoryName)
  const contextTerms = signals.tokenTags
    .filter((tag) => QUERY_CONTEXT_TAGS.has(tag))
    .slice(0, 2)
    .map(tagToQuery)
  const variantTerms = variant === 'hero' ? ['isolated'] : variant === 'detail' ? ['closeup'] : []

  const primaryTags = signals.phraseTags.length > 0
    ? signals.phraseTags
    : signals.matchedHints.length > 0
      ? signals.matchedHints
      : signals.fallbackTags

  const queries = primaryTags.flatMap((tag) => {
    const query = tagToQuery(tag)
    const combined = contextTerms.length > 0 ? createSearchQuery(...contextTerms, query) : ''
    const variantQueries = variantTerms.flatMap((term) => {
      const combinedWithVariant = combined ? createSearchQuery(combined, term) : ''
      return [combinedWithVariant, createSearchQuery(query, term)]
    })
    return [...variantQueries, combined, query]
  })

  if (queries.length === 0) {
    queries.push(createSearchQuery(...signals.allTags.map(tagToQuery)))
  }

  return Array.from(new Set(queries.filter(Boolean))).slice(0, 6)
}

const createProductPhotoUrl = (
  seed: string,
  variant: (typeof PRODUCT_GALLERY_VARIANTS)[number],
  title: string,
  subtitle: string,
  description: string,
  categoryName?: string,
) => {
  const queries = buildProductPhotoQueries(seed, variant, title, subtitle, description, categoryName)
  const params = new URLSearchParams()
  params.set('seed', `${seed}:${variant}`)
  for (const query of queries) {
    params.append('query', query)
  }
  return `/api/v1/media/product-photo?${params.toString()}`
}

const renderProductIllustration = (kind: ProductVisualKind, palette: Palette, variant: string, seed: string) => {
  const shiftX = (hashString(`${seed}:${variant}:x`) % 36) - 18
  const shiftY = (hashString(`${seed}:${variant}:y`) % 28) - 14
  const scale = variant === 'detail' ? 1.06 : variant === 'lifestyle' ? 0.94 : 1
  const transform = `translate(${shiftX} ${shiftY}) scale(${scale})`

  const snippets: Record<ProductVisualKind, string> = {
    phone: `
      <g transform="${transform}">
        <ellipse cx="602" cy="900" rx="188" ry="42" fill="${palette.panel}" opacity="0.18" />
        <rect x="468" y="256" width="268" height="500" rx="56" fill="${palette.panel}" opacity="0.92" />
        <rect x="486" y="278" width="232" height="456" rx="42" fill="url(#screen)" />
        <rect x="556" y="238" width="92" height="16" rx="8" fill="${palette.textMuted}" opacity="0.55" />
        <circle cx="602" cy="704" r="12" fill="${palette.panel}" opacity="0.42" />
      </g>
    `,
    laptop: `
      <g transform="${transform}">
        <ellipse cx="610" cy="912" rx="248" ry="46" fill="${palette.panel}" opacity="0.18" />
        <rect x="390" y="278" width="440" height="288" rx="28" fill="${palette.panel}" opacity="0.96" />
        <rect x="418" y="304" width="384" height="236" rx="20" fill="url(#screen)" />
        <path d="M328 662H892L826 768H394L328 662Z" fill="${palette.accentSoft}" opacity="0.92" />
        <rect x="548" y="706" width="124" height="18" rx="9" fill="${palette.panel}" opacity="0.18" />
      </g>
    `,
    audio: `
      <g transform="${transform}">
        <ellipse cx="600" cy="912" rx="210" ry="42" fill="${palette.panel}" opacity="0.18" />
        <path d="M472 370C472 278 524 218 600 218C676 218 728 278 728 370" stroke="${palette.accentSoft}" stroke-width="44" stroke-linecap="round" />
        <rect x="430" y="352" width="92" height="214" rx="38" fill="${palette.accentSoft}" opacity="0.95" />
        <rect x="678" y="352" width="92" height="214" rx="38" fill="${palette.accentSoft}" opacity="0.95" />
        <circle cx="476" cy="454" r="42" fill="${palette.panel}" opacity="0.22" />
        <circle cx="724" cy="454" r="42" fill="${palette.panel}" opacity="0.22" />
      </g>
    `,
    camera: `
      <g transform="${transform}">
        <ellipse cx="598" cy="908" rx="208" ry="40" fill="${palette.panel}" opacity="0.18" />
        <rect x="386" y="388" width="424" height="248" rx="42" fill="${palette.accentSoft}" opacity="0.95" />
        <rect x="450" y="350" width="110" height="62" rx="20" fill="${palette.accent}" opacity="0.9" />
        <circle cx="610" cy="512" r="94" fill="${palette.panel}" opacity="0.82" />
        <circle cx="610" cy="512" r="54" fill="${palette.accentSoft}" opacity="0.9" />
        <circle cx="742" cy="454" r="16" fill="${palette.panel}" opacity="0.55" />
      </g>
    `,
    gaming: `
      <g transform="${transform}">
        <ellipse cx="600" cy="914" rx="226" ry="40" fill="${palette.panel}" opacity="0.18" />
        <path d="M394 520C394 430 460 374 548 374H652C740 374 806 430 806 520V582C806 640 768 688 712 704L638 726L600 676L562 726L488 704C432 688 394 640 394 582V520Z" fill="${palette.accentSoft}" opacity="0.96" />
        <circle cx="520" cy="540" r="32" fill="${palette.panel}" opacity="0.22" />
        <circle cx="694" cy="520" r="16" fill="${palette.panel}" opacity="0.22" />
        <circle cx="730" cy="556" r="16" fill="${palette.panel}" opacity="0.22" />
        <rect x="502" y="526" width="36" height="12" rx="6" fill="${palette.panel}" opacity="0.2" />
        <rect x="514" y="514" width="12" height="36" rx="6" fill="${palette.panel}" opacity="0.2" />
      </g>
    `,
    router: `
      <g transform="${transform}">
        <ellipse cx="600" cy="920" rx="198" ry="36" fill="${palette.panel}" opacity="0.18" />
        <rect x="398" y="544" width="404" height="118" rx="30" fill="${palette.accentSoft}" opacity="0.96" />
        <rect x="454" y="492" width="20" height="88" rx="10" fill="${palette.panel}" opacity="0.54" />
        <rect x="570" y="460" width="20" height="120" rx="10" fill="${palette.panel}" opacity="0.54" />
        <rect x="692" y="492" width="20" height="88" rx="10" fill="${palette.panel}" opacity="0.54" />
        <circle cx="476" cy="604" r="10" fill="${palette.panel}" opacity="0.24" />
        <circle cx="514" cy="604" r="10" fill="${palette.panel}" opacity="0.24" />
        <circle cx="552" cy="604" r="10" fill="${palette.panel}" opacity="0.24" />
      </g>
    `,
    appliance: `
      <g transform="${transform}">
        <ellipse cx="600" cy="920" rx="220" ry="40" fill="${palette.panel}" opacity="0.16" />
        <rect x="420" y="274" width="360" height="504" rx="48" fill="${palette.accentSoft}" opacity="0.96" />
        <rect x="460" y="326" width="280" height="320" rx="28" fill="${palette.text}" opacity="0.58" />
        <circle cx="600" cy="486" r="82" fill="${palette.panel}" opacity="0.18" />
        <circle cx="600" cy="486" r="42" fill="${palette.accent}" opacity="0.48" />
        <rect x="478" y="700" width="244" height="18" rx="9" fill="${palette.panel}" opacity="0.16" />
      </g>
    `,
    furniture: `
      <g transform="${transform}">
        <ellipse cx="600" cy="930" rx="236" ry="40" fill="${palette.panel}" opacity="0.16" />
        <rect x="406" y="428" width="388" height="168" rx="44" fill="${palette.accentSoft}" opacity="0.95" />
        <rect x="440" y="314" width="320" height="160" rx="42" fill="${palette.accent}" opacity="0.76" />
        <rect x="458" y="588" width="22" height="168" rx="11" fill="${palette.panel}" opacity="0.34" />
        <rect x="720" y="588" width="22" height="168" rx="11" fill="${palette.panel}" opacity="0.34" />
        <rect x="520" y="588" width="22" height="168" rx="11" fill="${palette.panel}" opacity="0.28" />
        <rect x="658" y="588" width="22" height="168" rx="11" fill="${palette.panel}" opacity="0.28" />
      </g>
    `,
    fashion: `
      <g transform="${transform}">
        <ellipse cx="600" cy="922" rx="182" ry="34" fill="${palette.panel}" opacity="0.16" />
        <path d="M520 296L600 256L680 296L744 418L666 462L638 390V760H562V390L534 462L456 418L520 296Z" fill="${palette.accentSoft}" opacity="0.96" />
        <path d="M562 390H638V760H562V390Z" fill="${palette.accent}" opacity="0.44" />
      </g>
    `,
    shoe: `
      <g transform="${transform}">
        <ellipse cx="600" cy="920" rx="224" ry="34" fill="${palette.panel}" opacity="0.16" />
        <path d="M420 640C470 612 506 560 540 492L658 564C706 594 772 616 816 642V714H420V640Z" fill="${palette.accentSoft}" opacity="0.98" />
        <path d="M438 680H794" stroke="${palette.panel}" stroke-width="16" stroke-linecap="round" opacity="0.18" />
        <path d="M560 560L618 590M530 600L606 628M500 638L594 664" stroke="${palette.panel}" stroke-width="12" stroke-linecap="round" opacity="0.18" />
      </g>
    `,
    beauty: `
      <g transform="${transform}">
        <ellipse cx="600" cy="920" rx="196" ry="36" fill="${palette.panel}" opacity="0.16" />
        <rect x="510" y="280" width="180" height="420" rx="40" fill="${palette.accentSoft}" opacity="0.95" />
        <rect x="544" y="226" width="112" height="82" rx="22" fill="${palette.panel}" opacity="0.24" />
        <rect x="470" y="628" width="116" height="116" rx="28" fill="${palette.text}" opacity="0.6" />
        <rect x="614" y="610" width="124" height="138" rx="34" fill="${palette.accent}" opacity="0.56" />
      </g>
    `,
    bag: `
      <g transform="${transform}">
        <ellipse cx="600" cy="920" rx="206" ry="38" fill="${palette.panel}" opacity="0.16" />
        <rect x="438" y="360" width="324" height="360" rx="48" fill="${palette.accentSoft}" opacity="0.96" />
        <path d="M520 404C520 330 552 286 600 286C648 286 680 330 680 404" stroke="${palette.panel}" stroke-width="24" stroke-linecap="round" opacity="0.32" />
        <rect x="500" y="450" width="200" height="28" rx="14" fill="${palette.panel}" opacity="0.14" />
      </g>
    `,
    book: `
      <g transform="${transform}">
        <ellipse cx="600" cy="924" rx="210" ry="34" fill="${palette.panel}" opacity="0.16" />
        <path d="M408 332C408 304 430 282 458 282H690C714 282 734 302 736 326L756 772C758 800 736 822 708 822H474C446 822 424 802 422 774L408 332Z" fill="${palette.accentSoft}" opacity="0.96" />
        <path d="M724 322L744 770C746 790 730 808 710 808H492L476 296H688C708 296 722 308 724 322Z" fill="${palette.text}" opacity="0.42" />
        <rect x="520" y="346" width="168" height="18" rx="9" fill="${palette.panel}" opacity="0.14" />
        <rect x="520" y="394" width="206" height="18" rx="9" fill="${palette.panel}" opacity="0.12" />
      </g>
    `,
    food: `
      <g transform="${transform}">
        <ellipse cx="600" cy="924" rx="196" ry="34" fill="${palette.panel}" opacity="0.16" />
        <path d="M478 314H722L694 772H506L478 314Z" fill="${palette.accentSoft}" opacity="0.96" />
        <rect x="510" y="364" width="180" height="182" rx="34" fill="${palette.text}" opacity="0.54" />
        <rect x="542" y="596" width="116" height="22" rx="11" fill="${palette.panel}" opacity="0.16" />
      </g>
    `,
    tool: `
      <g transform="${transform}">
        <ellipse cx="600" cy="920" rx="214" ry="36" fill="${palette.panel}" opacity="0.16" />
        <path d="M424 598L608 414C650 372 718 372 760 414L786 440C818 472 818 524 786 556L602 740L424 598Z" fill="${palette.accentSoft}" opacity="0.96" />
        <rect x="370" y="618" width="190" height="80" rx="28" fill="${palette.accent}" opacity="0.58" transform="rotate(-24 370 618)" />
      </g>
    `,
    pet: `
      <g transform="${transform}">
        <ellipse cx="600" cy="922" rx="206" ry="36" fill="${palette.panel}" opacity="0.16" />
        <path d="M456 344H744L710 776H490L456 344Z" fill="${palette.accentSoft}" opacity="0.96" />
        <circle cx="552" cy="500" r="44" fill="${palette.panel}" opacity="0.16" />
        <circle cx="648" cy="500" r="44" fill="${palette.panel}" opacity="0.16" />
        <circle cx="600" cy="438" r="46" fill="${palette.panel}" opacity="0.16" />
        <circle cx="560" cy="540" r="18" fill="${palette.panel}" opacity="0.18" />
        <circle cx="640" cy="540" r="18" fill="${palette.panel}" opacity="0.18" />
        <circle cx="600" cy="578" r="18" fill="${palette.panel}" opacity="0.18" />
      </g>
    `,
    sports: `
      <g transform="${transform}">
        <ellipse cx="600" cy="920" rx="218" ry="36" fill="${palette.panel}" opacity="0.16" />
        <circle cx="510" cy="546" r="98" fill="${palette.accentSoft}" opacity="0.96" />
        <path d="M448 492C480 520 500 562 510 644M572 492C540 520 520 562 510 644M424 550H596" stroke="${palette.panel}" stroke-width="14" stroke-linecap="round" opacity="0.16" />
        <rect x="620" y="418" width="74" height="232" rx="26" fill="${palette.accent}" opacity="0.64" />
        <rect x="708" y="418" width="74" height="232" rx="26" fill="${palette.accent}" opacity="0.64" />
        <rect x="642" y="492" width="118" height="86" rx="26" fill="${palette.accentSoft}" opacity="0.96" />
      </g>
    `,
    jewelry: `
      <g transform="${transform}">
        <ellipse cx="600" cy="924" rx="180" ry="30" fill="${palette.panel}" opacity="0.16" />
        <circle cx="600" cy="548" r="146" stroke="${palette.accentSoft}" stroke-width="42" opacity="0.96" />
        <circle cx="600" cy="548" r="86" fill="${palette.text}" opacity="0.2" />
        <rect x="554" y="338" width="92" height="92" rx="24" fill="${palette.accent}" opacity="0.78" transform="rotate(45 554 338)" />
      </g>
    `,
    baby: `
      <g transform="${transform}">
        <ellipse cx="600" cy="924" rx="216" ry="36" fill="${palette.panel}" opacity="0.16" />
        <path d="M454 666C454 542 536 454 646 454H702C748 454 786 492 786 538V664H454V666Z" fill="${palette.accentSoft}" opacity="0.96" />
        <circle cx="556" cy="772" r="52" fill="${palette.panel}" opacity="0.18" />
        <circle cx="724" cy="772" r="52" fill="${palette.panel}" opacity="0.18" />
        <path d="M704 404C704 338 652 286 584 286" stroke="${palette.panel}" stroke-width="24" stroke-linecap="round" opacity="0.28" />
      </g>
    `,
    auto: `
      <g transform="${transform}">
        <ellipse cx="600" cy="924" rx="232" ry="36" fill="${palette.panel}" opacity="0.16" />
        <path d="M416 624L468 494C484 456 520 430 562 430H676C718 430 754 456 770 494L824 624V694H416V624Z" fill="${palette.accentSoft}" opacity="0.96" />
        <circle cx="500" cy="694" r="52" fill="${palette.panel}" opacity="0.22" />
        <circle cx="740" cy="694" r="52" fill="${palette.panel}" opacity="0.22" />
        <rect x="490" y="500" width="220" height="68" rx="24" fill="${palette.text}" opacity="0.34" />
      </g>
    `,
    generic: `
      <g transform="${transform}">
        <ellipse cx="600" cy="920" rx="210" ry="34" fill="${palette.panel}" opacity="0.16" />
        <rect x="432" y="338" width="336" height="336" rx="42" fill="${palette.accentSoft}" opacity="0.96" />
        <rect x="486" y="392" width="228" height="96" rx="28" fill="${palette.text}" opacity="0.44" />
        <rect x="486" y="518" width="174" height="28" rx="14" fill="${palette.panel}" opacity="0.14" />
        <rect x="486" y="568" width="142" height="28" rx="14" fill="${palette.panel}" opacity="0.12" />
      </g>
    `,
  }

  return snippets[kind]
}

const renderProductSvg = (
  seed: string,
  variant: string,
  title: string,
  subtitle: string,
  badges: string[],
  options: Pick<MediaOptions, 'categoryName' | 'price' | 'currency' | 'stock' | 'sku'>,
) => {
  const palette = getPalette(`${seed}:${variant}`)
  void subtitle
  void badges
  const visualKind = detectProductVisualKind(title, options.categoryName)
  const illustrationMarkup = renderProductIllustration(visualKind, palette, variant, seed)

  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 1200" fill="none">
      <defs>
        <linearGradient id="bg" x1="0" y1="0" x2="1200" y2="1200" gradientUnits="userSpaceOnUse">
          <stop stop-color="${palette.backgroundStart}" />
          <stop offset="1" stop-color="${palette.backgroundEnd}" />
        </linearGradient>
        <linearGradient id="stage" x1="180" y1="150" x2="1020" y2="880" gradientUnits="userSpaceOnUse">
          <stop stop-color="${palette.text}" stop-opacity="0.18" />
          <stop offset="1" stop-color="${palette.accentSoft}" stop-opacity="0.12" />
        </linearGradient>
        <linearGradient id="screen" x1="486" y1="280" x2="718" y2="734" gradientUnits="userSpaceOnUse">
          <stop stop-color="${palette.text}" stop-opacity="0.9" />
          <stop offset="1" stop-color="${palette.accentSoft}" stop-opacity="0.38" />
        </linearGradient>
      </defs>
      <rect width="1200" height="1200" rx="56" fill="url(#bg)" />
      <rect x="78" y="78" width="1044" height="808" rx="58" fill="url(#stage)" />
      ${illustrationMarkup}
    </svg>
  `

  return encodeSvg(svg)
}

const renderSellerLogoSvg = (seed: string, title: string, subtitle: string) => {
  const palette = getPalette(`${seed}:logo`)
  const initials = escapeXml(getInitials(title) || 'MP')
  const label = escapeXml(trimText(subtitle || 'Магазин', 14))
  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 240 240" fill="none">
      <defs>
        <linearGradient id="bg" x1="0" y1="0" x2="240" y2="240" gradientUnits="userSpaceOnUse">
          <stop stop-color="${palette.backgroundStart}" />
          <stop offset="1" stop-color="${palette.backgroundEnd}" />
        </linearGradient>
      </defs>
      <rect width="240" height="240" rx="56" fill="url(#bg)" />
      <circle cx="190" cy="52" r="52" fill="${palette.accent}" opacity="0.24" />
      <circle cx="60" cy="206" r="78" fill="${palette.accentSoft}" opacity="0.12" />
      <rect x="28" y="28" width="184" height="184" rx="42" fill="${palette.text}" opacity="0.08" />
      <text x="120" y="132" text-anchor="middle" fill="${palette.text}" font-family="Manrope, Segoe UI, sans-serif" font-size="76" font-weight="800">${initials}</text>
      <text x="120" y="188" text-anchor="middle" fill="${palette.textMuted}" font-family="Manrope, Segoe UI, sans-serif" font-size="20" font-weight="600">${label}</text>
    </svg>
  `

  return encodeSvg(svg)
}

const renderSellerBannerSvg = (seed: string, title: string, subtitle: string, badges: string[]) => {
  const palette = getPalette(`${seed}:banner`)
  const titleLines = splitLines(title, 26, 2)
  const supportLine = trimText(subtitle || 'Актуальный ассортимент и быстрая доставка по витрине', 60)
  const sellerBadges = normalizeBadges(badges)

  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1440 480" fill="none">
      <defs>
        <linearGradient id="bg" x1="0" y1="0" x2="1440" y2="480" gradientUnits="userSpaceOnUse">
          <stop stop-color="${palette.backgroundStart}" />
          <stop offset="1" stop-color="${palette.backgroundEnd}" />
        </linearGradient>
      </defs>
      <rect width="1440" height="480" rx="42" fill="url(#bg)" />
      <circle cx="1240" cy="90" r="150" fill="${palette.accent}" opacity="0.2" />
      <circle cx="1370" cy="390" r="180" fill="${palette.accentSoft}" opacity="0.14" />
      <path d="M0 356C154 286 334 266 536 320C738 374 935 392 1440 244V480H0V356Z" fill="${palette.panel}" opacity="0.28" />
      <rect x="72" y="70" width="188" height="40" rx="20" fill="${palette.text}" opacity="0.12" />
      <text x="104" y="96" fill="${palette.text}" font-family="Manrope, Segoe UI, sans-serif" font-size="21" font-weight="700">Витрина магазина</text>
      ${titleLines
        .map(
          (line, index) =>
            `<text x="72" y="${178 + index * 72}" fill="${palette.text}" font-family="Manrope, Segoe UI, sans-serif" font-size="62" font-weight="800">${escapeXml(
              line,
            )}</text>`,
        )
        .join('')}
      <text x="72" y="308" fill="${palette.textMuted}" font-family="Manrope, Segoe UI, sans-serif" font-size="26" font-weight="500">${escapeXml(
        supportLine,
      )}</text>
      ${sellerBadges
        .map(
          (badge, index) => `
            <rect x="${72 + index * 226}" y="348" width="208" height="42" rx="21" fill="${palette.text}" opacity="0.14" />
            <text x="${104 + index * 226}" y="375" fill="${palette.text}" font-family="Manrope, Segoe UI, sans-serif" font-size="20" font-weight="700">${escapeXml(
              trimText(badge, 18),
            )}</text>
          `,
        )
        .join('')}
      <rect x="1018" y="84" width="252" height="312" rx="34" fill="${palette.text}" opacity="0.12" />
      <rect x="1060" y="128" width="168" height="18" rx="9" fill="${palette.text}" opacity="0.24" />
      <rect x="1060" y="166" width="130" height="18" rx="9" fill="${palette.text}" opacity="0.16" />
      <rect x="1060" y="214" width="168" height="128" rx="24" fill="${palette.text}" opacity="0.14" />
    </svg>
  `

  return encodeSvg(svg)
}

export const createManagedProductMediaRef = (
  slug: string,
  variant: (typeof PRODUCT_GALLERY_VARIANTS)[number] = 'hero',
) => `${MANAGED_MEDIA_PROTOCOL}product/${slug}/${variant}`

export const createManagedSellerLogoRef = (storeSlug: string) => `${MANAGED_MEDIA_PROTOCOL}seller/${storeSlug}/logo`

export const createManagedSellerBannerRef = (storeSlug: string) => `${MANAGED_MEDIA_PROTOCOL}seller/${storeSlug}/banner`

export const resolveMediaUrl = (value: string | undefined, options: MediaOptions = {}) => {
  const parsed = value ? parseManagedMediaRef(value) : null
  const kind = parsed?.kind ?? options.kind ?? 'product'
  if (value && !shouldUseManagedMedia(value) && !(kind === 'product' && options.renderMode === 'illustration')) {
    return value
  }
  const seed = parsed?.seed ?? options.seed ?? 'marketplace'
  const variant = parsed?.variant ?? 'hero'
  const title = options.title?.trim() || (kind === 'product' ? 'Товар каталога' : 'Магазин продавца')
  const subtitle = options.subtitle?.trim() || ''
  const description = options.description?.trim() || ''
  const badges = normalizeBadges(options.badges ?? [])

  if (kind === 'seller-logo') {
    return renderSellerLogoSvg(seed, title, subtitle || badges[0] || 'Магазин')
  }

  if (kind === 'seller-banner') {
    return renderSellerBannerSvg(seed, title, subtitle, badges)
  }

  if (options.renderMode !== 'illustration') {
    return createProductPhotoUrl(
      seed,
      variant as (typeof PRODUCT_GALLERY_VARIANTS)[number],
      title,
      subtitle,
      description,
      options.categoryName,
    )
  }

  return renderProductSvg(seed, variant, title, subtitle, badges, options)
}

export const resolveProductImage = (
  product: Pick<Product, 'id' | 'slug' | 'title' | 'brand' | 'categoryName' | 'sellerName' | 'imageUrl' | 'images' | 'price' | 'currency' | 'stock' | 'sku'>,
  index = 0,
  source?: string,
) => {
  const fallbackVariant = PRODUCT_GALLERY_VARIANTS[clamp(index, 0, PRODUCT_GALLERY_VARIANTS.length - 1)] ?? 'hero'
  const mediaSource =
    source ??
    product.images[index] ??
    product.imageUrl ??
    createManagedProductMediaRef(product.slug || product.id, fallbackVariant)
  return resolveMediaUrl(mediaSource, {
    kind: 'product',
    seed: product.slug || product.id || `product-${index}`,
    title: product.title,
    subtitle: product.sellerName || product.brand || product.categoryName || 'Товар каталога',
    badges: normalizeBadges([product.categoryName, product.brand, product.sellerName]),
    categoryName: product.categoryName,
    price: product.price,
    currency: product.currency,
    stock: product.stock,
    sku: product.sku,
    renderMode: 'illustration',
  })
}

export const resolveProductImageFallback = (
  product: Pick<Product, 'id' | 'slug' | 'title' | 'brand' | 'categoryName' | 'sellerName' | 'price' | 'currency' | 'stock' | 'sku'>,
  index = 0,
) =>
  resolveMediaUrl(undefined, {
    kind: 'product',
    seed: product.slug || product.id || `product-${index}`,
    title: product.title,
    subtitle: product.sellerName || product.brand || product.categoryName || 'РўРѕРІР°СЂ РєР°С‚Р°Р»РѕРіР°',
    badges: normalizeBadges([product.categoryName, product.brand, product.sellerName]),
    categoryName: product.categoryName,
    price: product.price,
    currency: product.currency,
    stock: product.stock,
    sku: product.sku,
    renderMode: 'illustration',
  })

export const resolveCartItemImage = (
  item: Pick<CartItem, 'id' | 'productId' | 'slug' | 'title' | 'sellerName' | 'sku' | 'imageUrl' | 'price' | 'currency' | 'stock'>,
) =>
  resolveMediaUrl(item.imageUrl, {
    kind: 'product',
    seed: item.slug || item.productId || item.id,
    title: item.title,
    subtitle: item.sellerName || item.sku || 'Товар каталога',
    badges: normalizeBadges([item.sellerName, item.sku]),
    price: item.price,
    currency: item.currency,
    stock: item.stock,
    sku: item.sku,
    renderMode: 'illustration',
  })

export const resolveCartItemImageFallback = (
  item: Pick<CartItem, 'id' | 'productId' | 'slug' | 'title' | 'sellerName' | 'sku' | 'price' | 'currency' | 'stock'>,
) =>
  resolveMediaUrl(undefined, {
    kind: 'product',
    seed: item.slug || item.productId || item.id,
    title: item.title,
    subtitle: item.sellerName || item.sku || 'РўРѕРІР°СЂ РєР°С‚Р°Р»РѕРіР°',
    badges: normalizeBadges([item.sellerName, item.sku]),
    price: item.price,
    currency: item.currency,
    stock: item.stock,
    sku: item.sku,
    renderMode: 'illustration',
  })

export const swapImageToFallback = (image: HTMLImageElement, fallbackSrc: string) => {
  if (!fallbackSrc || image.src === fallbackSrc || image.currentSrc === fallbackSrc) {
    return
  }

  image.src = fallbackSrc
}

export const resolveSellerLogo = (
  profile: Pick<SellerProfile, 'storeSlug' | 'storeName' | 'city' | 'status' | 'logoUrl'>,
) =>
  resolveMediaUrl(profile.logoUrl, {
    kind: 'seller-logo',
    seed: profile.storeSlug || profile.storeName,
    title: profile.storeName,
    subtitle: profile.city || profile.status || 'Магазин',
  })

export const resolveSellerBanner = (
  profile: Pick<SellerProfile, 'storeSlug' | 'storeName' | 'city' | 'status' | 'description' | 'bannerUrl'>,
) =>
  resolveMediaUrl(profile.bannerUrl, {
    kind: 'seller-banner',
    seed: profile.storeSlug || profile.storeName,
    title: profile.storeName,
    subtitle: profile.description || profile.city || 'Витрина продавца',
    badges: normalizeBadges([profile.status, profile.city]),
  })
